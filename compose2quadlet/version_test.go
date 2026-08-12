package compose2quadlet_test

import (
	"strings"
	"testing"

	c2q "github.com/Inoriol/comquad/compose2quadlet"
)

func TestVersion_Entrypoint_P3toP1(t *testing.T) {
	project := loadProject(t, "testdata/version-basic.yaml")

	t.Run("podman_4_8_fallback", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPodmanVersion(c2q.Version{Major: 4, Minor: 8}),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-web", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-web.container unit")
		}
		sec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if !hasDirectiveValue(sec.Directives, "PodmanArgs", "--entrypoint /custom-init --verbose") {
			t.Fatal("expected PodmanArgs fallback for entrypoint at 4.8")
		}
		if hasDirectiveValue(sec.Directives, "Entrypoint", "/custom-init --verbose") {
			t.Fatal("expected no P1 Entrypoint at 4.8")
		}
	})

	t.Run("podman_5_0_native", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPodmanVersion(c2q.Version{Major: 5, Minor: 0}),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-web", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-web.container unit")
		}
		sec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if !hasDirectiveValue(sec.Directives, "Entrypoint", "/custom-init --verbose") {
			t.Fatal("expected P1 Entrypoint at 5.0")
		}
		if hasDirectiveValue(sec.Directives, "PodmanArgs", "--entrypoint /custom-init --verbose") {
			t.Fatal("expected no PodmanArgs fallback at 5.0")
		}
	})

	t.Run("podman_latest_native", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-web", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-web.container unit")
		}
		sec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if !hasDirectiveValue(sec.Directives, "Entrypoint", "/custom-init --verbose") {
			t.Fatal("expected P1 Entrypoint at latest")
		}
	})
}

func TestVersion_StopSignal_Gate(t *testing.T) {
	project := loadProject(t, "testdata/version-basic.yaml")

	t.Run("podman_5_1_skipped", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPodmanVersion(c2q.Version{Major: 5, Minor: 1}),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-web", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-web.container unit")
		}
		sec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if hasDirectiveValue(sec.Directives, "StopSignal", "SIGQUIT") {
			t.Fatal("expected no StopSignal before 5.2")
		}
	})

	t.Run("podman_5_2_available", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPodmanVersion(c2q.Version{Major: 5, Minor: 2}),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-web", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-web.container unit")
		}
		sec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if !hasDirectiveValue(sec.Directives, "StopSignal", "SIGQUIT") {
			t.Fatal("expected StopSignal at 5.2")
		}
	})
}

func TestVersion_NetworkAliases_Gate(t *testing.T) {
	project := loadProject(t, "testdata/version-features.yaml")

	t.Run("podman_5_1_skipped", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPodmanVersion(c2q.Version{Major: 5, Minor: 1}),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-app", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-app.container unit")
		}
		sec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if hasDirectiveValue(sec.Directives, "NetworkAlias", "www:frontend") {
			t.Fatal("expected no NetworkAlias before 5.2")
		}
	})

	t.Run("podman_5_2_available", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPodmanVersion(c2q.Version{Major: 5, Minor: 2}),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-app", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-app.container unit")
		}
		sec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if !hasDirectiveValue(sec.Directives, "NetworkAlias", "www:frontend") {
			t.Fatal("expected NetworkAlias at 5.2")
		}
	})
}

func TestVersion_LogOptions_Gate(t *testing.T) {
	project := loadProject(t, "testdata/version-features.yaml")

	t.Run("podman_5_1_skipped", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPodmanVersion(c2q.Version{Major: 5, Minor: 1}),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-app", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-app.container unit")
		}
		sec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if hasDirectiveValue(sec.Directives, "LogOpt", "max-size=10m") {
			t.Fatal("expected no LogOpt before 5.2")
		}
	})

	t.Run("podman_5_2_available", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPodmanVersion(c2q.Version{Major: 5, Minor: 2}),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-app", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-app.container unit")
		}
		sec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if !hasDirectiveValue(sec.Directives, "LogOpt", "max-size=10m") {
			t.Fatal("expected LogOpt at 5.2")
		}
	})
}

func TestVersion_ExtraHosts_Gate(t *testing.T) {
	project := loadProject(t, "testdata/version-features.yaml")

	t.Run("podman_5_2_skipped", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPodmanVersion(c2q.Version{Major: 5, Minor: 2}),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-app", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-app.container unit")
		}
		sec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if hasDirectiveValue(sec.Directives, "AddHost", "host.internal:10.0.0.1") {
			t.Fatal("expected no AddHost before 5.3")
		}
	})

	t.Run("podman_5_3_available", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPodmanVersion(c2q.Version{Major: 5, Minor: 3}),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-app", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-app.container unit")
		}
		sec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if !hasDirectiveValue(sec.Directives, "AddHost", "host.internal:10.0.0.1") {
			t.Fatal("expected AddHost at 5.3")
		}
	})
}

func TestVersion_MemorySectionSwitch(t *testing.T) {
	project := loadProject(t, "testdata/version-memory.yaml")

	t.Run("podman_5_4_service_section", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPodmanVersion(c2q.Version{Major: 5, Minor: 4}),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-db", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-db.container unit")
		}
		sec, ok := hasSection(unit, c2q.SectionService)
		if !ok {
			t.Fatal("expected [Service] section at 5.4")
		}
		if !hasDirectiveValue(sec.Directives, "MemoryMax", "536870912") {
			t.Fatal("expected MemoryMax in [Service] at 5.4")
		}
		containerSec, _ := hasSection(unit, c2q.SectionContainer)
		if hasDirectiveValue(containerSec.Directives, "Memory", "536870912") {
			t.Fatal("expected no Memory in [Container] at 5.4")
		}
	})

	t.Run("podman_5_5_container_section", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPodmanVersion(c2q.Version{Major: 5, Minor: 5}),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-db", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-db.container unit")
		}
		containerSec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if !hasDirectiveValue(containerSec.Directives, "Memory", "536870912") {
			t.Fatal("expected Memory in [Container] at 5.5")
		}
		serviceSec, _ := hasSection(unit, c2q.SectionService)
		if hasDirectiveValue(serviceSec.Directives, "MemoryMax", "536870912") {
			t.Fatal("expected no MemoryMax in [Service] at 5.5")
		}
	})
}

func TestVersion_Build_FatalError(t *testing.T) {
	project := loadProject(t, "testdata/version-build.yaml")

	_, err := c2q.Transpile(project,
		c2q.WithProjectName("test"),
		c2q.WithPodmanVersion(c2q.Version{Major: 4, Minor: 8}),
		c2q.WithoutPrefix(),
		c2q.WithoutDefaultNetwork(),
		c2q.WithoutSELinux(),
		c2q.WithoutNetworkAliases(),
		c2q.WithoutInstallSection(),
	)
	if err == nil {
		t.Fatal("expected fatal error for build at podman 4.8")
	}
	if !strings.Contains(err.Error(), "5.2.0") {
		t.Fatalf("expected error mentioning 5.2.0, got: %v", err)
	}
}

func TestVersion_Build_Available(t *testing.T) {
	project := loadProject(t, "testdata/version-build.yaml")

	units, err := c2q.Transpile(project,
		c2q.WithProjectName("test"),
		c2q.WithPodmanVersion(c2q.Version{Major: 5, Minor: 2}),
		c2q.WithoutPrefix(),
		c2q.WithoutDefaultNetwork(),
		c2q.WithoutSELinux(),
		c2q.WithoutNetworkAliases(),
		c2q.WithoutInstallSection(),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := findUnit(units, "test-app", c2q.UnitBuild)
	if !ok {
		t.Fatal("expected test-app.build unit at 5.2")
	}
	_, ok = findUnit(units, "test-app", c2q.UnitContainer)
	if !ok {
		t.Fatal("expected test-app.container unit at 5.2")
	}
}

func TestVersion_WarningCollectionSmoke(t *testing.T) {
	project := loadProject(t, "testdata/version-basic.yaml")

	units, err := c2q.Transpile(project,
		c2q.WithProjectName("test"),
		c2q.WithPodmanVersion(c2q.Version{Major: 4, Minor: 8}),
		c2q.WithoutPrefix(),
		c2q.WithoutDefaultNetwork(),
		c2q.WithoutSELinux(),
		c2q.WithoutNetworkAliases(),
		c2q.WithoutInstallSection(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(units) == 0 {
		t.Fatal("expected at least a container unit")
	}
}
