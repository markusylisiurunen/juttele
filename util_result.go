package juttele

import "github.com/markusylisiurunen/juttele/internal/util"

type Result[T any] = util.Result[T]

func Ok[T any](value T) Result[T]    { return util.Ok(value) }
func Err[T any](err error) Result[T] { return util.Err[T](err) }
