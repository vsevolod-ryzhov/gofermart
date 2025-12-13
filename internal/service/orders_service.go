package service

import (
	"context"

	"github.com/vsevolod-ryzhov/gofermart/internal/model"
	"github.com/vsevolod-ryzhov/gofermart/internal/repository"
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
