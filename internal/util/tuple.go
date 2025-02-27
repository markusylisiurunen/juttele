package util

type Tuple[T1 any, T2 any] struct {
	T1 T1
	T2 T2
}

func NewTuple[T1 any, T2 any](t1 T1, t2 T2) Tuple[T1, T2] {
	return Tuple[T1, T2]{T1: t1, T2: t2}
}
