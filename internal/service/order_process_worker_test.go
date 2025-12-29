package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vsevolod-ryzhov/gofermart/internal/model"
)

type MockOrdersService struct {
	mock.Mock
}

func (m *MockOrdersService) ProcessPendingOrders(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
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
	if user := args.Get(0); user != nil {
		return user.(*model.User), args.Error(1)
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

func TestNewOrderWorker(t *testing.T) {
	t.Run("CreateWorkerWithValidParameters", func(t *testing.T) {
		mockService := new(MockOrdersService)
		interval := 10 * time.Second
		workers := 3

		worker := NewOrderWorker(mockService, interval, workers)

		require.NotNil(t, worker)
		assert.Equal(t, mockService, worker.ordersService)
		assert.Equal(t, interval, worker.interval)
		assert.Equal(t, workers, worker.workers)
		assert.False(t, worker.IsRunning())
	})

	t.Run("CreateWorkerWithZeroWorkers", func(t *testing.T) {
		mockService := new(MockOrdersService)
		interval := 10 * time.Second

		worker := NewOrderWorker(mockService, interval, 0)

		require.NotNil(t, worker)
		assert.Equal(t, 0, worker.workers)
		assert.False(t, worker.IsRunning())
	})

	t.Run("CreateWorkerWithZeroInterval", func(t *testing.T) {
		mockService := new(MockOrdersService)

		worker := NewOrderWorker(mockService, 0, 3)

		require.NotNil(t, worker)
		assert.Equal(t, time.Duration(0), worker.interval)
	})

	t.Run("CreateWorkerWithNegativeInterval", func(t *testing.T) {
		mockService := new(MockOrdersService)

		worker := NewOrderWorker(mockService, -10*time.Second, 3)

		require.NotNil(t, worker)
		assert.Equal(t, -10*time.Second, worker.interval)
	})

	t.Run("CreateWorkerWithNegativeWorkers", func(t *testing.T) {
		mockService := new(MockOrdersService)

		worker := NewOrderWorker(mockService, 10*time.Second, -3)

		require.NotNil(t, worker)
		assert.Equal(t, -3, worker.workers)
	})
}

func TestOrderWorker_Start(t *testing.T) {
	t.Run("StartWorkerSuccessfully", func(t *testing.T) {
		mockService := new(MockOrdersService)
		worker := NewOrderWorker(mockService, 100*time.Millisecond, 2)

		assert.False(t, worker.IsRunning())

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		worker.Start(ctx)

		time.Sleep(50 * time.Millisecond)

		assert.True(t, worker.IsRunning())

		worker.Stop()
		assert.False(t, worker.IsRunning())
	})

	t.Run("MultipleStartCalls", func(t *testing.T) {
		mockService := new(MockOrdersService)
		worker := NewOrderWorker(mockService, 100*time.Millisecond, 1)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		worker.Start(ctx)
		assert.True(t, worker.IsRunning())

		worker.Start(ctx)
		assert.True(t, worker.IsRunning())

		worker.Stop()
		assert.False(t, worker.IsRunning())
	})

	t.Run("StartWithZeroWorkers", func(t *testing.T) {
		mockService := new(MockOrdersService)
		worker := NewOrderWorker(mockService, 100*time.Millisecond, 0)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		worker.Start(ctx)

		time.Sleep(50 * time.Millisecond)

		assert.True(t, worker.IsRunning())

		worker.Stop()
		assert.False(t, worker.IsRunning())
	})
}

func TestOrderWorker_Stop(t *testing.T) {
	t.Run("StopRunningWorker", func(t *testing.T) {
		mockService := new(MockOrdersService)
		worker := NewOrderWorker(mockService, 100*time.Millisecond, 2)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		worker.Start(ctx)
		time.Sleep(50 * time.Millisecond)
		assert.True(t, worker.IsRunning())

		worker.Stop()
		assert.False(t, worker.IsRunning())
	})

	t.Run("MultipleStopCalls", func(t *testing.T) {
		mockService := new(MockOrdersService)
		worker := NewOrderWorker(mockService, 100*time.Millisecond, 1)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		worker.Start(ctx)
		time.Sleep(50 * time.Millisecond)

		worker.Stop()
		assert.False(t, worker.IsRunning())

		worker.Stop()
		assert.False(t, worker.IsRunning())
	})

	t.Run("StopNotStartedWorker", func(t *testing.T) {
		mockService := new(MockOrdersService)
		worker := NewOrderWorker(mockService, 100*time.Millisecond, 1)

		assert.NotPanics(t, func() {
			worker.Stop()
		})
		assert.False(t, worker.IsRunning())
	})

	t.Run("StopAfterContextCancellation", func(t *testing.T) {
		mockService := new(MockOrdersService)
		worker := NewOrderWorker(mockService, 100*time.Millisecond, 1)

		ctx, cancel := context.WithCancel(context.Background())
		worker.Start(ctx)

		cancel()
		time.Sleep(50 * time.Millisecond)

		worker.Stop()
		assert.False(t, worker.IsRunning())
	})
}
