package data

import "errors"

const (
	UniqueConstraintViolation = "23505"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
)
