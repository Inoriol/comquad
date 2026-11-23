#!/bin/bash
set -e

mkdir -p completions
go run ./cmd/comquad completion bash > completions/comquad.bash
go run ./cmd/comquad completion zsh > completions/comquad.zsh
go run ./cmd/comquad completion fish > completions/comquad.fish
