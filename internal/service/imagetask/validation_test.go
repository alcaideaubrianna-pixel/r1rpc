package imagetask

import (
	"strconv"
	"testing"
)

func TestValidateSourceAndFiles(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		fileIDs []string
		wantErr bool
	}{
		{name: "single", source: "admin", fileIDs: []string{"file-1"}},
		{name: "one hundred", source: "api", fileIDs: uniqueFileIDs(100)},
		{name: "empty source", source: "", fileIDs: []string{"file-1"}, wantErr: true},
		{name: "empty files", source: "admin", wantErr: true},
		{name: "over limit", source: "admin", fileIDs: uniqueFileIDs(101), wantErr: true},
		{name: "empty file", source: "admin", fileIDs: []string{" "}, wantErr: true},
		{name: "duplicate", source: "admin", fileIDs: []string{"file-1", "file-1"}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSourceAndFiles(test.source, test.fileIDs)
			if (err != nil) != test.wantErr {
				t.Fatalf("error=%v, wantErr=%v", err, test.wantErr)
			}
		})
	}
}

func TestNullableString(t *testing.T) {
	if got := nullableString("  "); got != nil {
		t.Fatalf("空 externalId 应转换为 nil，实际=%#v", got)
	}
	if got := nullableString(" id-1 "); got != "id-1" {
		t.Fatalf("externalId=%#v", got)
	}
}

func uniqueFileIDs(count int) []string {
	result := make([]string, count)
	for index := range result {
		result[index] = "file-" + strconv.Itoa(index+1)
	}
	return result
}
