package model

type User struct {
	Balance   float64 `json:"balance"`
	Withdrawn float64 `json:"withdrawn"`
}
