package service

import (
	"context"
	"database/sql"
	"errors"
	"regexp"

	"github.com/vsevolod-ryzhov/gofermart/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrValidationFailed   = errors.New("validation failed")
)

type AuthRepository interface {
	UserExists(ctx context.Context, login string) (bool, error)
	CreateUser(ctx context.Context, login, password string) (int, error)
	GetUserByLogin(ctx context.Context, login string) (*repository.User, error)
}

type TokenService interface {
	GenerateToken(userID int) (string, error)
	ParseToken(tokenString string) (*Claims, error)
}

type AuthService struct {
	repo       AuthRepository
	jwtService TokenService
}

func NewAuthService(repo AuthRepository, jwtService TokenService) *AuthService {
	return &AuthService{
		repo:       repo,
		jwtService: jwtService,
	}
}

type RegisterResponse struct {
	UserID int
	Token  string
}

func (s *AuthService) Register(ctx context.Context, login, password string) (*RegisterResponse, error) {
	if err := validateCredentials(login, password); err != nil {
		return nil, ErrValidationFailed
	}

	exists, err := s.repo.UserExists(ctx, login)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	userID, err := s.repo.CreateUser(ctx, login, string(hashedPassword))
	if err != nil {
		return nil, err
	}

	token, err := s.jwtService.GenerateToken(userID)
	if err != nil {
		return nil, err
	}

	return &RegisterResponse{
		UserID: userID,
		Token:  token,
	}, nil
}

type LoginResponse struct {
	UserID int
	Token  string
}

func (s *AuthService) Login(ctx context.Context, login, password string) (*LoginResponse, error) {
	if err := validateCredentials(login, password); err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.jwtService.GenerateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		UserID: user.ID,
		Token:  token,
	}, nil
}

func (s *AuthService) ValidateToken(tokenString string) (int, error) {
	claims, err := s.jwtService.ParseToken(tokenString)
	if err != nil {
		return 0, err
	}

	return claims.UserID, nil
}

func validateCredentials(login, password string) error {
	if len(login) < 3 || len(login) > 255 {
		return errors.New("login must be between 3 and 255 characters")
	}

	if len(password) < 4 || len(password) > 64 {
		return errors.New("password must be between 4 and 64 characters")
	}

	if matched, _ := regexp.MatchString("^[a-zA-Z0-9._-]+$", login); !matched {
		return errors.New("login can only contain letters, numbers, dots, underscores and hyphens")
	}

	return nil
}
