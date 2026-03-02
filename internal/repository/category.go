package repository

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"expense-tracker/internal/model"
)

type CategoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

func (r *CategoryRepository) Create(ctx context.Context, cat *model.Category) error {
	query, args, err := psql.
		Insert("categories").
		Columns("user_id", "name", "icon").
		Values(cat.UserID, cat.Name, cat.Icon).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return err
	}

	return r.pool.QueryRow(ctx, query, args...).Scan(&cat.ID, &cat.CreatedAt)
}

func (r *CategoryRepository) ListByUser(ctx context.Context, userID string) ([]model.Category, error) {
	query, args, err := psql.
		Select("id", "user_id", "name", "icon", "created_at").
		From("categories").
		Where(squirrel.Eq{"user_id": userID}).
		OrderBy("name ASC").
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []model.Category
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Icon, &c.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, nil
}

func (r *CategoryRepository) CreateDefaults(ctx context.Context, userID string) error {
	defaults := []struct{ name, icon string }{
		{"Food", "utensils"},
		{"Transport", "car"},
		{"Shopping", "shopping-bag"},
		{"Bills", "file-text"},
		{"Entertainment", "film"},
		{"Health", "heart"},
		{"Other", "more-horizontal"},
	}

	builder := psql.Insert("categories").Columns("user_id", "name", "icon")
	for _, d := range defaults {
		builder = builder.Values(userID, d.name, d.icon)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, query, args...)
	return err
}
