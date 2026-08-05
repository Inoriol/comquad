.PHONY: build
build:
	go build -ldflags "-X main.version=$(shell git describe --tags --always 2>/dev/null || echo dev)" -o comquad ./cmd/comquad/

# ── Unit tests ────────────────────────────────────────────────────────────────

.PHONY: test-unit
test-unit:
	go test -count=1 -timeout 2m ./...

.PHONY: test-race
test-race:
	go test -race -count=1 -timeout 3m ./...

.PHONY: test-cover
test-cover:
	go test -cover -coverprofile=coverage.out -count=1 -timeout 2m ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: test-short
test-short:
	go test -short -count=1 -timeout 1m ./...

# ── Integration tests ─────────────────────────────────────────────────────────

.PHONY: test-image
test-image:
	podman build -t comquad-test:latest -f tests/integration/Containerfile .

.PHONY: integration-root
integration-root: build
	podman run --rm --privileged \
		--cgroupns=host \
		-v /sys/fs/cgroup:/sys/fs/cgroup:rw \
		-v $(shell pwd):/workspace:z \
		-w /workspace \
		-e CQ_BINARY=/workspace/comquad \
		comquad-test:latest \
		/bin/bash -c " \
			set -e; \
			go test -tags integration -count=1 -timeout 5m ./tests/integration/... \
		"

.PHONY: integration-rootless
integration-rootless: build
	podman run --rm --privileged \
		--cgroupns=host \
		-v /sys/fs/cgroup:/sys/fs/cgroup:rw \
		-v $(shell pwd):/workspace:z \
		-w /workspace \
		-e CQ_BINARY=/workspace/comquad \
		comquad-test:latest \
		/bin/bash -c " \
			set -e; \
			mkdir -p /run/user/1000 && chown testuser /run/user/1000; \
			su - testuser -c ' \
				export CQ_BINARY=/workspace/comquad; \
				export XDG_RUNTIME_DIR=/run/user/\$$(id -u); \
				cd /workspace; \
				go test -tags integration -count=1 -timeout 5m ./tests/integration/... \
			' \
		"

.PHONY: integration
integration: build test-image integration-root integration-rootless
