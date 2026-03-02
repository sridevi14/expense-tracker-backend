package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"expense-tracker/internal/middleware"
	"expense-tracker/internal/repository"
	"expense-tracker/internal/response"
	"expense-tracker/internal/service"
)

type BudgetHandler struct {
	budgetService *service.BudgetService
}

func NewBudgetHandler(budgetService *service.BudgetService) *BudgetHandler {
	return &BudgetHandler{budgetService: budgetService}
}

func (h *BudgetHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	month := c.Query("month")

	if month == "" {
		response.ValidationError(c, "month query param is required (format: YYYY-MM)")
		return
	}

	budget, err := h.budgetService.GetByMonth(c.Request.Context(), userID, month)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Success(c, http.StatusOK, nil)
			return
		}
		response.InternalError(c, "failed to get budget")
		return
	}

	response.Success(c, http.StatusOK, budget)
}

func (h *BudgetHandler) Upsert(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input service.UpsertBudgetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if input.Month == "" {
		response.ValidationError(c, "month is required (format: YYYY-MM)")
		return
	}

	if input.Amount <= 0 {
		response.ValidationError(c, "amount must be a positive number")
		return
	}

	budget, err := h.budgetService.Upsert(c.Request.Context(), userID, input)
	if err != nil {
		if err.Error() == "invalid month format, use YYYY-MM" {
			response.ValidationError(c, err.Error())
			return
		}
		response.InternalError(c, "failed to save budget")
		return
	}

	response.Success(c, http.StatusOK, budget)
}
