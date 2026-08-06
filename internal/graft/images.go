package graft

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Inoriol/comquad/internal/logger"
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

// Engine handles image pulling
type Engine struct {
	PullStrategy PullStrategy
}

// ImageResult represents the outcome of handling an image
type ImageResult struct {
	Service string
	Image   string
	Action  string // "pulled", "found", "skipped"
	Error   error
}

// ImageExists checks if an image exists locally.
// Returns false if the image is not found (exit code 125) or if podman
// encounters an unexpected error. Non-125 errors are logged as warnings
// since they may indicate a podman installation problem.
func (e *Engine) ImageExists(image string) bool {
	cmd := exec.Command("podman", "image", "inspect", image)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() != 125 {
			logger.Warn(fmt.Sprintf("podman image inspect failed for %s: %v", image, err))
		}
		return false
	}
	return true
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

// HandleImage handles image pulling based on strategy
func (e *Engine) HandleImage(service, image string) error {
	switch e.PullStrategy {
	case PullAlways:
		return e.PullImage(image)
	case PullNever:
		if !e.ImageExists(image) {
			return fmt.Errorf("image %s not found locally and pull strategy is 'never'", image)
		}
		logger.Action("Using local image: " + image)
		return nil
	case PullMissing:
		if e.ImageExists(image) {
			logger.Action("Image already exists locally: " + image)
			return nil
		}
		return e.PullImage(image)
	default:
		return fmt.Errorf("unknown pull strategy: %s", e.PullStrategy)
	}
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
