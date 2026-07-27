package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	query := `insert into users (login, password, role)
values ($1, $2, $3) returning id, login, password, role, created_at`

	err := r.pool.QueryRow(ctx, query, user.Login, user.PasswordHash, user.Role).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.Role, &user.CreatedAt)

	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, model.ErrConflict
		}

		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (r *Repository) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	query := `
		SELECT id, login, password, role, created_at
		FROM users
		WHERE login = $1
	`

	user := &model.User{}

	err := r.pool.QueryRow(ctx, query, login).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}

		return nil, fmt.Errorf("get user by login: %w", err)
	}

	return user, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	query := `
		SELECT id, login, password, role, created_at
		FROM users
		WHERE id = $1
	`

	user := &model.User{}

	err := r.pool.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}

		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}
