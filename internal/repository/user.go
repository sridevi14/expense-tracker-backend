package repository

import (
	"context"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"expense-tracker/internal/model"
)

var ErrNotFound = errors.New("record not found")

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query, args, err := psql.
		Insert("users").
		Columns("email", "name", "password_hash").
		Values(user.Email, user.Name, user.PasswordHash).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()
	if err != nil {
		return err
	}

	return r.pool.QueryRow(ctx, query, args...).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query, args, err := psql.
		Select("id", "email", "name", "password_hash", "created_at", "updated_at").
		From("users").
		Where(squirrel.Eq{"email": email}).
		ToSql()
	if err != nil {
		return nil, err
	}

	var u model.User
	err = r.pool.QueryRow(ctx, query, args...).
		Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}
