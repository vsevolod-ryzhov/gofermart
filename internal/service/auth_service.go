package service

import (
	"context"
	"errors"

	"github.com/vsevolod-ryzhov/gofermart/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
)

type AuthService struct {
	repo *repository.PostgresRepository
}

func NewAuthService(repo *repository.PostgresRepository) *AuthService {
	return &AuthService{
		repo: repo,
	}
}

func (s *AuthService) Register(ctx context.Context, login, password string) error {
	exists, err := s.repo.UserExists(ctx, login)
	if err != nil {
		return err
	}
	if exists {
		return ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.repo.CreateUser(ctx, login, string(hashedPassword))
}
