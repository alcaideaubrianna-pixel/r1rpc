package xhsprofile

import (
	"encoding/json"
	"testing"
)

func TestParseResponseFromNetworkEnvelope(t *testing.T) {
	raw := json.RawMessage(`{"statusCode":200,"bodyEncoding":"json","body":{"code":0,"success":true,"data":{"userid":"68708f46000000001b0207a0","red_id":"26159678150","nickname":"Siri","fans":679}}}`)
	data, err := parseResponse(raw)
	if err != nil {
		t.Fatalf("parseResponse() error = %v", err)
	}
	if got := firstString(data, "red_id"); got != "26159678150" {
		t.Fatalf("red_id = %q", got)
	}
	if got := int64Value(data["fans"]); got != 679 {
		t.Fatalf("fans = %d", got)
	}
}
