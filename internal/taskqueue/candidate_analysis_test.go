package taskqueue

import (
	"testing"

	"r1rpc/internal/imaging"
)

func TestIsMatchRequiresScoreAndPHashDistance(t *testing.T) {
	policy := matchPolicy{ScoreThreshold: 0.82, MaxPHashDistance: 12}
	tests := []struct {
		name       string
		comparison imaging.Comparison
		want       bool
	}{
		{name: "matched", comparison: imaging.Comparison{Score: 0.9, PHashDistance: 8}, want: true},
		{name: "score too low", comparison: imaging.Comparison{Score: 0.8, PHashDistance: 8}},
		{name: "phash too far", comparison: imaging.Comparison{Score: 0.9, PHashDistance: 13}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isMatch(test.comparison, policy); got != test.want {
				t.Fatalf("isMatch() = %v, want %v", got, test.want)
			}
		})
	}
}
