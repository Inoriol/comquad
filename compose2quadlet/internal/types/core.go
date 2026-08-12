package types

type WarningLevel int

const (
	WarningSkipped  WarningLevel = iota
	WarningDegraded
	WarningFatal
)

type Warning struct {
	Level    WarningLevel
	Service  string
	Field    string
	Message  string
	Since    string
}

type UnitType string

const (
	UnitContainer UnitType = "container"
	UnitNetwork   UnitType = "network"
	UnitVolume    UnitType = "volume"
	UnitImage     UnitType = "image"
	UnitBuild     UnitType = "build"
)

type QuadletUnit struct {
	Type     UnitType
	Name     string
	Sections []Section
}

type Section struct {
	Name       string
	Directives []Directive
}

type Directive struct {
	Key    string
	Values []string
}

const (
	SectionUnit      = "Unit"
	SectionService   = "Service"
	SectionInstall   = "Install"
	SectionContainer = "Container"
	SectionNetwork   = "Network"
	SectionVolume    = "Volume"
	SectionImage     = "Image"
	SectionBuild     = "Build"
)
