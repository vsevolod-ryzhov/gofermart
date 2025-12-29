package service

import (
	"context"
	"database/sql"
	"strings"

	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vsevolod-ryzhov/gofermart/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

const mockUserID = 123
const mockGeneratedToken = "generated-token"

type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) UserExists(ctx context.Context, login string) (bool, error) {
	args := m.Called(ctx, login)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthRepository) CreateUser(ctx context.Context, login, password string) (int, error) {
	args := m.Called(ctx, login, password)
	return args.Int(0), args.Error(1)
}

func (m *MockAuthRepository) GetUserByLogin(ctx context.Context, login string) (*repository.User, error) {
	args := m.Called(ctx, login)
	if user := args.Get(0); user != nil {
		return user.(*repository.User), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateToken(userID int) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) ParseToken(tokenString string) (*Claims, error) {
	args := m.Called(tokenString)
	if claims := args.Get(0); claims != nil {
		return claims.(*Claims), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestValidateCredentials(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		password string
		wantErr  bool
		errText  string
	}{
		{
			name:     "ValidCredentials",
			login:    "testuser",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "ValidCredentialsWithSpecialChars",
			login:    "test.user-name_123",
			password: "pass1234",
			wantErr:  false,
		},
		{
			name:     "MinLengthLogin",
			login:    "usr",
			password: "pass",
			wantErr:  false,
		},
		{
			name:     "MaxLengthLogin",
			login:    strings.Repeat("S", 255),
			password: "password",
			wantErr:  false,
		},
		{
			name:     "LoginTooShort",
			login:    "ab",
			password: "password",
			wantErr:  true,
			errText:  "login must be between 3 and 255 characters",
		},
		{
			name:     "LoginTooLong",
			login:    string(make([]byte, 256)),
			password: "password",
			wantErr:  true,
			errText:  "login must be between 3 and 255 characters",
		},
		{
			name:     "LoginEmpty",
			login:    "",
			password: "password",
			wantErr:  true,
			errText:  "login must be between 3 and 255 characters",
		},
		{
			name:     "LoginWithInvalidChars",
			login:    "user@test",
			password: "password",
			wantErr:  true,
			errText:  "login can only contain letters, numbers, dots, underscores and hyphens",
		},
		{
			name:     "LoginWithSpaces",
			login:    "user test",
			password: "password",
			wantErr:  true,
			errText:  "login can only contain letters, numbers, dots, underscores and hyphens",
		},
		{
			name:     "LoginWithCyrillic",
			login:    "пользователь",
			password: "password",
			wantErr:  true,
			errText:  "login can only contain letters, numbers, dots, underscores and hyphens",
		},
		{
			name:     "PasswordTooShort",
			login:    "testuser",
			password: "pas",
			wantErr:  true,
			errText:  "password must be between 4 and 64 characters",
		},
		{
			name:     "PasswordTooLong",
			login:    "testuser",
			password: string(make([]byte, 65)),
			wantErr:  true,
			errText:  "password must be between 4 and 64 characters",
		},
		{
			name:     "PasswordEmpty",
			login:    "testuser",
			password: "",
			wantErr:  true,
			errText:  "password must be between 4 and 64 characters",
		},
		{
			name:     "PasswordMaxLength",
			login:    "testuser",
			password: string(make([]byte, 64)),
			wantErr:  false,
		},
		{
			name:     "BothInvalid",
			login:    "ab",
			password: "pas",
			wantErr:  true,
			errText:  "login must be between 3 and 255 characters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCredentials(tc.login, tc.password)

			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if err.Error() != tc.errText {
					t.Errorf("Expected error %q, got %q", tc.errText, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessfulRegistration", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		repo.On("UserExists", ctx, "newuser").Return(false, nil)
		repo.On("CreateUser", ctx, "newuser", mock.AnythingOfType("string")).
			Run(func(args mock.Arguments) {
				// Проверяем что пароль хэширован
				password := args.Get(2).(string)
				err := bcrypt.CompareHashAndPassword([]byte(password), []byte("password123"))
				assert.NoError(t, err, "Password should be properly hashed")
			}).
			Return(mockUserID, nil)

		tokenService.On("GenerateToken", mockUserID).Return(mockGeneratedToken, nil)

		response, err := authService.Register(ctx, "newuser", "password123")

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, mockUserID, response.UserID)
		assert.Equal(t, mockGeneratedToken, response.Token)

		repo.AssertExpectations(t)
		tokenService.AssertExpectations(t)
	})

	t.Run("UserAlreadyExists", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		repo.On("UserExists", ctx, "existinguser").Return(true, nil)

		response, err := authService.Register(ctx, "existinguser", "password123")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.ErrorIs(t, err, ErrUserAlreadyExists)

		repo.AssertExpectations(t)
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		response, err := authService.Register(ctx, "ab", "password123")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.ErrorIs(t, err, ErrValidationFailed)

		repo.AssertNotCalled(t, "UserExists")
	})

	t.Run("RepositoryErrorOnUserExists", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		expectedErr := errors.New("database error")
		repo.On("UserExists", ctx, "testuser").Return(false, expectedErr)

		response, err := authService.Register(ctx, "testuser", "password123")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, expectedErr, err)

		repo.AssertExpectations(t)
	})

	t.Run("RepositoryErrorOnCreateUser", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		expectedErr := errors.New("create user error")
		repo.On("UserExists", ctx, "testuser").Return(false, nil)
		repo.On("CreateUser", ctx, "testuser", mock.AnythingOfType("string")).
			Return(0, expectedErr)

		response, err := authService.Register(ctx, "testuser", "password123")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, expectedErr, err)

		repo.AssertExpectations(t)
	})

	t.Run("TokenGenerationError", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		expectedErr := errors.New("token generation error")
		repo.On("UserExists", ctx, "testuser").Return(false, nil)
		repo.On("CreateUser", ctx, "testuser", mock.AnythingOfType("string")).
			Return(mockUserID, nil)
		tokenService.On("GenerateToken", mockUserID).Return("", expectedErr)

		response, err := authService.Register(ctx, "testuser", "password123")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, expectedErr, err)

		repo.AssertExpectations(t)
		tokenService.AssertExpectations(t)
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		repo.On("UserExists", cancelledCtx, "testuser").
			Return(false, context.Canceled)

		response, err := authService.Register(cancelledCtx, "testuser", "password123")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.ErrorIs(t, err, context.Canceled)

		repo.AssertExpectations(t)
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	t.Run("ValidToken", func(t *testing.T) {
		tokenService := new(MockTokenService)
		authService := NewAuthService(nil, tokenService)

		tokenService.On("ParseToken", "valid-token").Return(
			&Claims{UserID: 123},
			nil,
		)

		userID, err := authService.ValidateToken("valid-token")

		require.NoError(t, err)
		assert.Equal(t, 123, userID)

		tokenService.AssertExpectations(t)
	})

	t.Run("InvalidToken", func(t *testing.T) {
		tokenService := new(MockTokenService)
		authService := NewAuthService(nil, tokenService)

		expectedErr := errors.New("token is invalid")
		tokenService.On("ParseToken", "invalid-token").Return(nil, expectedErr)

		userID, err := authService.ValidateToken("invalid-token")

		require.Error(t, err)
		assert.Equal(t, 0, userID)
		assert.Equal(t, expectedErr, err)

		tokenService.AssertExpectations(t)
	})

	t.Run("EmptyToken", func(t *testing.T) {
		tokenService := new(MockTokenService)
		authService := NewAuthService(nil, tokenService)

		expectedErr := errors.New("token is empty")
		tokenService.On("ParseToken", "").Return(nil, expectedErr)

		userID, err := authService.ValidateToken("")

		require.Error(t, err)
		assert.Equal(t, 0, userID)
		assert.Equal(t, expectedErr, err)

		tokenService.AssertExpectations(t)
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		tokenService := new(MockTokenService)
		authService := NewAuthService(nil, tokenService)

		expectedErr := errors.New("token has expired")
		tokenService.On("ParseToken", "expired-token").Return(nil, expectedErr)

		userID, err := authService.ValidateToken("expired-token")

		require.Error(t, err)
		assert.Equal(t, 0, userID)
		assert.Equal(t, expectedErr, err)

		tokenService.AssertExpectations(t)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessfulLogin", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		require.NoError(t, err)

		repo.On("GetUserByLogin", ctx, "testuser").Return(
			&repository.User{
				ID:       mockUserID,
				Login:    "testuser",
				Password: string(hashedPassword),
			},
			nil,
		)

		tokenService.On("GenerateToken", mockUserID).Return("login-token", nil)

		response, err := authService.Login(ctx, "testuser", "password123")

		require.NoError(t, err)
		require.NotNil(t, response)

		assert.IsType(t, &LoginResponse{}, response)
		assert.Equal(t, mockUserID, response.UserID)
		assert.Equal(t, "login-token", response.Token)

		repo.AssertExpectations(t)
		tokenService.AssertExpectations(t)
	})

	t.Run("UserNotFound", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		repo.On("GetUserByLogin", ctx, "nonexistent").Return(
			nil,
			sql.ErrNoRows,
		)

		response, err := authService.Login(ctx, "nonexistent", "password123")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.ErrorIs(t, err, ErrInvalidCredentials)

		repo.AssertExpectations(t)
	})

	t.Run("InvalidPassword", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
		require.NoError(t, err)

		repo.On("GetUserByLogin", ctx, "testuser").Return(
			&repository.User{
				ID:       mockUserID,
				Login:    "testuser",
				Password: string(hashedPassword),
			},
			nil,
		)

		response, err := authService.Login(ctx, "testuser", "wrongpassword")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.ErrorIs(t, err, ErrInvalidCredentials)

		repo.AssertExpectations(t)
	})

	t.Run("RepositoryError", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		expectedErr := errors.New("database error")
		repo.On("GetUserByLogin", ctx, "testuser").Return(nil, expectedErr)

		response, err := authService.Login(ctx, "testuser", "password123")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, expectedErr, err)

		repo.AssertExpectations(t)
	})

	t.Run("CorruptedPasswordHash", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		repo.On("GetUserByLogin", ctx, "testuser").Return(
			&repository.User{
				ID:       mockUserID,
				Login:    "testuser",
				Password: "not-a-valid-bcrypt-hash",
			},
			nil,
		)

		response, err := authService.Login(ctx, "testuser", "password123")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.ErrorIs(t, err, ErrInvalidCredentials)

		repo.AssertExpectations(t)
	})

	t.Run("TokenGenerationError", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		require.NoError(t, err)

		expectedErr := errors.New("token error")
		repo.On("GetUserByLogin", ctx, "testuser").Return(
			&repository.User{
				ID:       mockUserID,
				Login:    "testuser",
				Password: string(hashedPassword),
			},
			nil,
		)
		tokenService.On("GenerateToken", mockUserID).Return("", expectedErr)

		response, err := authService.Login(ctx, "testuser", "password123")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, expectedErr, err)

		repo.AssertExpectations(t)
		tokenService.AssertExpectations(t)
	})

	t.Run("CaseSensitiveLogin", func(t *testing.T) {
		repo := new(MockAuthRepository)
		tokenService := new(MockTokenService)
		authService := NewAuthService(repo, tokenService)

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		require.NoError(t, err)

		repo.On("GetUserByLogin", ctx, "TestUser").Return(nil, sql.ErrNoRows)
		repo.On("GetUserByLogin", ctx, "testuser").Return(
			&repository.User{
				ID:       mockUserID,
				Login:    "testuser",
				Password: string(hashedPassword),
			},
			nil,
		)
		tokenService.On("GenerateToken", mockUserID).Return("token", nil)

		_, err = authService.Login(ctx, "TestUser", "password123")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCredentials)

		response, err := authService.Login(ctx, "testuser", "password123")
		require.NoError(t, err)
		assert.NotNil(t, response)

		repo.AssertExpectations(t)
		tokenService.AssertExpectations(t)
	})
}
