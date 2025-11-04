package domain

import "errors"

var (
	ErrBlocked = errors.New("identifier is blocked")
)

func IsBlockedError(err error) bool {
	return errors.Is(err, ErrBlocked)
}
