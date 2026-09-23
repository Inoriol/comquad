package output

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestSetJSONMode(t *testing.T) {
	SetJSONMode(true)
	if !IsJSONMode() {
		t.Error("expected JSONMode to be true")
	}
	SetJSONMode(false)
	if IsJSONMode() {
		t.Error("expected JSONMode to be false")
	}
}

func TestPrintJSON(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := &ProjectListData{
		Projects: []ProjectJSON{
			{
				Name:       "test-project",
				SourcePath: "/path/to/compose.yaml",
				Files:      3,
			},
		},
	}

	err := PrintJSON(data)
	if err != nil {
		t.Fatalf("PrintJSON failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var result Envelope
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result.Version != APIVersion {
		t.Errorf("expected version %s, got %s", APIVersion, result.Version)
	}
}

func TestPrintError(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintError("TEST_ERROR", "test message")
	if err != nil {
		t.Fatalf("PrintError failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var result ErrorEnvelope
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result.Version != APIVersion {
		t.Errorf("expected version %s, got %s", APIVersion, result.Version)
	}
	if result.Error.Code != "TEST_ERROR" {
		t.Errorf("expected error code TEST_ERROR, got %s", result.Error.Code)
	}
	if result.Error.Message != "test message" {
		t.Errorf("expected error message 'test message', got %s", result.Error.Message)
	}
}

func TestFormatPS(t *testing.T) {
	SetJSONMode(true)
	defer SetJSONMode(false)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	containers := []ContainerJSON{
		{
			Name:      "test-container",
			Image:     "nginx:latest",
			State:     "running",
			CreatedAt: time.Now(),
		},
	}

	err := FormatPS(containers)
	if err != nil {
		t.Fatalf("FormatPS failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var result Envelope
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result.Version != APIVersion {
		t.Errorf("expected version %s, got %s", APIVersion, result.Version)
	}
}

func TestFormatPS_NotJSONMode(t *testing.T) {
	SetJSONMode(false)

	err := FormatPS([]ContainerJSON{})
	if err != nil {
		t.Errorf("FormatPS should return nil when not in JSON mode, got: %v", err)
	}
}
