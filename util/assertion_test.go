package util

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

type testElemStruct struct {
}

func (testElemStruct) Ping() string { return "pong" }

type testPointerStruct struct {
}

func (*testPointerStruct) Ping() string { return "pong" }

type testInterface interface {
	Ping() string
}

func TestAs(t *testing.T) {
	type testCase[T any] struct {
		name string
		v    any
		can  bool
	}
	tests := []testCase[testInterface]{
		{
			name: "elem",
			v:    testElemStruct{},
			can:  true,
		},
		{
			name: "elem pointer",
			v:    &testElemStruct{},
			can:  true,
		},
		{
			name: "unaddressable",
			v:    testPointerStruct{},
			can:  false,
		},
		{
			name: "pointer",
			v:    &testPointerStruct{},
			can:  true,
		},
		{
			name: "not assertable",
			v:    "",
			can:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, can := As[testInterface](tt.v)
			assert.Equal(t, tt.can, can)
			if can {
				assert.Equal(t, "pong", got.Ping())
			}
		})
	}
}
