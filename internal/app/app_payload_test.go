package app

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMaterializeSearchByImagePayloadRemovesAuditImage(t *testing.T) {
	a := &App{}
	payload := json.RawMessage(`{"image":{"fileId":"0123456789abcdef0123456789abcdef"},"uploadHandle":"upload-1","crop":{"x":0,"y":0,"width":1,"height":1},"limit":20}`)

	got, err := a.materializeInvokePayload(context.Background(), "content.search_by_image", payload)
	if err != nil {
		t.Fatalf("materialize payload: %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(got, &object); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if _, exists := object["image"]; exists {
		t.Fatal("设备 payload 不应包含审计辅助字段 image")
	}
	if object["uploadHandle"] != "upload-1" {
		t.Fatalf("uploadHandle = %#v", object["uploadHandle"])
	}
}
