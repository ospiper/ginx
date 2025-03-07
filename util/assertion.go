package util

import (
	"fmt"
	"reflect"
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
		fmt.Println("not addressable")
		return ret, false
	}
}
