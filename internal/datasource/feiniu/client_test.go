package feiniu

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTimestampUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		value string
		zone  string
	}{
		{name: "带时区", value: `"2026-09-20T03:29:51.646818Z"`, zone: "UTC"},
		{name: "无时区", value: `"2026-09-20T03:29:51.646818"`, zone: "UTC"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got Timestamp
			if err := json.Unmarshal([]byte(test.value), &got); err != nil {
				t.Fatalf("解析时间失败: %v", err)
			}
			if got.Format("2006-01-02T15:04:05.999999") != "2026-09-20T03:29:51.646818" {
				t.Fatalf("时间不匹配: %s", got.Format(time.RFC3339Nano))
			}
			if got.Location().String() != test.zone {
				t.Fatalf("时区不匹配: %s", got.Location())
			}
		})
	}
}

func TestPageUnmarshalNaiveWatermark(t *testing.T) {
	var page Page[Channel]
	if err := json.Unmarshal([]byte(`{"items":[],"watermark":"2026-09-20T03:29:51.646818"}`), &page); err != nil {
		t.Fatalf("解析分页响应失败: %v", err)
	}
	if page.Watermark.IsZero() {
		t.Fatal("watermark 不应为空")
	}
}

func TestChannelNormalizesUpstreamID(t *testing.T) {
	var channel Channel
	if err := json.Unmarshal([]byte(`{"channel_id":123,"title":"测试频道"}`), &channel); err != nil {
		t.Fatalf("解析频道失败: %v", err)
	}
	if channel.ID != 123 {
		t.Fatalf("频道 ID 不匹配: %d", channel.ID)
	}
	encoded, err := json.Marshal(channel)
	if err != nil {
		t.Fatalf("编码频道失败: %v", err)
	}
	if string(encoded) != `{"id":123,"title":"测试频道","username":"","chat_type":""}` {
		t.Fatalf("标准化响应不匹配: %s", encoded)
	}
}

func TestNoteAcceptsNumericAndStringSourceIDs(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		message string
		grouped string
	}{
		{name: "数字", payload: `{"source_message_id":123456,"grouped_id":987654}`, message: "123456", grouped: "987654"},
		{name: "字符串", payload: `{"source_message_id":"123456","grouped_id":"987654"}`, message: "123456", grouped: "987654"},
		{name: "空值", payload: `{"source_message_id":null,"grouped_id":null}`, message: "", grouped: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var note Note
			if err := json.Unmarshal([]byte(test.payload), &note); err != nil {
				t.Fatalf("解析笔记 ID 失败: %v", err)
			}
			if string(note.SourceMessageID) != test.message || string(note.GroupedID) != test.grouped {
				t.Fatalf("ID 不匹配: message=%q grouped=%q", note.SourceMessageID, note.GroupedID)
			}
		})
	}
}
