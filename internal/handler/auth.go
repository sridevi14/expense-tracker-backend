package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"expense-tracker/internal/response"
	"expense-tracker/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input service.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if input.Email == "" || input.Password == "" || input.Name == "" {
		response.ValidationError(c, "email, name, and password are required")
		return
	}

	if len(input.Password) < 6 {
		response.ValidationError(c, "password must be at least 6 characters")
		return
	}

	result, err := h.authService.Register(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			response.Error(c, http.StatusConflict, "EMAIL_TAKEN", "email already registered")
			return
		}
		response.InternalError(c, "failed to register user")
		return
	}

	response.Success(c, http.StatusCreated, result)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input service.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if input.Email == "" || input.Password == "" {
		response.ValidationError(c, "email and password are required")
		return
	}

	result, err := h.authService.Login(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			response.Unauthorized(c, "invalid email or password")
			return
		}
		response.InternalError(c, "failed to login")
		return
	}

	response.Success(c, http.StatusOK, result)
}
