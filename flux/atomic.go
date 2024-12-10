package flux

import (
	"sync/atomic"
)

type AtomicValue[T any] struct {
	value *atomic.Value
}

func NewAtomicValue[T any](value T) *AtomicValue[T] {
	v := new(atomic.Value)
	v.Store(value)

	return &AtomicValue[T]{
		value: v,
	}
}

//nolint:ireturn
func (v *AtomicValue[T]) Get() (T, bool) {
	var zero T

	value := v.value.Load()
	if value == nil {
		return zero, false
	}

	typed, ok := value.(T)
	if !ok {
		return zero, false
	}

	return typed, true
}

func (v *AtomicValue[T]) Set(value T) {
	v.value.Store(value)
}
