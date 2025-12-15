package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/theplant/luhn"
	"github.com/vsevolod-ryzhov/gofermart/internal/model"
	"github.com/vsevolod-ryzhov/gofermart/internal/repository"
)

var (
	ErrNumberInvalid           = errors.New("invalid number")
	ErrOrderAlreadyExists      = errors.New("order already exists")
	ErrOrderAddedByAnotherUser = errors.New("order added by another user")
	ErrUserInfoNotFound        = errors.New("user info not found")
)

type OrdersService struct {
	repo *repository.PostgresRepository
}

func NewOrdersService(repo *repository.PostgresRepository) *OrdersService {
	return &OrdersService{repo}
}

func (o *OrdersService) GetUserOrders(ctx context.Context, userID int) (*model.UserOrders, error) {
	return o.repo.GetUserOrders(ctx, userID)
}

func (o *OrdersService) AddOrder(ctx context.Context, userID, orderID int) error {
	if !validateOrderNumber(orderID) {
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
	user := o.repo.GetUserBalanceInfo(ctx, userID)
	if user == nil {
		return nil, ErrUserInfoNotFound
	}

	return user, nil
}

func validateOrderNumber(orderNumber int) bool {
	return luhn.Valid(orderNumber)
}
