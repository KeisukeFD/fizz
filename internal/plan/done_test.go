package plan

import "testing"

func TestIsDone(t *testing.T) {
	old := Done
	t.Cleanup(func() { Done = old })
	Done = map[string]bool{"P1.1": true, "P2.2": true}

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"single done", "P1.1", true},
		{"single pending", "P9.9", false},
		{"composite all done", "P1.1+P2.2", true},
		{"composite some pending", "P1.1+P9.9", false},
		{"composite none done", "P8.8+P9.9", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDone(tt.id); got != tt.want {
				t.Errorf("IsDone(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}
