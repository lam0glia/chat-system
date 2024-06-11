package domain

import "context"

type UIDGenerator interface {
	NewUID(ctx context.Context) (int, error)
}
