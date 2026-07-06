# Build the test container image
.PHONY: test-image
test-image:
 podman build -t comquad-test:latest -f tests/integration/Containerfile .

# Run integration tests as root inside the container
.PHONY: integration-root
integration-root:
 podman run --rm --privileged \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  -v $(PWD):/workspace:z \
  -w /workspace \
  -e CQ_BINARY=/workspace/comquad \
  comquad-test:latest \
  /bin/bash -c " \
   go build -o /workspace/comquad ./cmd/comquad && \
   go test -tags integration -v -count=1 ./tests/integration/... \
  "

# Run integration tests as testuser (rootless) inside the container
.PHONY: integration-rootless
integration-rootless:
 podman run --rm --privileged \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  -v $(PWD):/workspace:z \
  -w /workspace \
  -e CQ_BINARY=/workspace/comquad \
  comquad-test:latest \
  /bin/bash -c " \
   go build -o /workspace/comquad ./cmd/comquad && \
   su - testuser -c ' \
    export CQ_BINARY=/workspace/comquad && \
    export XDG_RUNTIME_DIR=/run/user/\$(id -u) && \
    mkdir -p \$$XDG_RUNTIME_DIR && \
    go test -tags integration -v -count=1 /workspace/tests/integration/... \
   ' \
  "

.PHONY: integration
integration: integration-root integration-rootless
