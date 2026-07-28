package product

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ddlProduct = `create table if not exists products (
    id bigserial primary key,
    seller_id bigint not null references users(id),
    name varchar(255) not null,
    category varchar(100) not null default '',
    price bigint not null,
    stock bigint not null default 0,
    status varchar(20) not null default 'active'
        check (status in ('active', 'hidden')),
    photo_path text,
    created_at timestamp with time zone not null default now(),
    updated_at timestamp with time zone not null default now()
)`

func RunDDL(ctx context.Context, pool *pgxpool.Pool) error {
	return runDDL(ctx, pool)
}

func runDDL(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, ddlProduct)
	if err != nil {
		return fmt.Errorf("create products: %w", err)
	}
	return nil
}
