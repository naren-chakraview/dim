package s3

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// Live against LocalStack (deploy/docker-compose.e2e.yml). Runs only when
// DIM_LIVE_LOCALSTACK is set. An empty object must not be emitted as an empty message;
// once it has content, it must be.
func TestS3SourceSkipsEmptyObjectLive(t *testing.T) {
	endpoint := os.Getenv("DIM_LIVE_LOCALSTACK")
	if endpoint == "" {
		t.Skip("set DIM_LIVE_LOCALSTACK=http://localhost:4566 to run against LocalStack")
	}
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	bucket := "empty-object-test-" + time.Now().Format("150405")

	put := func(key string, body []byte) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPut, endpoint+"/"+bucket+"/"+key, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("PUT %s: HTTP %d", key, resp.StatusCode)
		}
	}
	req, _ := http.NewRequest(http.MethodPut, endpoint+"/"+bucket, nil)
	if resp, err := http.DefaultClient.Do(req); err != nil {
		t.Fatal(err)
	} else {
		resp.Body.Close()
	}
	put("empty.json", nil)
	put("full.json", []byte(`{"order": "full"}`))

	out := engine.NewChannel("s3-empty-test", 20)
	src, err := NewS3Source(SourceConfig{
		Endpoint:     endpoint,
		Bucket:       bucket,
		Region:       "us-east-1",
		PollInterval: 200 * time.Millisecond,
	}, out)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go src.Start(ctx)

	received := func(window time.Duration) []interface{} {
		var bodies []interface{}
		deadline := time.After(window)
		for {
			select {
			case <-deadline:
				return bodies
			default:
			}
			rctx, rcancel := context.WithTimeout(ctx, 100*time.Millisecond)
			msg, err := out.Recv(rctx)
			rcancel()
			if err == nil && msg != nil {
				bodies = append(bodies, msg.Body)
			}
		}
	}

	first := received(2 * time.Second)
	if len(first) != 1 {
		t.Fatalf("first poll emitted %d message(s), want 1 (the empty object must be skipped): %#v", len(first), first)
	}
	if m, ok := first[0].(map[string]interface{}); !ok || m["order"] != "full" {
		t.Fatalf("emitted %#v, want the full object", first[0])
	}

	// Once the empty object has content, it is a new version and must be emitted.
	put("empty.json", []byte(`{"order": "later"}`))
	second := received(2 * time.Second)
	found := false
	for _, b := range second {
		if m, ok := b.(map[string]interface{}); ok && m["order"] == "later" {
			found = true
		}
	}
	if !found {
		t.Fatalf("object written after the skip was not emitted; got %#v", second)
	}
}
