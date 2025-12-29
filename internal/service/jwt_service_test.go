package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vsevolod-ryzhov/gofermart/internal/config"
)

const mockSecretKey = "test-secret-key-very-long-and-secure-12345"
const mockTTL = 24 * time.Hour

func createTestConfig() *config.Config {
	return &config.Config{
		JWTSecretKey: mockSecretKey,
		JWTTTL:       mockTTL,
	}
}

func TestJWTService_GenerateToken(t *testing.T) {
	t.Run("SuccessfulTokenGeneration", func(t *testing.T) {
		cfg := createTestConfig()
		service := NewJWTService(cfg)

		tokenString, err := service.GenerateToken(mockUserID)

		require.NoError(t, err)
		require.NotEmpty(t, tokenString)
		assert.Contains(t, tokenString, ".")

		claims, err := service.ParseToken(tokenString)
		require.NoError(t, err)
		assert.Equal(t, mockUserID, claims.UserID)
	})

	t.Run("DifferentUsersGenerateDifferentTokens", func(t *testing.T) {
		cfg := createTestConfig()
		service := NewJWTService(cfg)

		token1, err := service.GenerateToken(1)
		require.NoError(t, err)

		token2, err := service.GenerateToken(2)
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)

		claims1, err := service.ParseToken(token1)
		require.NoError(t, err)
		assert.Equal(t, 1, claims1.UserID)

		claims2, err := service.ParseToken(token2)
		require.NoError(t, err)
		assert.Equal(t, 2, claims2.UserID)
	})

	t.Run("TokenContainsExpiration", func(t *testing.T) {
		cfg := createTestConfig()
		service := NewJWTService(cfg)

		tokenString, err := service.GenerateToken(mockUserID)
		require.NoError(t, err)

		claims, err := service.ParseToken(tokenString)
		require.NoError(t, err)

		require.NotNil(t, claims.ExpiresAt)
		assert.True(t, claims.ExpiresAt.After(time.Now()))
		assert.True(t, claims.ExpiresAt.Before(time.Now().Add(mockTTL+time.Minute))) // С запасом

		require.NotNil(t, claims.IssuedAt)
		assert.True(t, claims.IssuedAt.Before(time.Now().Add(time.Second)))
		assert.True(t, claims.IssuedAt.After(time.Now().Add(-time.Second)))
	})
}

func TestJWTService_ParseToken(t *testing.T) {
	t.Run("SuccessfulTokenParsing", func(t *testing.T) {
		cfg := createTestConfig()
		service := NewJWTService(cfg)

		tokenString, err := service.GenerateToken(mockUserID)
		require.NoError(t, err)

		claims, err := service.ParseToken(tokenString)

		require.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, mockUserID, claims.UserID)
		assert.NotNil(t, claims.ExpiresAt)
		assert.NotNil(t, claims.IssuedAt)
	})

	t.Run("InvalidTokenFormat", func(t *testing.T) {
		cfg := createTestConfig()
		service := NewJWTService(cfg)

		invalidTokens := []string{
			"",                     // empty string
			"invalid",              // no dots
			"invalid.token",        // 2 parts only
			"invalid.token.format", // 3 parts but not JWT
			"header.payload",       // no signature
			"header..signature",    // no payload
			".payload.signature",   // no header
		}

		for _, token := range invalidTokens {
			t.Run(token, func(t *testing.T) {
				claims, err := service.ParseToken(token)
				require.Error(t, err)
				assert.Nil(t, claims)
				assert.Contains(t, err.Error(), "token")
			})
		}
	})

	t.Run("TamperedToken", func(t *testing.T) {
		cfg := createTestConfig()
		service := NewJWTService(cfg)

		tokenString, err := service.GenerateToken(mockUserID)
		require.NoError(t, err)

		tamperedToken := tokenString[:len(tokenString)-5] + "xxxxx"

		claims, err := service.ParseToken(tamperedToken)
		require.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "signature")
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		cfg := &config.Config{
			JWTSecretKey: mockSecretKey,
			JWTTTL:       time.Millisecond, // 1ms expired immediately
		}
		service := NewJWTService(cfg)

		tokenString, err := service.GenerateToken(mockUserID)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		claims, err := service.ParseToken(tokenString)
		require.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "expired")
	})
}
