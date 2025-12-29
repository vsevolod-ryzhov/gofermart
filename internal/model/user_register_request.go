package model

type UserRegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
