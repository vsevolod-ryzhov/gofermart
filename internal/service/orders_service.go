package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/theplant/luhn"
	"github.com/vsevolod-ryzhov/gofermart/internal/model"
)

var (
	ErrNumberInvalid           = errors.New("invalid number")
	ErrOrderAlreadyExists      = errors.New("order already exists")
	ErrOrderAddedByAnotherUser = errors.New("order added by another user")
	ErrUserInfoNotFound        = errors.New("user info not found")

	mutex   sync.Mutex
	rwMutex sync.RWMutex
)

type OrdersRepository interface {
	GetUserOrders(ctx context.Context, userID int) (*model.UserDisplayOrders, error)
	GetOrder(ctx context.Context, orderID int) (*model.UserOrder, error)
	AddOrder(ctx context.Context, orderID, userID int) error
	GetUserBalanceInfo(ctx context.Context, userID int) *model.User
	GetPendingOrders(ctx context.Context) (*model.UserOrders, error)
	UpdateOrderStatus(ctx context.Context, userID, orderID int, status string, accrual float64) error
	CreateWithdrawal(ctx context.Context, userID, orderID int, sumFloat float64) error
	GetUserWithdrawals(ctx context.Context, userID int) (*model.Withdrawals, error)
}

type OrdersService struct {
	repo        OrdersRepository
	accrualPort string
	client      *resty.Client
	worker      *OrderWorker
}

func NewOrdersService(repo OrdersRepository, accrualPort string) *OrdersService {
	service := &OrdersService{
		repo:        repo,
		accrualPort: accrualPort,
		client:      resty.New().SetTimeout(10 * time.Second),
	}

	service.worker = NewOrderWorker(service, 10*time.Second, 1)

	return service
}

func (o *OrdersService) StartWorker(ctx context.Context) {
	o.worker.Start(ctx)
}

func (o *OrdersService) StopWorker() {
	o.worker.Stop()
}

func (o *OrdersService) GetUserOrders(ctx context.Context, userID int) (*model.UserDisplayOrders, error) {
	return o.repo.GetUserOrders(ctx, userID)
}

func (o *OrdersService) AddOrder(ctx context.Context, userID, orderID int) error {
	if !o.ValidateOrderNumber(orderID) {
		return ErrNumberInvalid
	}

	order, err := o.repo.GetOrder(ctx, orderID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if order != nil {
		if order.UserID != userID {
			return ErrOrderAddedByAnotherUser
		}
		return ErrOrderAlreadyExists
	}

	err = o.repo.AddOrder(ctx, orderID, userID)
	if err != nil {
		return err
	}

	return nil
}

func (o *OrdersService) GetUserBalanceInfo(ctx context.Context, userID int) (*model.User, error) {
	rwMutex.Lock()
	user := o.repo.GetUserBalanceInfo(ctx, userID)
	rwMutex.Unlock()
	if user == nil {
		return nil, ErrUserInfoNotFound
	}

	return user, nil
}

func (o *OrdersService) ProcessPendingOrders() error {
	mutex.Lock()
	orders, err := o.repo.GetPendingOrders(context.Background())
	mutex.Unlock()

	if err != nil {
		return err
	}

	if len(*orders) == 0 {
		return nil
	}

	for _, order := range *orders {
		err := o.processOrder(order)
		if err != nil {
			fmt.Println(err)
		}
	}

	return nil
}

func (o *OrdersService) processOrder(order model.UserOrder) error {
	client := resty.New()
	url := fmt.Sprintf("%s/api/orders/%s", o.accrualPort, strconv.Itoa(order.Number))

	resp, err := client.R().Get(url)
	if err != nil {
		return fmt.Errorf("request failed for order %d: %w", order.Number, err)
	}

	respStatus := resp.StatusCode()
	if respStatus != http.StatusOK {
		return nil
	}

	var result struct {
		Order   string  `json:"order"`
		Status  string  `json:"status"`
		Accrual float64 `json:"accrual,omitempty"`
	}

	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		jsonStr := string(resp.Body())
		return fmt.Errorf("failed to parse response for order %d: %w. Resp status %s. Original JSON: %s", order.Number, err, resp.Status(), jsonStr)
	}

	return o.repo.UpdateOrderStatus(context.Background(), order.UserID, order.Number, result.Status, result.Accrual)
}

func (o *OrdersService) ValidateOrderNumber(orderNumber int) bool {
	return luhn.Valid(orderNumber)
}

func (o *OrdersService) ApplyWithdrawal(ctx context.Context, userID, orderID int, sum float64) error {
	return o.repo.CreateWithdrawal(ctx, userID, orderID, sum)
}

func (o *OrdersService) GetUserWithdrawals(ctx context.Context, userID int) (*model.Withdrawals, error) {
	return o.repo.GetUserWithdrawals(ctx, userID)
}
