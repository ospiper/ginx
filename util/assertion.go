package util

import (
	"reflect"

	"github.com/ospiper/ginx/dbx"
)

func As[T any](v any) (T, bool) {
	ret, ok := v.(T)
	if ok {
		return ret, true
	}
	vl := reflect.ValueOf(v)
	if vl.Kind() == reflect.Ptr {
		return As[T](vl.Elem().Interface())
	} else if vl.CanAddr() {
		return As[T](vl.Addr().Interface())
	} else {
		return ret, false
	}
}

func AsIDList[T dbx.WithID](ts []T) []int64 {
	ret := make([]int64, len(ts))
	for i, t := range ts {
		ret[i] = t.GetID()
	}
	return ret
}
