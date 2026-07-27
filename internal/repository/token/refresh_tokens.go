package refreshtoken

import (
	"context"
	"fmt"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

func (r *Repository) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (
			user_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
	`

	_, err := r.pool.Exec(ctx, query, userID, tokenHash, expiresAt)

	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *Repository) GetByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	token := &model.RefreshToken{}

	err := r.pool.QueryRow(
		ctx,
		query,
		tokenHash,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return token, nil
}

func (r *Repository) Revoke(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1
		WHERE token_hash = $2
		AND revoked_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query, time.Now(), tokenHash)

	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}
