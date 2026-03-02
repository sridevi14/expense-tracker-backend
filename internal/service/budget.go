package service

import (
	"context"
	"errors"
	"time"

	"expense-tracker/internal/model"
	"expense-tracker/internal/repository"
)

type BudgetService struct {
	budgetRepo *repository.BudgetRepository
}

func NewBudgetService(budgetRepo *repository.BudgetRepository) *BudgetService {
	return &BudgetService{budgetRepo: budgetRepo}
}

type UpsertBudgetInput struct {
	Month  string  `json:"month"`
	Amount float64 `json:"amount"`
}

func (s *BudgetService) Upsert(ctx context.Context, userID string, input UpsertBudgetInput) (*model.Budget, error) {
	t, err := time.Parse("2006-01", input.Month)
	if err != nil {
		return nil, errors.New("invalid month format, use YYYY-MM")
	}

	budget := &model.Budget{
		UserID: userID,
		Month:  t,
		Amount: input.Amount,
	}

	if err := s.budgetRepo.Upsert(ctx, budget); err != nil {
		return nil, err
	}

	return budget, nil
}

func (s *BudgetService) GetByMonth(ctx context.Context, userID, month string) (*model.Budget, error) {
	return s.budgetRepo.GetByMonth(ctx, userID, month)
}
