package compose2quadlet_test

import (
	"testing"

	c2q "github.com/Inoriol/comquad/compose2quadlet"
)

func TestTranspile_SimpleWeb(t *testing.T) {
	project := loadProject(t, "testdata/simple-web.yaml")
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
	if !hasDirectiveValue(sec.Directives, "Image", "test-web.image") {
		t.Fatal("expected Image=test-web.image")
	}
	if !hasDirectiveValue(sec.Directives, "PublishPort", "8080:80") {
		t.Fatal("expected PublishPort=8080:80")
	}
	_, ok = findUnit(units, "test-web", c2q.UnitImage)
	if !ok {
		t.Fatal("expected test-web.image unit")
	}
}

func TestTranspile_MultiService(t *testing.T) {
	project := loadProject(t, "testdata/multi-service.yaml")
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
	web, ok := findUnit(units, "test-web", c2q.UnitContainer)
	if !ok {
		t.Fatal("expected test-web.container")
	}
	unitSec, ok := hasSection(web, c2q.SectionUnit)
	if !ok {
		t.Fatal("expected [Unit] section in web")
	}
	if !hasDirectiveValue(unitSec.Directives, "Requires", "test-db.container") {
		t.Fatal("expected Requires=test-db.container")
	}

	db, ok := findUnit(units, "test-db", c2q.UnitContainer)
	if !ok {
		t.Fatal("expected test-db.container")
	}
	containerSec, ok := hasSection(db, c2q.SectionContainer)
	if !ok {
		t.Fatal("expected [Container] section in db")
	}
	if !hasDirectiveValue(containerSec.Directives, "Environment", "POSTGRES_PASSWORD=secret") {
		t.Fatal("expected Environment=POSTGRES_PASSWORD=secret")
	}
	if !hasDirectiveValue(containerSec.Directives, "HealthCmd", "pg_isready") {
		t.Fatal("expected HealthCmd=pg_isready")
	}
}

func TestTranspile_TopLevelOnly(t *testing.T) {
	project := loadProject(t, "testdata/top-level.yaml")
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
	_, ok := findUnit(units, "test-backend", c2q.UnitNetwork)
	if !ok {
		t.Fatal("expected test-backend.network")
	}
	_, ok = findUnit(units, "test-frontend", c2q.UnitNetwork)
	if !ok {
		t.Fatal("expected test-frontend.network")
	}
	_, ok = findUnit(units, "test-db_data", c2q.UnitVolume)
	if !ok {
		t.Fatal("expected test-db_data.volume")
	}
}

func TestTranspile_EdgeCases_BuildService(t *testing.T) {
	project := loadProject(t, "testdata/edge-cases.yaml")
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
	worker, ok := findUnit(units, "test-worker", c2q.UnitContainer)
	if !ok {
		t.Fatal("expected test-worker.container")
	}
	sec, ok := hasSection(worker, c2q.SectionContainer)
	if !ok {
		t.Fatal("expected [Container] section")
	}
	if !hasDirectiveValue(sec.Directives, "Image", "test-worker.build") {
		t.Fatal("expected Image=test-worker.build")
	}
	if !hasDirectiveValue(sec.Directives, "PodmanArgs", "--tty") {
		t.Fatal("expected PodmanArgs=--tty")
	}
	if !hasDirectiveValue(sec.Directives, "PodmanArgs", "--attach stdin") {
		t.Fatal("expected PodmanArgs=--attach stdin")
	}
	_, ok = findUnit(units, "test-worker", c2q.UnitBuild)
	if !ok {
		t.Fatal("expected test-worker.build unit")
	}
}

func TestTranspile_OptionCombinatorics(t *testing.T) {
	project := loadProject(t, "testdata/simple-web.yaml")

	t.Run("with_install_and_autoupdate", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithoutPrefix(),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithAutoUpdate(),
		)
		if err != nil {
			t.Fatal(err)
		}
		unit, ok := findUnit(units, "test-web", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected test-web.container unit")
		}
		_, ok = hasSection(unit, c2q.SectionInstall)
		if !ok {
			t.Fatal("expected [Install] section")
		}
		sec, ok := hasSection(unit, c2q.SectionContainer)
		if !ok {
			t.Fatal("expected [Container] section")
		}
		if !hasDirectiveValue(sec.Directives, "AutoUpdate", "registry") {
			t.Fatal("expected AutoUpdate=registry")
		}
	})

	t.Run("with_selinux_and_default_network", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithoutPrefix(),
			c2q.WithoutInstallSection(),
			c2q.WithoutNetworkAliases(),
		)
		if err != nil {
			t.Fatal(err)
		}
		// default network should be injected
		_, ok := findUnit(units, "test-default", c2q.UnitNetwork)
		if !ok {
			t.Fatal("expected default network")
		}
	})

	t.Run("with_port_offset", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithPortOffset(1000),
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
		if !hasDirectiveValue(sec.Directives, "PublishPort", "8080:80") {
			t.Fatal("expected PublishPort=8080:80")
		}
	})

	t.Run("with_default_prefix", func(t *testing.T) {
		units, err := c2q.Transpile(project,
			c2q.WithProjectName("test"),
			c2q.WithoutDefaultNetwork(),
			c2q.WithoutSELinux(),
			c2q.WithoutNetworkAliases(),
			c2q.WithoutInstallSection(),
		)
		if err != nil {
			t.Fatal(err)
		}
		_, ok := findUnit(units, "cq-test-web", c2q.UnitContainer)
		if !ok {
			t.Fatal("expected cq-test-web.container with default prefix")
		}
	})
}

func TestTranspile_ExternalVolumesSkipped(t *testing.T) {
	project := loadProject(t, "testdata/top-level.yaml")

	project.Volumes["db_data"] = project.Volumes["db_data"]

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
	_, ok := findUnit(units, "test-db_data", c2q.UnitVolume)
	if !ok {
		t.Fatal("expected test-db_data.volume")
	}
}
