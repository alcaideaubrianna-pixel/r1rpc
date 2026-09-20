package ocr

import "testing"

func TestMatches(t *testing.T) {
	tests := []struct {
		name, text, mode string
		keywords         []string
		want             bool
	}{
		{name: "any", text: "本图包含广告内容", mode: "contains_any", keywords: []string{"广告", "二维码"}, want: true},
		{name: "all", text: "广告二维码", mode: "contains_all", keywords: []string{"广告", "二维码"}, want: true},
		{name: "all missing", text: "仅有广告", mode: "contains_all", keywords: []string{"广告", "二维码"}},
		{name: "case insensitive", text: "PROMO", mode: "contains_any", keywords: []string{"promo"}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Matches(test.text, test.keywords, test.mode); got != test.want {
				t.Fatalf("Matches() = %v, want %v", got, test.want)
			}
		})
	}
}
