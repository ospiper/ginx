package rest

import (
	"reflect"
	"testing"
)

func Test_explode(t *testing.T) {
	type args struct {
		v any
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "single string",
			args: args{
				v: "hello world",
			},
			want: []string{"hello", "world"},
		},
		{
			name: "multiple strings",
			args: args{
				v: []string{"hello 1", "world  2"},
			},
			want: []string{"hello", "1", "world", "2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := explode(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("explode() = %v, want %v", got, tt.want)
			}
		})
	}
}
