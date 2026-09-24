package mapper

import (
	"bytes"
	"strings"
	"testing"
)

func TestPatchDockerfileFROM_Normalize(t *testing.T) {
	input := "FROM nginx:latest\nRUN echo hello\n"
	result, err := PatchDockerfileFROM(bytes.NewReader([]byte(input)), false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "FROM docker.io/library/nginx:latest") {
		t.Fatalf("expected normalized image, got: %s", string(result))
	}
}

func TestPatchDockerfileFROM_SkipNormalize(t *testing.T) {
	input := "FROM nginx:latest\nRUN echo hello\n"
	result, err := PatchDockerfileFROM(bytes.NewReader([]byte(input)), true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "FROM nginx:latest") {
		t.Fatalf("expected unmodified image, got: %s", string(result))
	}
	if strings.Contains(string(result), "docker.io") {
		t.Fatalf("image should not be normalized, got: %s", string(result))
	}
}

func TestPatchDockerfileFROM_UserImage(t *testing.T) {
	input := "FROM myuser/myimage:v1\n"
	result, err := PatchDockerfileFROM(bytes.NewReader([]byte(input)), false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "FROM docker.io/myuser/myimage:v1") {
		t.Fatalf("expected normalized user image, got: %s", string(result))
	}
}

func TestPatchDockerfileFROM_UserImage_SkipNormalize(t *testing.T) {
	input := "FROM myuser/myimage:v1\n"
	result, err := PatchDockerfileFROM(bytes.NewReader([]byte(input)), true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "FROM myuser/myimage:v1") {
		t.Fatalf("expected unmodified user image, got: %s", string(result))
	}
}

func TestPatchDockerfileFROM_CustomRegistry(t *testing.T) {
	input := "FROM ghcr.io/owner/image:tag\n"
	result, err := PatchDockerfileFROM(bytes.NewReader([]byte(input)), false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "FROM ghcr.io/owner/image:tag") {
		t.Fatalf("expected unchanged custom registry image, got: %s", string(result))
	}
}

func TestPatchDockerfileFROM_Scratch(t *testing.T) {
	input := "FROM scratch\n"
	result, err := PatchDockerfileFROM(bytes.NewReader([]byte(input)), false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "FROM scratch") {
		t.Fatalf("scratch should not be normalized, got: %s", string(result))
	}
}

func TestPatchDockerfileFROM_MultiStage(t *testing.T) {
	input := "FROM golang:1.21 AS builder\nRUN go build\nFROM alpine:latest\nCOPY --from=builder /app /app\n"
	result, err := PatchDockerfileFROM(bytes.NewReader([]byte(input)), false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "FROM docker.io/library/golang:1.21 AS builder") {
		t.Fatalf("expected normalized builder image, got: %s", string(result))
	}
	if !strings.Contains(string(result), "FROM docker.io/library/alpine:latest") {
		t.Fatalf("expected normalized final image, got: %s", string(result))
	}
}

func TestPatchDockerfileFROM_MultiStage_SkipNormalize(t *testing.T) {
	input := "FROM golang:1.21 AS builder\nRUN go build\nFROM alpine:latest\n"
	result, err := PatchDockerfileFROM(bytes.NewReader([]byte(input)), true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "FROM golang:1.21 AS builder") {
		t.Fatalf("expected unmodified builder image, got: %s", string(result))
	}
	if !strings.Contains(string(result), "FROM alpine:latest") {
		t.Fatalf("expected unmodified final image, got: %s", string(result))
	}
}

func TestPatchDockerfileFROM_StageAlias(t *testing.T) {
	input := "FROM golang:1.21 AS builder\nFROM builder AS runner\n"
	result, err := PatchDockerfileFROM(bytes.NewReader([]byte(input)), false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "FROM builder AS runner") {
		t.Fatalf("stage alias should not be normalized, got: %s", string(result))
	}
}
