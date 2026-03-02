package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"expense-tracker/internal/model"
)

type ExpenseRepository struct {
	pool *pgxpool.Pool
}

func NewExpenseRepository(pool *pgxpool.Pool) *ExpenseRepository {
	return &ExpenseRepository{pool: pool}
}

func (r *ExpenseRepository) Create(ctx context.Context, exp *model.Expense) error {
	query, args, err := psql.
		Insert("expenses").
		Columns("user_id", "category_id", "amount", "description", "expense_date").
		Values(exp.UserID, exp.CategoryID, exp.Amount, exp.Description, exp.ExpenseDate).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()
	if err != nil {
		return err
	}

	return r.pool.QueryRow(ctx, query, args...).
		Scan(&exp.ID, &exp.CreatedAt, &exp.UpdatedAt)
}

func (r *ExpenseRepository) Update(ctx context.Context, exp *model.Expense) error {
	query, args, err := psql.
		Update("expenses").
		Set("category_id", exp.CategoryID).
		Set("amount", exp.Amount).
		Set("description", exp.Description).
		Set("expense_date", exp.ExpenseDate).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": exp.ID, "user_id": exp.UserID}).
		Suffix("RETURNING updated_at").
		ToSql()
	if err != nil {
		return err
	}

	err = r.pool.QueryRow(ctx, query, args...).Scan(&exp.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *ExpenseRepository) Delete(ctx context.Context, id, userID string) error {
	query, args, err := psql.
		Delete("expenses").
		Where(squirrel.Eq{"id": id, "user_id": userID}).
		ToSql()
	if err != nil {
		return err
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ExpenseRepository) List(ctx context.Context, filter model.ExpenseFilter) ([]model.Expense, error) {
	builder := psql.
		Select(
			"e.id", "e.user_id", "e.category_id", "c.name",
			"e.amount", "e.description", "e.expense_date",
			"e.created_at", "e.updated_at",
		).
		From("expenses e").
		Join("categories c ON c.id = e.category_id").
		Where(squirrel.Eq{"e.user_id": filter.UserID}).
		OrderBy("e.expense_date DESC", "e.created_at DESC")

	if filter.CategoryID != "" {
		builder = builder.Where(squirrel.Eq{"e.category_id": filter.CategoryID})
	}
	if filter.StartDate != nil {
		builder = builder.Where(squirrel.GtOrEq{"e.expense_date": *filter.StartDate})
	}
	if filter.EndDate != nil {
		builder = builder.Where(squirrel.LtOrEq{"e.expense_date": *filter.EndDate})
	}
	if filter.Cursor != "" {
		builder = builder.Where(squirrel.Lt{"e.created_at": filter.Cursor})
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	builder = builder.Limit(uint64(limit + 1))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []model.Expense
	for rows.Next() {
		var e model.Expense
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.CategoryID, &e.CategoryName,
			&e.Amount, &e.Description, &e.ExpenseDate,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}

	return expenses, nil
}

func (r *ExpenseRepository) Summary(ctx context.Context, userID string, startDate, endDate *time.Time) (*model.ExpenseSummary, error) {
	whereClause := squirrel.Eq{"e.user_id": userID}

	totalBuilder := psql.
		Select("COALESCE(SUM(e.amount), 0)", "COUNT(*)").
		From("expenses e").
		Where(whereClause)

	catBuilder := psql.
		Select("e.category_id", "c.name", "COALESCE(SUM(e.amount), 0)", "COUNT(*)").
		From("expenses e").
		Join("categories c ON c.id = e.category_id").
		Where(whereClause).
		GroupBy("e.category_id", "c.name").
		OrderBy("SUM(e.amount) DESC")

	dowBuilder := psql.
		Select(
			"TRIM(TO_CHAR(e.expense_date, 'Day'))",
			"COALESCE(SUM(e.amount), 0)",
			"COUNT(*)",
		).
		From("expenses e").
		Where(whereClause).
		GroupBy("TRIM(TO_CHAR(e.expense_date, 'Day'))", "EXTRACT(DOW FROM e.expense_date)").
		OrderBy("EXTRACT(DOW FROM e.expense_date)")

	if startDate != nil {
		totalBuilder = totalBuilder.Where(squirrel.GtOrEq{"e.expense_date": *startDate})
		catBuilder = catBuilder.Where(squirrel.GtOrEq{"e.expense_date": *startDate})
		dowBuilder = dowBuilder.Where(squirrel.GtOrEq{"e.expense_date": *startDate})
	}
	if endDate != nil {
		totalBuilder = totalBuilder.Where(squirrel.LtOrEq{"e.expense_date": *endDate})
		catBuilder = catBuilder.Where(squirrel.LtOrEq{"e.expense_date": *endDate})
		dowBuilder = dowBuilder.Where(squirrel.LtOrEq{"e.expense_date": *endDate})
	}

	summary := &model.ExpenseSummary{}

	tq, ta, err := totalBuilder.ToSql()
	if err != nil {
		return nil, err
	}
	if err := r.pool.QueryRow(ctx, tq, ta...).Scan(&summary.TotalAmount, &summary.Count); err != nil {
		return nil, err
	}

	cq, ca, err := catBuilder.ToSql()
	if err != nil {
		return nil, err
	}
	catRows, err := r.pool.Query(ctx, cq, ca...)
	if err != nil {
		return nil, err
	}
	defer catRows.Close()
	for catRows.Next() {
		var cs model.CategorySummary
		if err := catRows.Scan(&cs.CategoryID, &cs.CategoryName, &cs.TotalAmount, &cs.Count); err != nil {
			return nil, err
		}
		summary.ByCategory = append(summary.ByCategory, cs)
	}

	dq, da, err := dowBuilder.ToSql()
	if err != nil {
		return nil, err
	}
	dowRows, err := r.pool.Query(ctx, dq, da...)
	if err != nil {
		return nil, err
	}
	defer dowRows.Close()
	for dowRows.Next() {
		var ds model.DaySummary
		if err := dowRows.Scan(&ds.DayOfWeek, &ds.TotalAmount, &ds.Count); err != nil {
			return nil, err
		}
		summary.ByDayOfWeek = append(summary.ByDayOfWeek, ds)
	}

	return summary, nil
}
