package service

import (
	"math/rand"
)

type randInt struct {
}

func (s *randInt) NextID() (uint64, error) {
	return rand.Uint64(), nil
}

func NewRandInt() *randInt {
	return &randInt{}
}
