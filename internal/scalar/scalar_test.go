package scalar

import (
	"net"
	"reflect"
	"testing"
	"time"
)

func TestIs(t *testing.T) {
	tests := []struct {
		v           any
		scalar, val bool
	}{
		{"", true, true},
		{false, true, true},
		{int8(0), true, true},
		{uint64(0), true, true},
		{float32(0), true, true},
		{time.Time{}, true, true}, // TextUnmarshaler
		{net.IP{}, true, true},    // TextUnmarshaler slice
		{[]int{}, false, true},
		{[][]int{}, false, false},
		{struct{}{}, false, false},
		{map[string]int{}, false, false},
		{complex64(0), false, false},
		{new(int), false, false},
	}
	for _, tt := range tests {
		typ := reflect.TypeOf(tt.v)
		if got := Is(typ); got != tt.scalar {
			t.Errorf("Is(%s) = %v, want %v", typ, got, tt.scalar)
		}
		if got := IsValue(typ); got != tt.val {
			t.Errorf("IsValue(%s) = %v, want %v", typ, got, tt.val)
		}
	}
}
