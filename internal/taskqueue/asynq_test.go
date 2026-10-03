package taskqueue

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

func TestImageTaskPayloadContainsOnlyJobID(t *testing.T) {
	raw, err := json.Marshal(imageJobPayload{JobID: "job-1"})
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	if len(object) != 1 || object["jobId"] != "job-1" {
		t.Fatalf("payload=%v", object)
	}
}

func TestSourceMaterialTimeout(t *testing.T) {
	if sourceMaterialTimeout != 100*time.Second {
		t.Fatalf("资料搜索超时 = %s, want 100s", sourceMaterialTimeout)
	}
}

func TestRedisOptions(t *testing.T) {
	got := RedisOptions("redis:6379", "secret", 2)
	want := asynq.RedisClientOpt{Addr: "redis:6379", Password: "secret", DB: 2}
	if got != want {
		t.Fatalf("options=%+v", got)
	}
}

func TestStageRequestIDIsUniqueAndFitsDatabase(t *testing.T) {
	first := stageRequestID("0123456789abcdef0123456789abcdef", "u", 1)
	second := stageRequestID("0123456789abcdef0123456789abcdef", "u", 2)
	if first == second {
		t.Fatal("不同重试必须使用不同 RPC requestId")
	}
	if len(first) > 64 || len(second) > 64 {
		t.Fatalf("requestId 超过数据库限制: %q / %q", first, second)
	}
}
