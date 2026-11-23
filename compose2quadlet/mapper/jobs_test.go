package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func boolPtrJob(b bool) *bool {
	return &b
}

func TestJobContainer_Image(t *testing.T) {
	job := types.JobConfig{Name: "migrate", ContainerSpec: types.ContainerSpec{Image: "myapp:latest"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "Image", "migrate.image")
}

func TestJobContainer_Command(t *testing.T) {
	job := types.JobConfig{
		Name:          "migrate",
		ContainerSpec: types.ContainerSpec{Command: types.ShellCommand{"python", "manage.py", "migrate"}},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "Exec", "'python' 'manage.py' 'migrate'")
}

func TestJobContainer_Environment(t *testing.T) {
	job := types.JobConfig{
		Name: "migrate",
		ContainerSpec: types.ContainerSpec{
			Environment: types.MappingWithEquals{"DB_HOST": strPtr("localhost"), "DB_PORT": strPtr("5432")},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "Environment", "DB_HOST=localhost")
	assertDirective(t, dirs, "Environment", "DB_PORT=5432")
}

func TestJobContainer_Volumes(t *testing.T) {
	job := types.JobConfig{
		Name: "backup",
		ContainerSpec: types.ContainerSpec{
			Volumes: []types.ServiceVolumeConfig{
				{Type: "volume", Source: "data", Target: "/data"},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "Volume", "data.volume:/data:rw")
}

func TestJobContainer_BindMount(t *testing.T) {
	job := types.JobConfig{
		Name: "job",
		ContainerSpec: types.ContainerSpec{
			Volumes: []types.ServiceVolumeConfig{
				{Type: "bind", Source: "/host/path", Target: "/container/path"},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "Mount", "type=bind,source=/host/path,destination=/container/path")
}

func TestJobContainer_User(t *testing.T) {
	job := types.JobConfig{Name: "job", ContainerSpec: types.ContainerSpec{User: "1000"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "User", "1000")
}

func TestJobContainer_WorkingDir(t *testing.T) {
	job := types.JobConfig{Name: "job", ContainerSpec: types.ContainerSpec{WorkingDir: "/app"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "WorkingDir", "/app")
}

func TestJobContainer_WorkingDir_Unavailable(t *testing.T) {
	job := types.JobConfig{Name: "job", ContainerSpec: types.ContainerSpec{WorkingDir: "/app"}}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 4, Minor: 5}
	dirs := JobContainer(job, cfg)
	if hasDirective(dirs, "WorkingDir", "/app") {
		t.Fatal("expected no WorkingDir directive at 4.5")
	}
	if !hasWarning(cfg, "job", "working_dir") {
		t.Fatal("expected WarningSkipped for working_dir")
	}
}

func TestJobContainer_Init(t *testing.T) {
	job := types.JobConfig{Name: "job", ContainerSpec: types.ContainerSpec{Init: boolPtrJob(true)}}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "RunInit", "true")
}

func TestJobContainer_ReadOnly(t *testing.T) {
	job := types.JobConfig{Name: "job", ContainerSpec: types.ContainerSpec{ReadOnly: true}}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "ReadOnly", "true")
}

func TestJobContainer_Labels(t *testing.T) {
	job := types.JobConfig{
		Name: "job",
		ContainerSpec: types.ContainerSpec{
			Labels: types.Labels{"env": "prod", "team": "infra"},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "Label", "env=prod")
	assertDirective(t, dirs, "Label", "team=infra")
}

func TestJobContainer_CapAdd(t *testing.T) {
	job := types.JobConfig{Name: "job", ContainerSpec: types.ContainerSpec{CapAdd: []string{"NET_ADMIN"}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "AddCapability", "NET_ADMIN")
}

func TestJobContainer_CapDrop(t *testing.T) {
	job := types.JobConfig{Name: "job", ContainerSpec: types.ContainerSpec{CapDrop: []string{"NET_RAW"}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "DropCapability", "NET_RAW")
}

func TestJobContainer_Ports(t *testing.T) {
	job := types.JobConfig{
		Name: "job",
		WorkloadSpec: types.WorkloadSpec{
			Ports: []types.ServicePortConfig{{Published: "8080", Target: 80}},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "PublishPort", "8080:80")
}

func TestJobContainer_NetworkMode_Host(t *testing.T) {
	job := types.JobConfig{Name: "job", ContainerSpec: types.ContainerSpec{NetworkMode: "host"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "Network", "host")
}

func TestJobContainer_NetworkMode_None(t *testing.T) {
	job := types.JobConfig{Name: "job", ContainerSpec: types.ContainerSpec{NetworkMode: "none"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "Network", "none")
}

func TestJobContainer_Networks(t *testing.T) {
	job := types.JobConfig{
		Name: "job",
		ContainerSpec: types.ContainerSpec{
			Networks: map[string]*types.ServiceNetworkConfig{"frontend": {}},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "Network", "frontend.network")
}

func TestJobContainer_Tty(t *testing.T) {
	job := types.JobConfig{Name: "job", WorkloadSpec: types.WorkloadSpec{Tty: true}}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--tty")
}

func TestJobContainer_Privileged(t *testing.T) {
	job := types.JobConfig{Name: "job", ContainerSpec: types.ContainerSpec{Privileged: true}}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobContainer(job, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--privileged")
}

func TestJobUnit_DependsOn(t *testing.T) {
	job := types.JobConfig{
		Name: "migrate",
		WorkloadSpec: types.WorkloadSpec{
			DependsOn: types.DependsOnConfig{
				"db": {Condition: "service_started", Required: true},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobUnit(job, cfg)
	assertDirective(t, dirs, "After", "db.container")
	assertDirective(t, dirs, "Requires", "db.container")
}

func TestJobUnit_NoDependsOn(t *testing.T) {
	job := types.JobConfig{Name: "migrate"}
	cfg := c2qtypes.DefaultConfig()
	dirs := JobUnit(job, cfg)
	if len(dirs) != 0 {
		t.Fatalf("expected no directives, got %v", dirs)
	}
}

func TestJobImages_Basic(t *testing.T) {
	jobs := types.Jobs{
		"migrate": types.JobConfig{
			Name:          "migrate",
			ContainerSpec: types.ContainerSpec{Image: "myapp:latest"},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := JobImages(jobs, cfg)

	if len(units) != 1 {
		t.Fatalf("expected 1 image unit, got %d", len(units))
	}

	u := units[0]
	if u.Type != c2qtypes.UnitImage {
		t.Fatalf("expected image unit type, got %s", u.Type)
	}
	if u.Name != "migrate" {
		t.Fatalf("expected image name 'migrate', got %s", u.Name)
	}

	dirs := u.Sections[0].Directives
	assertDirective(t, dirs, "Image", "docker.io/library/myapp:latest")
}

func TestJobImages_NoImage(t *testing.T) {
	jobs := types.Jobs{
		"build-only": types.JobConfig{
			Name: "build-only",
			WorkloadSpec: types.WorkloadSpec{
				Build: &types.BuildConfig{Context: "."},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := JobImages(jobs, cfg)

	if len(units) != 0 {
		t.Fatal("expected no image units for build-only job")
	}
}

func TestJobImages_PullPolicy(t *testing.T) {
	jobs := types.Jobs{
		"job": types.JobConfig{
			Name:          "job",
			ContainerSpec: types.ContainerSpec{Image: "myapp:latest", PullPolicy: "always"},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 6}
	units := JobImages(jobs, cfg)

	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "Policy", "always")
}

func TestJobImages_Platform(t *testing.T) {
	jobs := types.Jobs{
		"job": types.JobConfig{
			Name:          "job",
			ContainerSpec: types.ContainerSpec{Image: "myapp:latest", Platform: "linux/arm64"},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := JobImages(jobs, cfg)

	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "OS", "linux")
	assertDirective(t, dirs, "Arch", "arm64")
}

func TestJobBuilds_Basic(t *testing.T) {
	jobs := types.Jobs{
		"build-job": types.JobConfig{
			Name: "build-job",
			WorkloadSpec: types.WorkloadSpec{
				Build: &types.BuildConfig{
					Context:    "./app",
					Dockerfile: "Dockerfile.prod",
				},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 2}
	units := JobBuilds(jobs, cfg)

	if len(units) != 1 {
		t.Fatalf("expected 1 build unit, got %d", len(units))
	}

	u := units[0]
	if u.Type != c2qtypes.UnitBuild {
		t.Fatalf("expected build unit type, got %s", u.Type)
	}

	dirs := u.Sections[0].Directives
	assertDirective(t, dirs, "SetWorkingDirectory", "./app")
	assertDirective(t, dirs, "File", "Dockerfile.prod")
}

func TestJobBuilds_DefaultTag(t *testing.T) {
	jobs := types.Jobs{
		"job": types.JobConfig{
			Name: "job",
			WorkloadSpec: types.WorkloadSpec{
				Build: &types.BuildConfig{Context: "."},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 2}
	cfg.ProjectName = "myapp"
	units := JobBuilds(jobs, cfg)

	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "ImageTag", "myapp_job:latest")
}

func TestJobBuilds_CustomTags(t *testing.T) {
	jobs := types.Jobs{
		"job": types.JobConfig{
			Name: "job",
			WorkloadSpec: types.WorkloadSpec{
				Build: &types.BuildConfig{
					Context: ".",
					Tags:    []string{"myapp:prod", "myapp:latest"},
				},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 2}
	units := JobBuilds(jobs, cfg)

	dirs := units[0].Sections[0].Directives
	assertDirective(t, dirs, "ImageTag", "myapp:prod")
	assertDirective(t, dirs, "ImageTag", "myapp:latest")
}

func TestJobBuilds_NoBuild(t *testing.T) {
	jobs := types.Jobs{
		"image-only": types.JobConfig{
			Name:          "image-only",
			ContainerSpec: types.ContainerSpec{Image: "nginx:latest"},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	units := JobBuilds(jobs, cfg)

	if len(units) != 0 {
		t.Fatal("expected no build units for image-only job")
	}
}

func TestJobBuilds_Unavailable(t *testing.T) {
	jobs := types.Jobs{
		"job": types.JobConfig{
			Name: "job",
			WorkloadSpec: types.WorkloadSpec{
				Build: &types.BuildConfig{Context: "."},
			},
		},
	}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 4, Minor: 8}
	cfg.PseudoQuadlet = false
	units := JobBuilds(jobs, cfg)

	if len(units) != 0 {
		t.Fatal("expected no build units before 5.2")
	}
	if !hasWarning(cfg, "job", "build") {
		t.Fatal("expected fatal warning for build before 5.2")
	}
}
