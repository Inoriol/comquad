package build

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"comquad/internal/logger"
)

// PullStrategy defines how images should be pulled
type PullStrategy string

const (
	// PullAlways always pulls images
	PullAlways PullStrategy = "always"
	// PullMissing pulls only if image not found locally
	PullMissing PullStrategy = "missing"
	// PullNever never pulls images
	PullNever PullStrategy = "never"
)

// Engine handles image building and pulling
type Engine struct {
	PullStrategy PullStrategy
}

// ImageResult represents the outcome of handling an image
type ImageResult struct {
	Service  string
	Image    string
	Action   string // "built", "pulled", "found", "skipped"
	Error    error
}

// ImageExists checks if an image exists locally
func (e *Engine) ImageExists(image string) bool {
	cmd := exec.Command("podman", "image", "inspect", image)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// BuildService builds an image from the given build configuration
func (e *Engine) BuildService(service string, context string, dockerfile string, args []string, target string, tag string) error {
	cmdArgs := []string{"build", "-t", tag}

	if dockerfile != "" {
		cmdArgs = append(cmdArgs, "--file", dockerfile)
	}

	if target != "" {
		cmdArgs = append(cmdArgs, "--target", target)
	}

	for _, arg := range args {
		cmdArgs = append(cmdArgs, "--build-arg", arg)
	}

	cmdArgs = append(cmdArgs, context)

	cmd := exec.Command("podman", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logger.Action("Building image for service " + service + ": " + tag)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build image for service %s: %w", service, err)
	}

	return nil
}

// PullImage pulls an image from a registry
func (e *Engine) PullImage(image string) error {
	logger.Action("Pulling image: " + image)

	cmd := exec.Command("podman", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", image, err)
	}

	return nil
}

// HandleImage handles image pulling or building based on strategy
func (e *Engine) HandleImage(service, image string) error {
	switch e.PullStrategy {
	case PullAlways:
		return e.PullImage(image)
	case PullNever:
		if !e.ImageExists(image) {
			return fmt.Errorf("image %s not found locally and pull strategy is 'never'", image)
		}
		logger.Info("Using local image: " + image)
		return nil
	case PullMissing:
		if e.ImageExists(image) {
			logger.Info("Image already exists locally: " + image)
			return nil
		}
		return e.PullImage(image)
	default:
		return fmt.Errorf("unknown pull strategy: %s", e.PullStrategy)
	}
}


// GetBuildArgs returns podman build-arg flags from a map
func GetBuildArgs(args map[string]string) []string {
	result := []string{}
	for k, v := range args {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

// GenerateBuildTag generates a tag for a built image
func GenerateBuildTag(projectName, serviceName string) string {
	return fmt.Sprintf("%s-%s:latest", projectName, serviceName)
}

// ParsePullStrategy converts a string to PullStrategy
func ParsePullStrategy(s string) (PullStrategy, error) {
	switch strings.ToLower(s) {
	case "always":
		return PullAlways, nil
	case "missing":
		return PullMissing, nil
	case "never":
		return PullNever, nil
	default:
		return "", fmt.Errorf("invalid pull strategy: %s (must be 'always', 'missing', or 'never')", s)
	}
}
