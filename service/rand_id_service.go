package service

import (
	"context"
	"math/rand"
)

type randInt struct {
}

func (s *randInt) NewUID(ctx context.Context) (int, error) {
	return rand.Int(), nil
}

func NewRandInt() *randInt {
	return &randInt{}
}
