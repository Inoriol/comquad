package compose2quadlet

import "github.com/Inoriol/comquad/compose2quadlet/internal/types"

type WarningLevel = types.WarningLevel

const (
	WarningSkipped  = types.WarningSkipped
	WarningDegraded = types.WarningDegraded
	WarningFatal    = types.WarningFatal
)

type Warning = types.Warning

type UnitType = types.UnitType

const (
	UnitContainer = types.UnitContainer
	UnitNetwork   = types.UnitNetwork
	UnitVolume    = types.UnitVolume
	UnitImage     = types.UnitImage
	UnitBuild     = types.UnitBuild
)

type QuadletUnit = types.QuadletUnit
type Section = types.Section
type Directive = types.Directive

const (
	SectionUnit      = types.SectionUnit
	SectionService   = types.SectionService
	SectionInstall   = types.SectionInstall
	SectionContainer = types.SectionContainer
	SectionNetwork   = types.SectionNetwork
	SectionVolume    = types.SectionVolume
	SectionImage     = types.SectionImage
	SectionBuild     = types.SectionBuild
)

type TranspileOption = types.Option
type Version = types.Version
