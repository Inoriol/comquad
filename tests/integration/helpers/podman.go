//go:build integration

package helpers

import (
    "encoding/json"
    "os/exec"
    "strings"
    "testing"
)

type PodmanContainer struct {
    Names  []string          `json:"Names"`
    State  string            `json:"State"`
    Labels map[string]string `json:"Labels"`
}

// InspectMount represents a single mount entry from podman inspect.
type InspectMount struct {
 Type        string `json:"Type"`
 Source      string `json:"Source"`
 Destination string `json:"Destination"`
 Mode        string `json:"Mode"`
 Options     []string `json:"Options"`
}

// ListContainers returns all podman containers matching the name filter.
// Pass an empty filter to list all.
func ListContainers(t *testing.T, nameFilter string) []PodmanContainer {
    t.Helper()
    args := []string{"podman", "ps", "-a", "--format=json"}
    if nameFilter != "" {
        args = append(args, "--filter", "name="+nameFilter)
    }
    out, err := exec.Command(args[0], args[1:]...).Output()
    if err != nil {
        t.Fatalf("podman ps failed: %v", err)
    }
    var containers []PodmanContainer
    if err := json.Unmarshal(out, &containers); err != nil {
        t.Fatalf("failed to parse podman ps output: %v", err)
    }
    return containers
}

// AssertContainerRunning fails if no running container matches the name prefix.
func AssertContainerRunning(t *testing.T, namePrefix string) {
    t.Helper()
    containers := ListContainers(t, namePrefix)
    for _, c := range containers {
        if strings.EqualFold(c.State, "running") {
            return
        }
    }
    t.Fatalf("no running container matching %q found", namePrefix)
}

// AssertContainerGone fails if any container matching the name prefix still exists.
func AssertContainerGone(t *testing.T, namePrefix string) {
    t.Helper()
    containers := ListContainers(t, namePrefix)
    if len(containers) > 0 {
        t.Fatalf("expected no containers matching %q but found %d", namePrefix, len(containers))
    }
}

// NetworkExists returns true if a podman network with the given name exists.
func NetworkExists(t *testing.T, name string) bool {
    t.Helper()
    err := exec.Command("podman", "network", "inspect", name).Run()
    return err == nil
}

// AssertNetworkGone fails if the network still exists.
func AssertNetworkGone(t *testing.T, name string) {
    t.Helper()
    if NetworkExists(t, name) {
        t.Fatalf("expected network %q to be removed but it still exists", name)
    }
}

// VolumeExists returns true if a podman volume with the given name exists.
func VolumeExists(t *testing.T, name string) bool {
    t.Helper()
    err := exec.Command("podman", "volume", "inspect", name).Run()
    return err == nil
}

// InspectContainer returns the full low-level JSON for a single container.
// containerName is the Podman container name (not the unit name).
func InspectContainer(t *testing.T, containerName string) map[string]interface{} {
 t.Helper()
 out, err := exec.Command("podman", "inspect", "--format=json", containerName).Output()
 if err != nil {
  t.Fatalf("podman inspect %q failed: %v", containerName, err)
 }

 // inspect returns a JSON array
 var result []map[string]interface{}
 if err := json.Unmarshal(out, &result); err != nil {
  t.Fatalf("failed to parse podman inspect output: %v", err)
 }
 if len(result) == 0 {
  t.Fatalf("podman inspect returned empty result for %q", containerName)
 }
 return result[0]
}

// GetContainerMounts returns the mount list for a running container.
func GetContainerMounts(t *testing.T, containerName string) []InspectMount {
 t.Helper()
 data := InspectContainer(t, containerName)

 raw, ok := data["Mounts"]
 if !ok {
  return nil
 }

 // Re-marshal and unmarshal into typed struct
 b, err := json.Marshal(raw)
 if err != nil {
  t.Fatalf("failed to re-marshal mounts: %v", err)
 }
 var mounts []InspectMount
 if err := json.Unmarshal(b, &mounts); err != nil {
  t.Fatalf("failed to parse mounts: %v", err)
 }
 return mounts
}

// AssertMountHasOption fails if no mount at destination has the given option.
func AssertMountHasOption(t *testing.T, containerName, destination, option string) {
 t.Helper()
 mounts := GetContainerMounts(t, containerName)
 for _, m := range mounts {
  if m.Destination != destination {
   continue
  }
  // Check Mode string first (bind mounts)
  if strings.Contains(m.Mode, option) {
   return
  }
  // Check Options slice (volume mounts)
  for _, o := range m.Options {
   if o == option {
    return
   }
  }
  t.Fatalf("mount at %q exists but missing option %q (mode=%q, options=%v)",
   destination, option, m.Mode, m.Options)
 }
 t.Fatalf("no mount found at destination %q in container %q", destination, containerName)
}

// AssertMountMissingOption fails if a mount at destination has the given option.
func AssertMountMissingOption(t *testing.T, containerName, destination, option string) {
 t.Helper()
 mounts := GetContainerMounts(t, containerName)
 for _, m := range mounts {
  if m.Destination != destination {
   continue
  }
  if strings.Contains(m.Mode, option) {
   t.Fatalf("mount at %q unexpectedly has option %q (mode=%q)", destination, option, m.Mode)
  }
  for _, o := range m.Options {
   if o == option {
    t.Fatalf("mount at %q unexpectedly has option %q", destination, option)
   }
  }
  return
 }
 // Mount not found at all — that's fine for this assertion
}

