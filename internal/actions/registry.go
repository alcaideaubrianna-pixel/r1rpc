package actions

import "sort"

type Input struct {
	Path        string   `json:"path"`
	Type        string   `json:"type"`
	Label       string   `json:"label"`
	Required    bool     `json:"required"`
	Accept      []string `json:"accept,omitempty"`
	Description string   `json:"description,omitempty"`
}

type Definition struct {
	Name            string         `json:"name"`
	PayloadTemplate map[string]any `json:"payloadTemplate"`
	Inputs          []Input        `json:"inputs"`
}

var registry = map[string]Definition{
	"content.search_notes": {
		Name: "content.search_notes",
		PayloadTemplate: map[string]any{"query": "", "limit": 20},
		Inputs: []Input{{Path: "query", Type: "text", Label: "搜索词", Required: true}, {Path: "limit", Type: "number", Label: "数量"}},
	},
	"content.search": {
		Name: "content.search",
		PayloadTemplate: map[string]any{"query": "", "limit": 20},
		Inputs: []Input{{Path: "query", Type: "text", Label: "搜索词", Required: true}, {Path: "limit", Type: "number", Label: "数量"}},
	},
	"media.upload_image": {
		Name: "media.upload_image",
		PayloadTemplate: map[string]any{"image": map[string]any{"fileId": ""}, "purpose": ""},
		Inputs: []Input{{Path: "image.fileId", Type: "file", Label: "图片", Required: true, Accept: []string{"image/jpeg", "image/png", "image/heic", "image/heif"}}, {Path: "purpose", Type: "text", Label: "用途"}},
	},
	"content.search_by_image": {
		Name: "content.search_by_image",
		PayloadTemplate: map[string]any{"image": map[string]any{"fileId": ""}},
		Inputs: []Input{{Path: "image.fileId", Type: "file", Label: "搜索图片", Required: true, Accept: []string{"image/jpeg", "image/png", "image/heic", "image/heif"}}},
	},
	"diagnostics.image_search_native_probe": {
		Name: "diagnostics.image_search_native_probe",
		PayloadTemplate: map[string]any{"image": map[string]any{"fileId": ""}},
		Inputs: []Input{{Path: "image.fileId", Type: "file", Label: "探测图片", Required: true, Accept: []string{"image/jpeg", "image/png", "image/heic", "image/heif"}}},
	},
	"network.request": {
		Name: "network.request",
		PayloadTemplate: map[string]any{"url": "", "method": "GET", "headers": map[string]any{}, "body": ""},
		Inputs: []Input{{Path: "url", Type: "text", Label: "URL", Required: true}, {Path: "method", Type: "text", Label: "方法"}},
	},
}

func All() map[string]Definition {
	result := make(map[string]Definition, len(registry))
	for name, definition := range registry {
		result[name] = definition
	}
	return result
}

func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func Get(name string) (Definition, bool) {
	definition, ok := registry[name]
	return definition, ok
}
