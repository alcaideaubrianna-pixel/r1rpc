package taskqueue

import (
	"testing"

	"r1rpc/internal/imaging"
)

func TestIsMatchRequiresAllConfiguredThresholds(t *testing.T) {
	policy := defaultMatchPolicy()
	tests := []struct {
		name       string
		comparison imaging.Comparison
		want       bool
	}{
		{name: "matched", comparison: imaging.Comparison{Score: 0.9, PHashDistance: 8, DHashDistance: 10, AHashDistance: 11}, want: true},
		{name: "score too low", comparison: imaging.Comparison{Score: 0.8, PHashDistance: 8, DHashDistance: 10, AHashDistance: 11}},
		{name: "phash too far", comparison: imaging.Comparison{Score: 0.9, PHashDistance: 13, DHashDistance: 10, AHashDistance: 11}},
		{name: "dhash too far", comparison: imaging.Comparison{Score: 0.9, PHashDistance: 8, DHashDistance: 17, AHashDistance: 11}},
		{name: "ahash too far", comparison: imaging.Comparison{Score: 0.9, PHashDistance: 8, DHashDistance: 10, AHashDistance: 17}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isMatch(test.comparison, policy); got != test.want {
				t.Fatalf("isMatch() = %v, want %v", got, test.want)
			}
		})
	}
}
