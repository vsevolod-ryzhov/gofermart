package service

import "github.com/vsevolod-ryzhov/gofermart/internal/repository"

type AuthService struct {
	repo *repository.PostgresRepository
}

func NewAuthService(repo *repository.PostgresRepository) *AuthService {
	return &AuthService{
		repo: repo,
	}
}
