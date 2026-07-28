package order

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ddlOrder = `create table if not exists orders(
    id bigserial primary key,
    user_id bigint not null references users(id),
    status varchar(50) not null default 'new'
        check (status in ('new', 'paid', 'shipped', 'cancelled')),
    total bigint not null default 0,
    created_at timestamp with time zone not null default now()
)`

const ddlOrderItem = `create table if not exists order_items(
    id bigserial primary key,
    order_id bigint not null references orders(id),
    product_id bigint not null references products(id),
    quantity bigint not null,
    price bigint not null
)`

func RunDDL(ctx context.Context, pool *pgxpool.Pool) error {
	return runDdl(ctx, pool)
}

func runDdl(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, ddlOrder)
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	_, err = pool.Exec(ctx, ddlOrderItem)
	if err != nil {
		return fmt.Errorf("create order_item: %w", err)
	}
	return nil
}
