package refreshtoken

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ddlRefreshToken = `
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE
)
`

func RunDDL(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, ddlRefreshToken)
	if err != nil {
		return fmt.Errorf("create refresh_tokens: %w", err)
	}

	return nil
}
