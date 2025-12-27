package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vsevolod-ryzhov/gofermart/internal/model"
)

const mockValidOrderID = 4561261212345467

type MockOrdersRepository struct {
	mock.Mock
}

func (m *MockOrdersRepository) GetUserOrders(ctx context.Context, userID int) (*model.UserDisplayOrders, error) {
	args := m.Called(ctx, userID)
	if orders := args.Get(0); orders != nil {
		return orders.(*model.UserDisplayOrders), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrdersRepository) GetOrder(ctx context.Context, orderID int) (*model.UserOrder, error) {
	args := m.Called(ctx, orderID)
	if order := args.Get(0); order != nil {
		return order.(*model.UserOrder), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrdersRepository) AddOrder(ctx context.Context, orderID, userID int) error {
	args := m.Called(ctx, orderID, userID)
	return args.Error(0)
}

func (m *MockOrdersRepository) GetUserBalanceInfo(ctx context.Context, userID int) *model.User {
	args := m.Called(ctx, userID)
	if user := args.Get(0); user != nil {
		return user.(*model.User)
	}
	return nil
}

func (m *MockOrdersRepository) GetPendingOrders(ctx context.Context) (*model.UserOrders, error) {
	args := m.Called(ctx)
	if orders := args.Get(0); orders != nil {
		return orders.(*model.UserOrders), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrdersRepository) UpdateOrderStatus(ctx context.Context, userID, orderID int, status string, accrual float64) error {
	args := m.Called(ctx, userID, orderID, status, accrual)
	return args.Error(0)
}

func (m *MockOrdersRepository) CreateWithdrawal(ctx context.Context, userID, orderID int, sumFloat float64) error {
	args := m.Called(ctx, userID, orderID, sumFloat)
	return args.Error(0)
}

func (m *MockOrdersRepository) GetUserWithdrawals(ctx context.Context, userID int) (*model.Withdrawals, error) {
	args := m.Called(ctx, userID)
	if withdrawals := args.Get(0); withdrawals != nil {
		return withdrawals.(*model.Withdrawals), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestOrdersService_ValidateOrderNumber(t *testing.T) {
	service := NewOrdersService(nil, "")

	tests := []struct {
		name        string
		orderNumber int
		expected    bool
	}{
		{"ValidNumber1", 4561261212345467, true},
		{"ValidNumber2", 79927398713, true},
		{"ValidNumber3", 4242424242424242, true},
		{"ValidNumber4", 4000000000000002, true},
		{"InvalidNumber1", 1234567812345678, false},
		{"InvalidNumber2", 79927398710, false},
		{"InvalidNumber3", -4561261212345467, false},
		{"InvalidNumber4", 4242424242424241, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := service.ValidateOrderNumber(tc.orderNumber)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestOrdersService_AddOrder(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessfulAddOrder", func(t *testing.T) {
		repo := new(MockOrdersRepository)
		service := NewOrdersService(repo, "http://localhost:8080")

		validOrderNumber := mockValidOrderID

		repo.On("GetOrder", ctx, validOrderNumber).Return(nil, sql.ErrNoRows)
		repo.On("AddOrder", ctx, validOrderNumber, mockUserID).Return(nil)

		err := service.AddOrder(ctx, mockUserID, validOrderNumber)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("InvalidOrderNumber", func(t *testing.T) {
		repo := new(MockOrdersRepository)
		service := NewOrdersService(repo, "http://localhost:8080")

		invalidOrderNumber := 1234567812345678

		err := service.AddOrder(ctx, 42, invalidOrderNumber)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNumberInvalid)
		repo.AssertNotCalled(t, "GetOrder")
		repo.AssertNotCalled(t, "AddOrder")
	})

	t.Run("OrderAlreadyExists_SameUser", func(t *testing.T) {
		repo := new(MockOrdersRepository)
		service := NewOrdersService(repo, "http://localhost:8080")

		validOrderNumber := mockValidOrderID
		existingOrder := &model.UserOrder{
			Number: validOrderNumber,
			UserID: mockUserID,
			Status: "NEW",
		}

		repo.On("GetOrder", ctx, validOrderNumber).Return(existingOrder, nil)

		err := service.AddOrder(ctx, mockUserID, validOrderNumber)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrOrderAlreadyExists)
		repo.AssertExpectations(t)
		repo.AssertNotCalled(t, "AddOrder")
	})

	t.Run("OrderAlreadyExists_DifferentUser", func(t *testing.T) {
		repo := new(MockOrdersRepository)
		service := NewOrdersService(repo, "http://localhost:8080")

		validOrderNumber := mockValidOrderID
		existingOrder := &model.UserOrder{
			Number: validOrderNumber,
			UserID: mockUserID + 1,
			Status: "NEW",
		}

		repo.On("GetOrder", ctx, validOrderNumber).Return(existingOrder, nil)

		err := service.AddOrder(ctx, mockUserID, validOrderNumber)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrOrderAddedByAnotherUser)
		repo.AssertExpectations(t)
		repo.AssertNotCalled(t, "AddOrder")
	})

	t.Run("RepositoryErrorOnGetOrder", func(t *testing.T) {
		repo := new(MockOrdersRepository)
		service := NewOrdersService(repo, "http://localhost:8080")

		validOrderNumber := mockValidOrderID
		expectedErr := errors.New("database error")

		repo.On("GetOrder", ctx, validOrderNumber).Return(nil, expectedErr)

		err := service.AddOrder(ctx, mockUserID, validOrderNumber)

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		repo.AssertExpectations(t)
		repo.AssertNotCalled(t, "AddOrder")
	})

	t.Run("NegativeOrderNumber", func(t *testing.T) {
		repo := new(MockOrdersRepository)
		service := NewOrdersService(repo, "http://localhost:8080")

		err := service.AddOrder(ctx, mockUserID, -123456)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNumberInvalid)
		repo.AssertNotCalled(t, "GetOrder")
	})
}
