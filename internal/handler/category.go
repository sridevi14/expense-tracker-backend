package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"expense-tracker/internal/middleware"
	"expense-tracker/internal/response"
	"expense-tracker/internal/service"
)

type CategoryHandler struct {
	categoryService *service.CategoryService
}

func NewCategoryHandler(categoryService *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

func (h *CategoryHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input service.CreateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if input.Name == "" {
		response.ValidationError(c, "name is required")
		return
	}

	cat, err := h.categoryService.Create(c.Request.Context(), userID, input)
	if err != nil {
		response.InternalError(c, "failed to create category")
		return
	}

	response.Success(c, http.StatusCreated, cat)
}

func (h *CategoryHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	categories, err := h.categoryService.List(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "failed to list categories")
		return
	}

	if categories == nil {
		response.Success(c, http.StatusOK, []struct{}{})
		return
	}

	response.Success(c, http.StatusOK, categories)
}
