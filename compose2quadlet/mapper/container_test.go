package mapper

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	c2qtypes "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func TestContainer_Image(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Image: "nginx:latest"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Image", "web.image")
}

func TestContainer_ContainerName(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", ContainerName: "my-web"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "ContainerName", "my-web")
}

func TestContainer_Init(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Init: boolPtr(true)}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "RunInit", "true")
}

func TestContainer_InitNil(t *testing.T) {
	svc := types.ServiceConfig{Name: "s", Init: nil}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	if hasDirective(dirs, "RunInit", "true") {
		t.Fatal("should not emit RunInit when Init is nil")
	}
}

func TestContainer_InitFalse(t *testing.T) {
	svc := types.ServiceConfig{Name: "s", Init: boolPtr(false)}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	if hasDirective(dirs, "RunInit", "true") {
		t.Fatal("should not emit RunInit when Init is false")
	}
}

func TestContainer_ReadOnly(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", ReadOnly: true}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "ReadOnly", "true")
}

func TestContainer_User(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", User: "1000"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "User", "1000")
}

func TestContainer_Labels(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Labels: types.Labels{"env": "prod", "team": "infra"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Label", "env=prod")
	assertDirective(t, dirs, "Label", "team=infra")
}

func TestContainer_Annotations(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Annotations: types.Mapping{"foo": "bar"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Annotation", "foo=bar")
}

func TestContainer_CapAdd(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", CapAdd: []string{"NET_ADMIN", "SYS_PTRACE"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "AddCapability", "NET_ADMIN")
	assertDirective(t, dirs, "AddCapability", "SYS_PTRACE")
}

func TestContainer_CapDrop(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", CapDrop: []string{"NET_RAW"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "DropCapability", "NET_RAW")
}

func TestContainer_Expose(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Expose: types.StringOrNumberList{"80", "443"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "ExposeHostPort", "80")
	assertDirective(t, dirs, "ExposeHostPort", "443")
}

func TestContainer_GroupAdd(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", GroupAdd: []string{"999"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "GroupAdd", "999")
}

func TestContainer_GroupAdd_Unavailable(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", GroupAdd: []string{"999"}}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 4, Minor: 8}
	dirs := Container(svc, cfg)

	if len(dirs) > 0 {
		t.Fatal("expected no directives for group_add at 4.8")
	}
	if !hasWarning(cfg, "web", "group_add") {
		t.Fatal("expected WarningSkipped for group_add")
	}
}

func TestContainer_WorkingDir(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", WorkingDir: "/app"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "WorkingDir", "/app")
}

func TestContainer_ServiceName(t *testing.T) {
	svc := types.ServiceConfig{Name: "my-web"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "ServiceName", "my-web")
}

func TestContainer_SecurityOpt_NoNewPrivileges(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", SecurityOpt: []string{"no-new-privileges"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "NoNewPrivileges", "")
}

func TestContainer_SecurityOpt_LabelDisable(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", SecurityOpt: []string{"label=disable"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "SecurityLabelDisable", "")
}

func TestContainer_SecurityOpt_LabelNested(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", SecurityOpt: []string{"label=nested"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "SecurityLabelNested", "")
}

func TestContainer_SecurityOpt_Seccomp(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", SecurityOpt: []string{"seccomp=/etc/seccomp.json"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "SeccompProfile", "/etc/seccomp.json")
}

func TestContainer_SecurityOpt_LabelType(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", SecurityOpt: []string{"label=type:spc_t"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "SecurityLabelType", "spc_t")
}

func TestContainer_SecurityOpt_LabelLevel(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", SecurityOpt: []string{"label=level:s0:c1,c2"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "SecurityLabelLevel", "s0:c1,c2")
}

func TestContainer_SecurityOpt_LabelFiletype(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", SecurityOpt: []string{"label=filetype:container_t"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "SecurityLabelFileType", "container_t")
}

func TestContainer_SecurityOpt_Mask(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", SecurityOpt: []string{"mask=/proc/keys"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Mask", "/proc/keys")
}

func TestContainer_SecurityOpt_Unmask(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", SecurityOpt: []string{"unmask=all"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Unmask", "all")
}

func TestContainer_SecurityOpt_AppArmor(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", SecurityOpt: []string{"apparmor=my-profile"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "AppArmor", "my-profile")
}

func TestContainer_AppArmor_Unavailable(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", SecurityOpt: []string{"apparmor=my-profile"}}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 5}
	dirs := Container(svc, cfg)

	if hasDirective(dirs, "AppArmor", "my-profile") {
		t.Fatal("expected no AppArmor directive at 5.5")
	}
	if !hasWarning(cfg, "web", "security_opt") {
		t.Fatal("expected WarningSkipped for apparmor")
	}
}

func TestContainer_UserNS(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", UserNSMode: "keep-id"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "UserNS", "keep-id")
}

func TestContainer_Hostname(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Hostname: "myhost"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "HostName", "myhost")
}

func TestContainer_ShmSize(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", ShmSize: 67108864}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "ShmSize", "67108864")
}

func TestContainer_Sysctls(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Sysctls: types.Mapping{"net.core.somaxconn": "1024", "vm.overcommit": "1"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Sysctl", "net.core.somaxconn=1024")
	assertDirective(t, dirs, "Sysctl", "vm.overcommit=1")
}

func TestContainer_DNS(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", DNS: types.StringList{"8.8.8.8", "1.1.1.1"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "DNS", "8.8.8.8")
	assertDirective(t, dirs, "DNS", "1.1.1.1")
}

func TestContainer_DNSSearch(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", DNSSearch: types.StringList{"example.com", "internal"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "DNSSearch", "example.com")
	assertDirective(t, dirs, "DNSSearch", "internal")
}

func TestContainer_DNSOpt(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", DNSOpts: []string{"ndots:2", "timeout:1"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "DNSOption", "ndots:2")
	assertDirective(t, dirs, "DNSOption", "timeout:1")
}

func TestContainer_StopGracePeriod(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", StopGracePeriod: durationPtr(types.Duration(30_000_000_000))}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "StopTimeout", "30s")
}

func TestContainer_StopSignal(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", StopSignal: "SIGQUIT"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "StopSignal", "SIGQUIT")
}

func TestContainer_PullPolicy(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", PullPolicy: "always"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Pull", "always")
}

func TestContainer_VersionGatedWarnings(t *testing.T) {
	tests := []struct {
		name    string
		svc     types.ServiceConfig
		version c2qtypes.Version
		field   string
	}{
		{"working_dir <4.6", types.ServiceConfig{Name: "s", WorkingDir: "/app"}, c2qtypes.Version{Major: 4, Minor: 5}, "working_dir"},
		{"userns_mode <4.5", types.ServiceConfig{Name: "s", UserNSMode: "keep-id"}, c2qtypes.Version{Major: 4, Minor: 4}, "userns_mode"},
		{"hostname <4.6", types.ServiceConfig{Name: "s", Hostname: "h"}, c2qtypes.Version{Major: 4, Minor: 5}, "hostname"},
		{"shm_size <4.7", types.ServiceConfig{Name: "s", ShmSize: 1024}, c2qtypes.Version{Major: 4, Minor: 6}, "shm_size"},
		{"sysctls <4.6", types.ServiceConfig{Name: "s", Sysctls: types.Mapping{"a": "b"}}, c2qtypes.Version{Major: 4, Minor: 5}, "sysctls"},
		{"dns <4.7", types.ServiceConfig{Name: "s", DNS: types.StringList{"8.8.8.8"}}, c2qtypes.Version{Major: 4, Minor: 6}, "dns"},
		{"dnsopt <4.7", types.ServiceConfig{Name: "s", DNSOpts: []string{"a"}}, c2qtypes.Version{Major: 4, Minor: 6}, "dns_opt"},
		{"stop_grace_period <5.0", types.ServiceConfig{Name: "s", StopGracePeriod: durationPtr(types.Duration(10_000_000_000))}, c2qtypes.Version{Major: 4, Minor: 9}, "stop_grace_period"},
		{"stop_signal <5.2", types.ServiceConfig{Name: "s", StopSignal: "SIGINT"}, c2qtypes.Version{Major: 5, Minor: 1}, "stop_signal"},
		{"pull_policy <4.6", types.ServiceConfig{Name: "s", PullPolicy: "always"}, c2qtypes.Version{Major: 4, Minor: 5}, "pull_policy"},
		{"label=nested <4.6", types.ServiceConfig{Name: "s", SecurityOpt: []string{"label=nested"}}, c2qtypes.Version{Major: 4, Minor: 5}, "security_opt"},
		{"mask <4.6", types.ServiceConfig{Name: "s", SecurityOpt: []string{"mask=/proc"}}, c2qtypes.Version{Major: 4, Minor: 5}, "security_opt"},
		{"unmask <4.6", types.ServiceConfig{Name: "s", SecurityOpt: []string{"unmask=/proc"}}, c2qtypes.Version{Major: 4, Minor: 5}, "security_opt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := c2qtypes.DefaultConfig()
			cfg.PodmanVersion = tt.version
			dirs := Container(tt.svc, cfg)
			if len(dirs) > 0 {
				t.Fatal("expected no directives but got:", dirs)
			}
			if !hasWarning(cfg, "s", tt.field) {
				t.Fatal("expected WarningSkipped for", tt.field)
			}
		})
	}
}

func TestContainer_Command(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Command: types.ShellCommand{"npm", "start", "--port", "3000"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Exec", "'npm' 'start' '--port' '3000'")
}

func TestContainer_Environment(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Environment: types.MappingWithEquals{"NODE_ENV": strPtr("production"), "DEBUG": strPtr("app:*")}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Environment", "NODE_ENV=production")
	assertDirective(t, dirs, "Environment", "DEBUG=app:*")
}

func TestContainer_Environment_BareKey(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Environment: types.MappingWithEquals{"BAZ": nil}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Environment", "BAZ")
}

func TestContainer_Environment_BareKeyUnavailable(t *testing.T) {
	svc := types.ServiceConfig{Environment: types.MappingWithEquals{"BAZ": nil}}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 5}
	dirs := Container(svc, cfg)
	if hasDirective(dirs, "Environment", "BAZ") {
		t.Fatal("expected no bare key Environment directive at 5.5")
	}
	if !hasWarning(cfg, "", "environment") {
		t.Fatal("expected warning for bare key environment")
	}
}

func TestContainer_EnvFile(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", EnvFiles: []types.EnvFile{{Path: "/etc/env", Required: true}, {Path: ".env.local", Required: false}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "EnvironmentFile", "/etc/env")
	assertDirective(t, dirs, "EnvironmentFile", ".env.local")
}

func TestContainer_Tmpfs(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Tmpfs: types.StringList{"/run"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Tmpfs", "/run")
}

func TestContainer_Logging(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Logging: &types.LoggingConfig{Driver: "json-file", Options: types.Options{"max-size": "10m"}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "LogDriver", "json-file")
	assertDirective(t, dirs, "LogOpt", "max-size=10m")
}

func TestContainer_LoggingOptionsUnavailable(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Logging: &types.LoggingConfig{Options: types.Options{"max-size": "10m"}}}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 1}
	dirs := Container(svc, cfg)
	if hasDirective(dirs, "LogOpt", "max-size=10m") {
		t.Fatal("LogOpt should not be emitted before 5.2")
	}
	if !hasWarning(cfg, "web", "logging.options") {
		t.Fatal("expected warning for LogOpt at <5.2")
	}
}

func TestContainer_LogDriverFallback(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", LogDriver: "syslog"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "LogDriver", "syslog")
}

func TestContainer_PidsLimit(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", PidsLimit: 100}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PidsLimit", "100")
}

func TestContainer_Ulimits(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Ulimits: map[string]*types.UlimitsConfig{"nofile": {Soft: 1024, Hard: 2048}, "nproc": {Single: 65535}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertNoDirective(t, dirs, "Ulimit")
}

func TestContainer_ExtraHosts(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", ExtraHosts: types.HostsList{"host.internal": {"10.0.0.1"}, "db": {"192.168.1.5", "192.168.1.6"}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "AddHost", "host.internal:10.0.0.1")
	assertDirective(t, dirs, "AddHost", "db:192.168.1.5")
	assertDirective(t, dirs, "AddHost", "db:192.168.1.6")
}

func TestContainer_Volumes(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Volumes: []types.ServiceVolumeConfig{{Type: "bind", Source: "/host/data", Target: "/container/data"}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Mount", "type=bind,source=/host/data,destination=/container/data")
}

func TestContainer_Volumes_ReadOnly(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Volumes: []types.ServiceVolumeConfig{{Type: "bind", Source: "/host/data", Target: "/app", ReadOnly: true}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Mount", "type=bind,source=/host/data,destination=/app,readonly")
}

func TestContainer_Volumes_VolumeType(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Volumes: []types.ServiceVolumeConfig{{Type: "volume", Source: "myvol", Target: "/data"}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Volume", "myvol.volume:/data:rw")
}

func TestContainer_Ports_ShortSyntax(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Ports: []types.ServicePortConfig{{Published: "8080", Target: 80}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PublishPort", "8080:80")
}

func TestContainer_Ports_WithProtocol(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Ports: []types.ServicePortConfig{{Published: "53", Target: 53, Protocol: "udp"}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PublishPort", "53:53/udp")
}

func TestContainer_Ports_WithHostIP(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Ports: []types.ServicePortConfig{{HostIP: "127.0.0.1", Published: "8080", Target: 80}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PublishPort", "127.0.0.1:8080:80")
}

func TestContainer_Ports_TCPOmitted(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Ports: []types.ServicePortConfig{{Published: "8080", Target: 80, Protocol: "tcp"}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PublishPort", "8080:80")
}

func TestContainer_NetworkMode_Host(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", NetworkMode: "host"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Network", "host")
}

func TestContainer_NetworkMode_None(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", NetworkMode: "none"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Network", "none")
}

func TestContainer_NetworkMode_Service(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", NetworkMode: "service:db"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Network", "container:db.container")
}

func TestContainer_NetworkMode_Service_Unavailable(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", NetworkMode: "service:db"}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 2}
	dirs := Container(svc, cfg)
	if hasDirective(dirs, "Network", "container:db.container") {
		t.Fatal("should not emit Network=container: before 5.3")
	}
	if !hasWarning(cfg, "web", "network_mode") {
		t.Fatal("expected WarningSkipped for network_mode: service")
	}
}

func TestContainer_Networks(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Networks: map[string]*types.ServiceNetworkConfig{"frontend": {}, "backend": {}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Network", "frontend.network")
	assertDirective(t, dirs, "Network", "backend.network")
}

func TestContainer_NetworksSortedOrder(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Networks: map[string]*types.ServiceNetworkConfig{
		"znet":   {},
		"anet":   {},
		"midnet": {},
	}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)

	var networks []string
	for _, d := range dirs {
		if d.Key == "Network" {
			networks = append(networks, d.Values...)
		}
	}
	want := []string{"anet.network", "midnet.network", "znet.network"}
	if len(networks) != len(want) {
		t.Fatalf("expected %d networks, got %v", len(want), networks)
	}
	for i := range want {
		if networks[i] != want[i] {
			t.Fatalf("expected sorted networks %v, got %v", want, networks)
		}
	}
}

func TestContainer_NetworkAliases(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Networks: map[string]*types.ServiceNetworkConfig{"frontend": {Aliases: []string{"app", "www"}}}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "NetworkAlias", "app:frontend")
	assertDirective(t, dirs, "NetworkAlias", "www:frontend")
}

func TestContainer_Entrypoint_P1(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Entrypoint: types.ShellCommand{"npm", "start"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Entrypoint", "npm start")
}

func TestContainer_Entrypoint_P3Fallback(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Entrypoint: types.ShellCommand{"/custom-init", "--verbose"}}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 4, Minor: 8}
	dirs := Container(svc, cfg)

	assertDirective(t, dirs, "PodmanArgs", "--entrypoint /custom-init --verbose")
	if !hasWarning(cfg, "web", "entrypoint") {
		t.Fatal("expected WarningDegraded for entrypoint at 4.8")
	}
}

func TestContainer_Entrypoint_Noop(t *testing.T) {
	svc := types.ServiceConfig{Name: "web"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	if hasDirective(dirs, "Entrypoint", "") || hasDirective(dirs, "PodmanArgs", "") {
		t.Fatal("should not emit entrypoint directives when field is empty")
	}
}

func TestContainer_Memory(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", MemLimit: 536870912}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Memory", "536870912")
}

func TestContainer_Memory_Unavailable(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", MemLimit: 536870912}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 4}
	dirs := Container(svc, cfg)
	if hasDirective(dirs, "Memory", "536870912") {
		t.Fatal("should not emit Memory= before 5.5")
	}
}

func TestContainer_Cgroup_Host(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Cgroup: "host"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "CgroupsMode", "host")
}

func TestContainer_Cgroup_Host_Unavailable(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Cgroup: "host"}
	cfg := c2qtypes.DefaultConfig()
	cfg.PodmanVersion = c2qtypes.Version{Major: 5, Minor: 2}
	dirs := Container(svc, cfg)
	if hasDirective(dirs, "CgroupsMode", "host") {
		t.Fatal("should not emit CgroupsMode before 5.3")
	}
	if !hasWarning(cfg, "web", "cgroup") {
		t.Fatal("expected WarningSkipped for cgroup host at 5.2")
	}
}

func TestContainer_Cgroup_Private(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Cgroup: "private"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--cgroupns private")
}

func TestContainer_BindMount(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Volumes: []types.ServiceVolumeConfig{
		{Type: "bind", Source: "/host/app", Target: "/app"},
	}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Mount", "type=bind,source=/host/app,destination=/app")
}

func TestContainer_BindMount_Propagation(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Volumes: []types.ServiceVolumeConfig{
		{Type: "bind", Source: "/host/shared", Target: "/shared", Bind: &types.ServiceVolumeBind{Propagation: "rshared"}},
	}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Mount", "type=bind,source=/host/shared,destination=/shared,bind-propagation=rshared")
}

func TestContainer_BindMount_SELinux(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Volumes: []types.ServiceVolumeConfig{
		{Type: "bind", Source: "/host/data", Target: "/data", Bind: &types.ServiceVolumeBind{SELinux: "z"}},
	}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Mount", "type=bind,source=/host/data,destination=/data,relabel=shared")
}

func TestContainer_Tmpfs_LongSyntax(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Volumes: []types.ServiceVolumeConfig{
		{Type: "tmpfs", Target: "/run", Tmpfs: &types.ServiceVolumeTmpfs{Size: 67108864, Mode: 1700}},
	}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "Tmpfs", "/run:size=67108864,mode=3244")
}

func TestContainer_Tty(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Tty: true}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--tty")
}

func TestContainer_StdinOpen(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", StdinOpen: true}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--attach stdin")
}

func TestContainer_Runtime(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Runtime: "crun"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "GlobalArgs", "--runtime crun")
}

func TestContainer_MacAddress(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", MacAddress: "02:42:ac:11:00:02"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--mac-address 02:42:ac:11:00:02")
}

func TestContainer_MacAddress_PerNetwork(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Networks: map[string]*types.ServiceNetworkConfig{
		"frontend": {MacAddress: "02:42:ac:11:00:03"},
	}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--mac-address 02:42:ac:11:00:03")
	assertDirective(t, dirs, "Network", "frontend.network")
}

func TestContainer_IpcShareable(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Ipc: "shareable"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--ipc shareable")
}

func TestContainer_PidHost(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Pid: "host"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--pid host")
}

func TestContainer_UtsHost(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Uts: "host"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--uts host")
}

func TestContainer_Privileged(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Privileged: true}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--privileged")
}

func TestContainer_MemSwappiness(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", MemSwappiness: 60}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--memory-swappiness 60")
}

func TestContainer_CpuRtRuntime(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", CPURTRuntime: 50000}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--cpu-rt-runtime 50000")
}

func TestContainer_CpuRtPeriod(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", CPURTPeriod: 1000000}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--cpu-rt-period 1000000")
}

func TestContainer_DeviceCgroupRules(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", DeviceCgroupRules: []string{"c 13:* rwm", "b 7:* rmw"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--device-cgroup-rule c 13:* rwm")
	assertDirective(t, dirs, "PodmanArgs", "--device-cgroup-rule b 7:* rmw")
}

func TestContainer_StorageOpt(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", StorageOpt: map[string]string{"size": "10G", "override": "true"}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "GlobalArgs", "--storage-opt size=10G")
	assertDirective(t, dirs, "GlobalArgs", "--storage-opt override=true")
}

func TestContainer_OomKillDisable_P3(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", OomKillDisable: true}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--oom-kill-disable")
}

func TestContainer_Devices_HostContainerPerms(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Devices: []types.DeviceMapping{
		{Source: "/dev/ttyUSB0", Target: "/dev/ttyUSB0", Permissions: "rwm"},
	}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "AddDevice", "/dev/ttyUSB0:/dev/ttyUSB0:rwm")
}

func TestContainer_Devices_NoPerms(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Devices: []types.DeviceMapping{
		{Source: "/dev/sda", Target: "/dev/sda"},
	}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "AddDevice", "/dev/sda:/dev/sda")
}

func TestContainer_Devices_CDI(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Devices: []types.DeviceMapping{
		{Source: "nvidia.com/gpu=all"},
	}}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "AddDevice", "nvidia.com/gpu=all")
}

func TestContainer_CgroupParent_P3(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", CgroupParent: "myapp.slice"}
	cfg := c2qtypes.DefaultConfig()
	dirs := Container(svc, cfg)
	assertDirective(t, dirs, "PodmanArgs", "--cgroup-parent myapp.slice")
}
