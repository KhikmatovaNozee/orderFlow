package router

import (
	"log/slog"
	"testing"
)

func TestNew(t *testing.T) {
	engine := New(slog.Default(), nil, nil, nil, nil, nil)

	want := []string{
		"GET /health",
		"POST /api/v1/auth/register",
		"POST /api/v1/auth/login",
		"POST /api/v1/auth/refresh",
		"POST /api/v1/auth/logout",
		"GET /api/v1/protected",
		"GET /api/v1/products",
		"GET /api/v1/products/:id",
		"POST /api/v1/orders",
		"GET /api/v1/orders",
		"GET /api/v1/orders/:id",
		"GET /api/v1/manage/test",
	}

	registered := make(map[string]bool)
	for _, route := range engine.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	for _, route := range want {
		if !registered[route] {
			t.Errorf("маршрут %s не зарегистрирован", route)
		}
	}

	if len(engine.Routes()) != len(want) {
		t.Errorf("маршрутов %d, ожидали %d", len(engine.Routes()), len(want))
	}
}
