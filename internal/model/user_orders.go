package model

import "time"

type UserOrder struct {
	Number    string    `json:"number"`
	Status    string    `json:"status"`
	Accrual   string    `json:"accrual"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserOrders []UserOrder
