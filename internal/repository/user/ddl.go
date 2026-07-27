package user

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ddlUser = `
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    login VARCHAR(255) NOT NULL UNIQUE,
    password TEXT NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
)
`

func RunDDL(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, ddlUser)
	if err != nil {
		return fmt.Errorf("create users: %w", err)
	}

	return nil
}
