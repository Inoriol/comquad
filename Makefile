# Build the test container image
.PHONY: test-image
test-image:
	podman build -t comquad-test:latest -f tests/integration/Containerfile .

# Run integration tests as root
.PHONY: integration-root
integration-root:
	podman run --rm --privileged \
		--cgroupns=host \
		-v /sys/fs/cgroup:/sys/fs/cgroup:rw \
		-v $(shell pwd):/workspace:z \
		-w /workspace \
		-e CQ_BINARY=/workspace/comquad \
		comquad-test:latest \
		/bin/bash -c " \
			set -e; \
			go build -o /workspace/comquad ./cmd/comquad && \
			go test -tags integration -v -count=1 -timeout 5m ./tests/integration/... \
		"

# Run integration tests as testuser (rootless)
.PHONY: integration-rootless
integration-rootless:
	podman run --rm --privileged \
		--cgroupns=host \
		-v /sys/fs/cgroup:/sys/fs/cgroup:rw \
		-v $(shell pwd):/workspace:z \
		-w /workspace \
		-e CQ_BINARY=/workspace/comquad \
		comquad-test:latest \
		/bin/bash -c " \
			set -e; \
			go build -o /workspace/comquad ./cmd/comquad; \
			mkdir -p /run/user/1000 && chown testuser /run/user/1000; \
			su - testuser -c ' \
				export CQ_BINARY=/workspace/comquad; \
				export XDG_RUNTIME_DIR=/run/user/\$$(id -u); \
				cd /workspace; \
				go test -tags integration -v -count=1 -timeout 5m ./tests/integration/... \
			' \
		"

.PHONY: integration
integration: test-image integration-root integration-rootless
