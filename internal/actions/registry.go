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
	"image.recognize": {
		Name:            "image.recognize",
		PayloadTemplate: map[string]any{"image": map[string]any{"fileId": ""}, "forceRefresh": false, "priority": 0},
		Inputs:          []Input{{Path: "image.fileId", Type: "file", Label: "识别图片", Required: true, Accept: []string{"image/jpeg", "image/png", "image/heic", "image/heif"}}},
	},
	"content.search_notes": {
		Name:            "content.search_notes",
		PayloadTemplate: map[string]any{"query": "", "limit": 20},
		Inputs:          []Input{{Path: "query", Type: "text", Label: "搜索词", Required: true}, {Path: "limit", Type: "number", Label: "数量"}},
	},
	"content.search": {
		Name:            "content.search",
		PayloadTemplate: map[string]any{"query": "", "limit": 20},
		Inputs:          []Input{{Path: "query", Type: "text", Label: "搜索词", Required: true}, {Path: "limit", Type: "number", Label: "数量"}},
	},
	"media.upload_image": {
		Name:            "media.upload_image",
		PayloadTemplate: map[string]any{"image": map[string]any{"fileId": ""}, "purpose": "image_search"},
		Inputs:          []Input{{Path: "image.fileId", Type: "file", Label: "图片", Required: true, Accept: []string{"image/jpeg", "image/png", "image/heic", "image/heif"}}, {Path: "purpose", Type: "text", Label: "用途"}},
	},
	"content.search_by_image": {
		Name:            "content.search_by_image",
		PayloadTemplate: map[string]any{"image": map[string]any{"fileId": ""}, "uploadHandle": "", "crop": map[string]any{"x": 0, "y": 0, "width": 1, "height": 1}, "limit": 20},
		Inputs: []Input{
			{Path: "image.fileId", Type: "file", Label: "搜索图片", Accept: []string{"image/jpeg", "image/png", "image/heic", "image/heif"}, Description: "选择图片时自动先执行 media.upload_image"},
			{Path: "uploadHandle", Type: "text", Label: "上传句柄", Description: "已有上传句柄时可直接搜索"},
		},
	},
	"diagnostics.image_search_native_probe": {
		Name:            "diagnostics.image_search_native_probe",
		PayloadTemplate: map[string]any{"uploadHandle": ""},
		Inputs:          []Input{{Path: "uploadHandle", Type: "text", Label: "上传句柄", Required: true}},
	},
	"network.request": {
		Name:            "network.request",
		PayloadTemplate: map[string]any{"path": "/api/sns/v3/user/info", "method": "GET", "query": map[string]any{}, "headers": map[string]any{}, "timeoutMilliseconds": 15000},
		Inputs:          []Input{{Path: "path", Type: "text", Label: "API Path", Required: true}, {Path: "method", Type: "text", Label: "方法"}},
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
