package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func() context.Context
		expectedID   int
		expectedOk   bool
	}{
		{
			name: "ValidUserIDInContext",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), UserIDKey, 123)
			},
			expectedID: 123,
			expectedOk: true,
		},
		{
			name: "NoUserIDInContext",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedID: 0,
			expectedOk: false,
		},
		{
			name: "WrongTypeInContext",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), UserIDKey, "not-an-int")
			},
			expectedID: 0,
			expectedOk: false,
		},
		{
			name: "NilValueInContext",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), UserIDKey, nil)
			},
			expectedID: 0,
			expectedOk: false,
		},
		{
			name: "DifferentContextKey",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), "differentKey", 456)
			},
			expectedID: 0,
			expectedOk: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req = req.WithContext(tc.setupContext())

			userID, ok := GetUserID(req)

			if ok != tc.expectedOk {
				t.Errorf("Expected ok=%v, got %v", tc.expectedOk, ok)
			}

			if userID != tc.expectedID {
				t.Errorf("Expected user ID %d, got %d", tc.expectedID, userID)
			}
		})
	}
}
