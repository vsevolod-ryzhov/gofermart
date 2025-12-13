package model

import (
	"time"
)

type UserOrder struct {
	Number    int       `json:"number,string"`
	UserID    int       `json:"user_id"`
	Status    string    `json:"status"`
	Accrual   *int      `json:"accrual,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserOrders []UserOrder
