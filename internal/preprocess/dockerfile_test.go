package preprocess

import (
	"strings"
	"testing"
)

func TestPatchDockerfile_SimpleFrom(t *testing.T) {
	input := "FROM nginx:alpine"
	expected := "FROM docker.io/library/nginx:alpine"
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPatchDockerfile_FromWithAS(t *testing.T) {
	input := "FROM nginx:alpine AS builder"
	expected := "FROM docker.io/library/nginx:alpine AS builder"
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPatchDockerfile_FromWithPlatform(t *testing.T) {
	input := "FROM --platform=linux/amd64 node:20 AS builder"
	expected := "FROM --platform=linux/amd64 docker.io/library/node:20 AS builder"
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPatchDockerfile_AlreadyQualified(t *testing.T) {
	input := "FROM docker.io/library/nginx:alpine"
	expected := "FROM docker.io/library/nginx:alpine"
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPatchDockerfile_CustomRegistry(t *testing.T) {
	input := "FROM myregistry.com/myimage:latest"
	expected := "FROM myregistry.com/myimage:latest"
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPatchDockerfile_Scratch(t *testing.T) {
	input := "FROM scratch"
	expected := "FROM scratch"
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPatchDockerfile_MultiStage(t *testing.T) {
	input := strings.Join([]string{
		"FROM node:20 AS build",
		"WORKDIR /app",
		"RUN npm build",
		"FROM nginx:alpine",
		"COPY --from=build /app/dist /usr/share/nginx/html",
	}, "\n")
	expected := strings.Join([]string{
		"FROM docker.io/library/node:20 AS build",
		"WORKDIR /app",
		"RUN npm build",
		"FROM docker.io/library/nginx:alpine",
		"COPY --from=build /app/dist /usr/share/nginx/html",
	}, "\n")
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

func TestPatchDockerfile_NoFromLine(t *testing.T) {
	input := "RUN echo hello\nCOPY . /app"
	result := string(PatchDockerfile([]byte(input)))
	if result != input {
		t.Errorf("expected no changes, got %q", result)
	}
}

func TestPatchDockerfile_LibraryPrefix(t *testing.T) {
	input := "FROM library/nginx:alpine"
	expected := "FROM docker.io/library/nginx:alpine"
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPatchDockerfile_WithoutTag(t *testing.T) {
	input := "FROM nginx"
	expected := "FROM docker.io/library/nginx"
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPatchDockerfile_PlatformOnly(t *testing.T) {
	input := "FROM --platform=linux/arm64 alpine:3.19"
	expected := "FROM --platform=linux/arm64 docker.io/library/alpine:3.19"
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPatchDockerfile_WithComment(t *testing.T) {
	input := strings.Join([]string{
		"# This is a comment",
		"FROM alpine:latest",
		"# Another comment",
		"RUN apk add curl",
	}, "\n")
	expected := strings.Join([]string{
		"# This is a comment",
		"FROM docker.io/library/alpine:latest",
		"# Another comment",
		"RUN apk add curl",
	}, "\n")
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

func TestPatchDockerfile_LowercaseFrom(t *testing.T) {
	input := "from nginx:alpine as builder"
	expected := "FROM docker.io/library/nginx:alpine as builder"
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPatchDockerfile_StageAliasNotNormalized(t *testing.T) {
	input := strings.Join([]string{
		"FROM node:20 AS base",
		"FROM base AS development",
	}, "\n")
	expected := strings.Join([]string{
		"FROM docker.io/library/node:20 AS base",
		"FROM base AS development",
	}, "\n")
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

func TestPatchDockerfile_StageAliasCaseInsensitive(t *testing.T) {
	input := strings.Join([]string{
		"FROM node:20 AS Base",
		"FROM base AS development",
	}, "\n")
	expected := strings.Join([]string{
		"FROM docker.io/library/node:20 AS Base",
		"FROM base AS development",
	}, "\n")
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

func TestPatchDockerfile_StageAliasSameAsRegistryImage(t *testing.T) {
	input := strings.Join([]string{
		"FROM alpine:latest AS builder",
		"FROM builder",
	}, "\n")
	expected := strings.Join([]string{
		"FROM docker.io/library/alpine:latest AS builder",
		"FROM builder",
	}, "\n")
	result := string(PatchDockerfile([]byte(input)))
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}
