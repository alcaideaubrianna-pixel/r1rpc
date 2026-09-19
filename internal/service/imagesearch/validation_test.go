package imagesearch

import (
	"strings"
	"testing"

	"r1rpc/internal/model/input"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

func TestValidateCreate(t *testing.T) {
	valid := input.CreateImageSearchRequest{
		Source: "admin",
		Groups: []input.ImageSearchGroup{{
			Images: []input.ImageSearchItem{{FileID: strings.Repeat("a", 32)}},
		}},
	}
	tests := []struct {
		name   string
		mutate func(*input.CreateImageSearchRequest)
		valid  bool
	}{
		{name: "valid", valid: true},
		{name: "missing source", mutate: func(in *input.CreateImageSearchRequest) { in.Source = "" }},
		{name: "invalid priority", mutate: func(in *input.CreateImageSearchRequest) { in.Priority = 11 }},
		{name: "invalid callback", mutate: func(in *input.CreateImageSearchRequest) { in.CallbackURL = "file:///tmp/a" }},
		{name: "missing groups", mutate: func(in *input.CreateImageSearchRequest) { in.Groups = nil }},
		{name: "duplicate image", mutate: func(in *input.CreateImageSearchRequest) {
			in.Groups[0].Images = append(in.Groups[0].Images, in.Groups[0].Images[0])
		}},
		{name: "invalid file id", mutate: func(in *input.CreateImageSearchRequest) { in.Groups[0].Images[0].FileID = "short" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			in := cloneCreateInput(valid)
			if test.mutate != nil {
				test.mutate(&in)
			}
			err := validateCreate(in)
			if test.valid && err != nil {
				t.Fatalf("validateCreate() error = %v", err)
			}
			if !test.valid && gerror.Code(err) != gcode.CodeInvalidParameter {
				t.Fatalf("validateCreate() code = %v, error = %v", gerror.Code(err), err)
			}
		})
	}
}

func cloneCreateInput(in input.CreateImageSearchRequest) input.CreateImageSearchRequest {
	clone := in
	clone.Groups = append([]input.ImageSearchGroup(nil), in.Groups...)
	for index := range clone.Groups {
		clone.Groups[index].Images = append([]input.ImageSearchItem(nil), in.Groups[index].Images...)
	}
	return clone
}
