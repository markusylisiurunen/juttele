package juttele

import "github.com/markusylisiurunen/juttele/internal/util"

type Tuple[T1 any, T2 any] = util.Tuple[T1, T2]

func NewTuple[T1 any, T2 any](t1 T1, t2 T2) Tuple[T1, T2] {
	return util.NewTuple(t1, t2)
}
