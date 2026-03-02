package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"expense-tracker/internal/model"
)

type BudgetRepository struct {
	pool *pgxpool.Pool
}

func NewBudgetRepository(pool *pgxpool.Pool) *BudgetRepository {
	return &BudgetRepository{pool: pool}
}

func (r *BudgetRepository) Upsert(ctx context.Context, budget *model.Budget) error {
	query := `
		INSERT INTO budgets (user_id, month, amount)
		VALUES ($1, DATE_TRUNC('month', $2::date), $3)
		ON CONFLICT (user_id, month) DO UPDATE
		  SET amount = EXCLUDED.amount, updated_at = NOW()
		RETURNING id, month, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query, budget.UserID, budget.Month, budget.Amount).
		Scan(&budget.ID, &budget.Month, &budget.CreatedAt, &budget.UpdatedAt)
}

func (r *BudgetRepository) GetByMonth(ctx context.Context, userID, month string) (*model.Budget, error) {
	query := `
		SELECT id, user_id, month, amount, created_at, updated_at
		FROM budgets
		WHERE user_id = $1 AND month = DATE_TRUNC('month', $2::date)
	`
	var b model.Budget
	err := r.pool.QueryRow(ctx, query, userID, month+"-01").
		Scan(&b.ID, &b.UserID, &b.Month, &b.Amount, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}
