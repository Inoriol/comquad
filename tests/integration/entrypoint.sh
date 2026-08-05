#!/bin/bash
# Integration test entrypoint — runs the test suite inside the container.
# Set CQ_ROOTLESS=1 to run tests as the testuser (rootless mode).
# The comquad binary must already be built at CQ_BINARY before running.

set -e

echo "=== Comquad Integration Test Runner ==="
echo "Using binary: $CQ_BINARY"

TEST_ARGS="${@:-./tests/integration/...}"

if [ "${CQ_ROOTLESS:-}" = "1" ]; then
    echo "=== Running ROOTLESS integration tests ==="
    su - testuser -c "
        export CQ_BINARY=$CQ_BINARY
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
