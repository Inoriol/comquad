#!/bin/bash
# Integration test entrypoint — builds comquad and runs the test suite.
# Set CQ_ROOTLESS=1 to run tests as the testuser (rootless mode).

set -e

echo "=== Comquad Integration Test Runner ==="
echo "Building comquad..."
go build -o /workspace/comquad ./cmd/comquad
export CQ_BINARY=/workspace/comquad

TEST_ARGS="${@:-./tests/integration/...}"

if [ "${CQ_ROOTLESS:-}" = "1" ]; then
    echo "=== Running ROOTLESS integration tests ==="
    su - testuser -c "
        export CQ_BINARY=/workspace/comquad
        export XDG_RUNTIME_DIR=/run/user/\$(id -u)
        mkdir -p \$XDG_RUNTIME_DIR
        cd /workspace
        go test -tags integration -v -count=1 $TEST_ARGS
    "
else
    echo "=== Running ROOT integration tests ==="
    go test -tags integration -v -count=1 $TEST_ARGS
fi

echo "=== Done ==="
