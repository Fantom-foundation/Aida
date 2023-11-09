package utils

import (
	"golang.org/x/exp/constraints"
)

func Min[T constraints.Ordered](s, t T) T {
	if s < t {
		return s
	}
	return t
}

func Max[T constraints.Ordered](s, t T) T {
	if s > t {
		return s
	}
	return t
}
