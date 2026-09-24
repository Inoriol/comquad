package compose2quadlet

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
)

func TestAnalyzeSharedVolumes_NamedVolumeShared(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{
			Name: "web",
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeVolume, Source: "data"},
			},
		},
		"db": types.ServiceConfig{
			Name: "db",
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeVolume, Source: "data"},
			},
		},
	}

	shared := analyzeSharedVolumes(services)
	if !shared["data"] {
		t.Fatal("expected 'data' volume to be shared")
	}
}

func TestAnalyzeSharedVolumes_NamedVolumeNotShared(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{
			Name: "web",
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeVolume, Source: "webdata"},
			},
		},
		"db": types.ServiceConfig{
			Name: "db",
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeVolume, Source: "dbdata"},
			},
		},
	}

	shared := analyzeSharedVolumes(services)
	if shared["webdata"] {
		t.Fatal("expected 'webdata' volume to not be shared")
	}
	if shared["dbdata"] {
		t.Fatal("expected 'dbdata' volume to not be shared")
	}
}

func TestAnalyzeSharedVolumes_BindMountShared(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{
			Name: "web",
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeBind, Source: "/shared/config"},
			},
		},
		"api": types.ServiceConfig{
			Name: "api",
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeBind, Source: "/shared/config"},
			},
		},
	}

	shared := analyzeSharedVolumes(services)
	if !shared["/shared/config"] {
		t.Fatal("expected '/shared/config' bind mount to be shared")
	}
}

func TestAnalyzeSharedVolumes_BindMountNotShared(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{
			Name: "web",
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeBind, Source: "/web/config"},
			},
		},
		"api": types.ServiceConfig{
			Name: "api",
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeBind, Source: "/api/config"},
			},
		},
	}

	shared := analyzeSharedVolumes(services)
	if shared["/web/config"] {
		t.Fatal("expected '/web/config' bind mount to not be shared")
	}
	if shared["/api/config"] {
		t.Fatal("expected '/api/config' bind mount to not be shared")
	}
}

func TestAnalyzeSharedVolumes_SameServiceMultipleRefs(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{
			Name: "web",
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeVolume, Source: "data"},
				{Type: types.VolumeTypeVolume, Source: "data"},
			},
		},
	}

	shared := analyzeSharedVolumes(services)
	if shared["data"] {
		t.Fatal("expected 'data' volume to not be shared when only used by one service")
	}
}

func TestAnalyzeSharedVolumes_TmpfsIgnored(t *testing.T) {
	services := types.Services{
		"web": types.ServiceConfig{
			Name: "web",
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeTmpfs, Target: "/tmp"},
			},
		},
		"api": types.ServiceConfig{
			Name: "api",
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeTmpfs, Target: "/tmp"},
			},
		},
	}

	shared := analyzeSharedVolumes(services)
	if len(shared) != 0 {
		t.Fatal("expected no shared volumes for tmpfs mounts")
	}
}
