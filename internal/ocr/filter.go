package ocr

import "strings"

func Matches(text string, keywords []string, mode string) bool {
	if len(keywords) == 0 {
		return false
	}
	normalized := strings.ToLower(text)
	matched := 0
	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" && strings.Contains(normalized, keyword) {
			matched++
			if mode != "contains_all" {
				return true
			}
		}
	}
	return mode == "contains_all" && matched == len(keywords)
}
