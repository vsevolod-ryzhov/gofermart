package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vsevolod-ryzhov/gofermart/internal/middleware"
	"github.com/vsevolod-ryzhov/gofermart/internal/model"
	"github.com/vsevolod-ryzhov/gofermart/internal/service"
)

const mockValidOrderID = 4561261212345467

func floatPtr(f float64) *float64 {
	return &f
}

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, login, password string) (*service.RegisterResponse, error) {
	args := m.Called(ctx, login, password)
	if resp := args.Get(0); resp != nil {
		return resp.(*service.RegisterResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, login, password string) (*service.LoginResponse, error) {
	args := m.Called(ctx, login, password)
	if resp := args.Get(0); resp != nil {
		return resp.(*service.LoginResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) ValidateToken(tokenString string) (int, error) {
	args := m.Called(tokenString)
	return args.Int(0), args.Error(1)
}

// Mock для OrdersService (реализует интерфейс)
type MockOrdersService struct {
	mock.Mock
}

func (m *MockOrdersService) GetUserOrders(ctx context.Context, userID int) (*model.UserDisplayOrders, error) {
	args := m.Called(ctx, userID)
	if orders := args.Get(0); orders != nil {
		return orders.(*model.UserDisplayOrders), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrdersService) AddOrder(ctx context.Context, userID, orderID int) error {
	args := m.Called(ctx, userID, orderID)
	return args.Error(0)
}

func (m *MockOrdersService) GetUserBalanceInfo(ctx context.Context, userID int) (*model.User, error) {
	args := m.Called(ctx, userID)
	if balance := args.Get(0); balance != nil {
		return balance.(*model.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrdersService) ApplyWithdrawal(ctx context.Context, userID, orderID int, sum float64) error {
	args := m.Called(ctx, userID, orderID, sum)
	return args.Error(0)
}

func (m *MockOrdersService) GetUserWithdrawals(ctx context.Context, userID int) (*model.Withdrawals, error) {
	args := m.Called(ctx, userID)
	if withdrawals := args.Get(0); withdrawals != nil {
		return withdrawals.(*model.Withdrawals), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrdersService) ValidateOrderNumber(orderNumber int) bool {
	args := m.Called(orderNumber)
	return args.Bool(0)
}

func createTestHandler() (*Handler, *MockAuthService, *MockOrdersService) {
	mockAuth := new(MockAuthService)
	mockOrders := new(MockOrdersService)
	handler := NewHandler(mockAuth, mockOrders)
	return handler, mockAuth, mockOrders
}

func createRequest(method, url string, body interface{}, headers map[string]string) *http.Request {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req := httptest.NewRequest(method, url, bytes.NewBuffer(reqBody))

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req
}

func TestHandler_handleRegister(t *testing.T) {
	t.Run("SuccessfulRegistration", func(t *testing.T) {
		handler, mockAuth, _ := createTestHandler()

		mockAuth.On("Register", mock.Anything, "testuser", "password123").Return(
			&service.RegisterResponse{
				UserID: 1,
				Token:  "jwt-token",
			},
			nil,
		)

		reqBody := model.UserRegisterRequest{
			Login:    "testuser",
			Password: "password123",
		}
		req := createRequest("POST", "/api/user/register", reqBody, map[string]string{
			"Content-Type": "application/json",
		})

		rr := httptest.NewRecorder()
		handler.handleRegister(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "jwt-token")

		cookies := rr.Result().Cookies()
		assert.NotEmpty(t, cookies)

		mockAuth.AssertExpectations(t)
	})

	t.Run("InvalidContentType", func(t *testing.T) {
		handler, mockAuth, _ := createTestHandler()

		req := createRequest("POST", "/api/user/register", nil, map[string]string{
			"Content-Type": "text/plain",
		})

		rr := httptest.NewRecorder()
		handler.handleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockAuth.AssertNotCalled(t, "Register")
	})

	t.Run("UserAlreadyExists", func(t *testing.T) {
		handler, mockAuth, _ := createTestHandler()

		mockAuth.On("Register", mock.Anything, "existing", "pass").Return(
			nil,
			service.ErrUserAlreadyExists,
		)

		reqBody := model.UserRegisterRequest{
			Login:    "existing",
			Password: "pass",
		}
		req := createRequest("POST", "/api/user/register", reqBody, map[string]string{
			"Content-Type": "application/json",
		})

		rr := httptest.NewRecorder()
		handler.handleRegister(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
		mockAuth.AssertExpectations(t)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		handler, mockAuth, _ := createTestHandler()

		req := httptest.NewRequest("POST", "/api/user/register",
			bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.handleRegister(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockAuth.AssertNotCalled(t, "Register")
	})
}

func TestHandler_handleLogin(t *testing.T) {
	t.Run("SuccessfulLogin", func(t *testing.T) {
		handler, mockAuth, _ := createTestHandler()

		mockAuth.On("Login", mock.Anything, "user", "pass").Return(
			&service.LoginResponse{
				UserID: 1,
				Token:  "login-token",
			},
			nil,
		)

		reqBody := model.UserLoginRequest{
			Login:    "user",
			Password: "pass",
		}
		req := createRequest("POST", "/api/user/login", reqBody, map[string]string{
			"Content-Type": "application/json",
		})

		rr := httptest.NewRecorder()
		handler.handleLogin(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "login-token")

		cookies := rr.Result().Cookies()
		assert.NotEmpty(t, cookies)

		mockAuth.AssertExpectations(t)
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		handler, mockAuth, _ := createTestHandler()

		mockAuth.On("Login", mock.Anything, "user", "wrong").Return(
			nil,
			service.ErrInvalidCredentials,
		)

		reqBody := model.UserLoginRequest{
			Login:    "user",
			Password: "wrong",
		}
		req := createRequest("POST", "/api/user/login", reqBody, map[string]string{
			"Content-Type": "application/json",
		})

		rr := httptest.NewRecorder()
		handler.handleLogin(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockAuth.AssertExpectations(t)
	})
}

func TestHandler_handleOrderUpload(t *testing.T) {
	t.Run("SuccessfulOrderUpload", func(t *testing.T) {
		handler, _, mockOrders := createTestHandler()

		req := createRequest("POST", "/api/user/orders", mockValidOrderID, map[string]string{
			"Content-Type": "text/plain",
		})

		ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
		req = req.WithContext(ctx)

		mockOrders.On("AddOrder", mock.Anything, 1, mockValidOrderID).Return(nil)

		rr := httptest.NewRecorder()
		handler.handleOrderUpload(rr, req)

		assert.Equal(t, http.StatusAccepted, rr.Code)
		mockOrders.AssertExpectations(t)
	})

	t.Run("InvalidOrderNumberFormat", func(t *testing.T) {
		handler, _, _ := createTestHandler()

		req := createRequest("POST", "/api/user/orders", "not-a-number", map[string]string{
			"Content-Type": "text/plain",
		})

		ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.handleOrderUpload(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("OrderAlreadyExists", func(t *testing.T) {
		handler, _, mockOrders := createTestHandler()

		req := createRequest("POST", "/api/user/orders", mockValidOrderID, map[string]string{
			"Content-Type": "text/plain",
		})

		ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
		req = req.WithContext(ctx)

		mockOrders.On("AddOrder", mock.Anything, 1, mockValidOrderID).Return(
			service.ErrOrderAlreadyExists,
		)

		rr := httptest.NewRecorder()
		handler.handleOrderUpload(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockOrders.AssertExpectations(t)
	})

	t.Run("OrderAddedByAnotherUser", func(t *testing.T) {
		handler, _, mockOrders := createTestHandler()

		req := createRequest("POST", "/api/user/orders", mockValidOrderID, map[string]string{
			"Content-Type": "application/json",
		})

		ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
		req = req.WithContext(ctx)

		mockOrders.On("AddOrder", mock.Anything, 1, mockValidOrderID).Return(
			service.ErrOrderAddedByAnotherUser,
		)

		rr := httptest.NewRecorder()
		handler.handleOrderUpload(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
		mockOrders.AssertExpectations(t)
	})
}

func TestHandler_handleGetOrders(t *testing.T) {
	t.Run("SuccessfulGetOrders", func(t *testing.T) {
		handler, _, mockOrders := createTestHandler()

		req := httptest.NewRequest("GET", "/api/user/orders", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
		req = req.WithContext(ctx)

		orders := &model.UserDisplayOrders{
			model.UserDisplayOrder{
				Number:  mockValidOrderID,
				UserID:  1,
				Status:  "PROCESSED",
				Accrual: floatPtr(100.5),
			},
		}

		mockOrders.On("GetUserOrders", mock.Anything, 1).Return(orders, nil)

		rr := httptest.NewRecorder()
		handler.handleGetOrders(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), strconv.Itoa(mockValidOrderID))
		mockOrders.AssertExpectations(t)
	})

	t.Run("NoOrders", func(t *testing.T) {
		handler, _, mockOrders := createTestHandler()

		req := httptest.NewRequest("GET", "/api/user/orders", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
		req = req.WithContext(ctx)

		emptyOrders := &model.UserDisplayOrders{}
		mockOrders.On("GetUserOrders", mock.Anything, 1).Return(emptyOrders, nil)

		rr := httptest.NewRecorder()
		handler.handleGetOrders(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
		mockOrders.AssertExpectations(t)
	})

	t.Run("UnauthorizedGetOrders", func(t *testing.T) {
		handler, _, mockOrders := createTestHandler()

		req := httptest.NewRequest("GET", "/api/user/orders", nil)

		rr := httptest.NewRecorder()
		handler.handleGetOrders(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockOrders.AssertNotCalled(t, "GetUserOrders")
	})
}

func TestHandler_handleGetBalance(t *testing.T) {
	t.Run("SuccessfulGetBalance", func(t *testing.T) {
		handler, _, mockOrders := createTestHandler()

		req := httptest.NewRequest("GET", "/api/user/balance", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
		req = req.WithContext(ctx)

		balance := &model.User{
			Balance:   150.75,
			Withdrawn: 50.25,
		}

		mockOrders.On("GetUserBalanceInfo", mock.Anything, 1).Return(balance, nil)

		rr := httptest.NewRecorder()
		handler.handleGetBalance(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "150.75")
		mockOrders.AssertExpectations(t)
	})

	t.Run("UnauthorizedGetBalance", func(t *testing.T) {
		handler, _, mockOrders := createTestHandler()

		req := httptest.NewRequest("GET", "/api/user/balance", nil)

		rr := httptest.NewRecorder()
		handler.handleGetBalance(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockOrders.AssertNotCalled(t, "GetUserBalanceInfo")
	})
}

func TestHandler_handleBalanceWithdraw(t *testing.T) {
	t.Run("SuccessfulWithdrawal", func(t *testing.T) {
		handler, _, mockOrders := createTestHandler()

		reqBody := model.WithdrawRequest{
			Order: strconv.Itoa(mockValidOrderID),
			Sum:   50.0,
		}
		req := createRequest("POST", "/api/user/balance/withdraw", reqBody, map[string]string{
			"Content-Type": "application/json",
		})

		ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
		req = req.WithContext(ctx)

		mockOrders.On("ValidateOrderNumber", mockValidOrderID).Return(true)
		mockOrders.On("GetUserBalanceInfo", mock.Anything, 1).Return(
			&model.User{Balance: 100.0, Withdrawn: 0},
			nil,
		)
		mockOrders.On("ApplyWithdrawal", mock.Anything, 1, mockValidOrderID, 50.0).Return(nil)

		rr := httptest.NewRecorder()
		handler.handleBalanceWithdraw(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockOrders.AssertExpectations(t)
	})

	t.Run("UnauthorizedWithdrawal", func(t *testing.T) {
		handler, _, mockOrders := createTestHandler()

		reqBody := model.WithdrawRequest{
			Order: strconv.Itoa(mockValidOrderID),
			Sum:   50.0,
		}
		req := createRequest("POST", "/api/user/balance/withdraw", reqBody, map[string]string{
			"Content-Type": "application/json",
		})

		rr := httptest.NewRecorder()
		handler.handleBalanceWithdraw(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockOrders.AssertNotCalled(t, "ValidateOrderNumber")
		mockOrders.AssertNotCalled(t, "GetUserBalanceInfo")
		mockOrders.AssertNotCalled(t, "ApplyWithdrawal")
	})
}

func TestHandler_handleWithdrawList(t *testing.T) {
	t.Run("SuccessfulGetWithdrawals", func(t *testing.T) {
		handler, _, mockOrders := createTestHandler()

		req := httptest.NewRequest("GET", "/api/user/withdrawals", nil)
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
		req = req.WithContext(ctx)

		withdrawals := &model.Withdrawals{
			model.Withdrawal{
				OrderNumber: strconv.Itoa(mockValidOrderID),
				Sum:         50.75,
				ProcessedAt: time.Now(),
			},
		}

		mockOrders.On("GetUserWithdrawals", mock.Anything, 1).Return(withdrawals, nil)

		rr := httptest.NewRecorder()
		handler.handleWithdrawList(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), strconv.Itoa(mockValidOrderID))
		mockOrders.AssertExpectations(t)
	})

	t.Run("UnauthorizedGetWithdrawals", func(t *testing.T) {
		handler, _, mockOrders := createTestHandler()

		req := httptest.NewRequest("GET", "/api/user/withdrawals", nil)

		rr := httptest.NewRecorder()
		handler.handleWithdrawList(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockOrders.AssertNotCalled(t, "GetUserWithdrawals")
	})
}
