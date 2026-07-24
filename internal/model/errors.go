package model

import "errors"

var (
	ErrNotFound  = errors.New("entity not found")
	ErrInvalid   = errors.New("invalid input")
	ErrForbidden = errors.New("access forbidden")
	ErrNoStock   = errors.New("insufficient stock")
	ErrConflict  = errors.New("entity already exists")
)
