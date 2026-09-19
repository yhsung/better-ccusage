// Package errs defines sentinel errors used across the apps/better-ccusage internal
// packages. Use errors.Is to branch on these.
package errs

import "errors"

var (
	ErrNoData           = errors.New("no usage data found")
	ErrUnknownModel     = errors.New("unknown model")
	ErrInvalidJSON      = errors.New("invalid JSONL entry")
	ErrConfigNotFound   = errors.New("config file not found")
	ErrIncompatibleMode = errors.New("cost mode incompatible with data")
)
