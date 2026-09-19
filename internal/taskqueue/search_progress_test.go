package taskqueue

import "testing"

func TestSearchGroupStatus(t *testing.T) {
	tests := []struct {
		name   string
		counts map[string]int
		want   string
	}{
		{name: "queued", counts: map[string]int{"queued": 2}, want: "queued"},
		{name: "running", counts: map[string]int{"running": 1, "queued": 1}, want: "running"},
		{name: "search completed", counts: map[string]int{"completed": 2}, want: "search_completed"},
		{name: "partial failed", counts: map[string]int{"completed": 1, "failed": 1}, want: "partial_failed"},
		{name: "failed", counts: map[string]int{"failed": 2}, want: "failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := searchGroupStatus(test.counts); got != test.want {
				t.Fatalf("searchGroupStatus() = %q, want %q", got, test.want)
			}
		})
	}
}
