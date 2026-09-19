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

func TestNoteFileID(t *testing.T) {
	raw := "https://sns-na-i4.xhscdn.com/notes_pre_post/1040g3k831ufeu16h36d05q3ght36s1t02jc3ma8?imageView2/2/w/576"
	if got := noteFileID(raw); got != "notes_pre_post/1040g3k831ufeu16h36d05q3ght36s1t02jc3ma8" {
		t.Fatalf("noteFileID() = %q", got)
	}
}
