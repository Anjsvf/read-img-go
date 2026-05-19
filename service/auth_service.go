package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Anjsvf/read-img-go/config"
	"github.com/Anjsvf/read-img-go/domain"
	"github.com/Anjsvf/read-img-go/repository"
)

var (
	ErrEmailAlreadyExists = fmt.Errorf("EMAIL_ALREADY_EXISTS")
	ErrInvalidCredentials = fmt.Errorf("INVALID_CREDENTIALS")
)

type AuthService interface {
	Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error)
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error)
}

type authSvc struct {
	userRepo  repository.UserRepository
	jwtSecret string
}

func NewAuthService(userRepo repository.UserRepository, cfg *config.Config) AuthService {
	return &authSvc{
		userRepo:  userRepo,
		jwtSecret: cfg.JWTSecret,
	}
}

func (s *authSvc) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error) {
	existing, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		CustomerCode: uuid.NewString(), // cada usuário tem seu próprio customer_code
		Name:         req.Name,
		Email:        req.Email,
		Password:     string(hashed),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.generateToken(user)
}

func (s *authSvc) Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.generateToken(user)
}

func (s *authSvc) generateToken(user *domain.User) (*domain.AuthResponse, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":           user.CustomerCode,
		"name":          user.Name,
		"email":         user.Email,
		"customer_code": user.CustomerCode,
		"exp":           time.Now().Add(24 * time.Hour).Unix(),
		"iat":           time.Now().Unix(),
	})

	signed, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}

	return &domain.AuthResponse{
		Token:        signed,
		ExpiresIn:    "24h",
		CustomerCode: user.CustomerCode,
		Name:         user.Name,
	}, nil
}
