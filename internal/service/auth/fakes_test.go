package auth

import (
	"context"
	"errors"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
)

var errNotStubbed = errors.New("метод репозитория не задан в тесте")

type fakeUserRepo struct {
	createFn     func(ctx context.Context, user *model.User) (*model.User, error)
	getByLoginFn func(ctx context.Context, login string) (*model.User, error)
	getByIDFn    func(ctx context.Context, id int64) (*model.User, error)

	created *model.User
}

func (f *fakeUserRepo) Create(ctx context.Context, user *model.User) (*model.User, error) {
	f.created = user
	if f.createFn == nil {
		return nil, errNotStubbed
	}
	return f.createFn(ctx, user)
}

func (f *fakeUserRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	if f.getByLoginFn == nil {
		return nil, errNotStubbed
	}
	return f.getByLoginFn(ctx, login)
}

func (f *fakeUserRepo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	if f.getByIDFn == nil {
		return nil, errNotStubbed
	}
	return f.getByIDFn(ctx, id)
}

type fakeTokenRepo struct {
	createFn    func(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	getByHashFn func(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	revokeFn    func(ctx context.Context, tokenHash string) error

	createdHashes []string
	revokedHashes []string
}

func (f *fakeTokenRepo) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	f.createdHashes = append(f.createdHashes, tokenHash)
	if f.createFn == nil {
		return nil
	}
	return f.createFn(ctx, userID, tokenHash, expiresAt)
}

func (f *fakeTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	if f.getByHashFn == nil {
		return nil, errNotStubbed
	}
	return f.getByHashFn(ctx, tokenHash)
}

func (f *fakeTokenRepo) Revoke(ctx context.Context, tokenHash string) error {
	f.revokedHashes = append(f.revokedHashes, tokenHash)
	if f.revokeFn == nil {
		return nil
	}
	return f.revokeFn(ctx, tokenHash)
}
