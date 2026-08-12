package compose2quadlet

import (
	"bytes"
	"os"

	"github.com/Inoriol/comquad/compose2quadlet/internal/types"
	"github.com/Inoriol/comquad/compose2quadlet/mapper"
)

func defaultConfig() *types.Config {
	return types.DefaultConfig()
}

func ParseVersion(s string) (Version, error) { return types.ParseVersion(s) }

func WithoutSELinux() types.Option     { return types.WithoutSELinux() }
func WithoutPrefix() types.Option       { return types.WithoutPrefix() }
func WithPrefix(p string) types.Option  { return types.WithPrefix(p) }
func WithoutDefaultNetwork() types.Option   { return types.WithoutDefaultNetwork() }
func WithPortOffset(o int) types.Option     { return types.WithPortOffset(o) }
func WithProjectName(n string) types.Option { return types.WithProjectName(n) }
func WithLabels(l map[string]string) types.Option { return types.WithLabels(l) }
func WithAutoUpdate() types.Option          { return types.WithAutoUpdate() }
func WithoutInstallSection() types.Option   { return types.WithoutInstallSection() }
func WithoutNetworkAliases() types.Option   { return types.WithoutNetworkAliases() }
func WithPodmanVersion(v Version) types.Option { return types.WithPodmanVersion(v) }
func WithImageRetry(n int) types.Option      { return types.WithImageRetry(n) }
func WithImageRetryDelay(s int) types.Option  { return types.WithImageRetryDelay(s) }
func WithWorkingDirectory(p string) types.Option      { return types.WithWorkingDirectory(p) }
func WithSecretsDirectory(p string) types.Option      { return types.WithSecretsDirectory(p) }
func WithDryRun() types.Option                        { return types.WithDryRun() }
func WithBuildCacheDir(p string) types.Option         { return types.WithBuildCacheDir(p) }
func WithDockerfileNormalization() types.Option       { return types.WithDockerfileNormalization() }
func WithInfo(fn func(string)) types.Option             { return types.WithInfo(fn) }

func PatchDockerfileFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	patched, err := mapper.PatchDockerfileFROM(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	return string(patched), nil
}
