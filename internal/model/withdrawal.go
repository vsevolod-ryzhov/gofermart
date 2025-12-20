package model

import "time"

type Withdrawal struct {
	UserID      int
	OrderNumber string    `json:"order,omitempty"`
	Sum         float64   `json:"sum,omitempty"`
	ProcessedAt time.Time `json:"processed_at"`
}

type Withdrawals []Withdrawal
