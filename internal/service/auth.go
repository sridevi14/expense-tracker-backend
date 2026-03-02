package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"expense-tracker/internal/model"
	"expense-tracker/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
)

type AuthService struct {
	userRepo     *repository.UserRepository
	categoryRepo *repository.CategoryRepository
	jwtSecret    []byte
}

func NewAuthService(userRepo *repository.UserRepository, categoryRepo *repository.CategoryRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		categoryRepo: categoryRepo,
		jwtSecret:    []byte(jwtSecret),
	}
}

type RegisterInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResponse, error) {
	existing, _ := s.userRepo.FindByEmail(ctx, input.Email)
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:        input.Email,
		Name:         input.Name,
		PasswordHash: string(hash),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	_ = s.categoryRepo.CreateDefaults(ctx, user.ID)

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: user}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: user}, nil
}

func (s *AuthService) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
