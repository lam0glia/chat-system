package domain

type UIDGenerator interface {
	NextID() (uint64, error)
}
