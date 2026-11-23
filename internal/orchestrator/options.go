package orchestrator

type UpOptions struct {
	PullStrategy   string
	Follow         bool
	DryRun         bool
	NoDiff         bool
	ForceRebuild   bool
	SkipBuild      bool
	ForceRecreate  bool
	SkipRecreate   bool
	RemoveOrphans  bool
	Services       []string
}

type DownOptions struct {
	RemoveVolumes bool
	DryRun        bool
	RemoveOrphans bool
}

type PullOptions struct {
	PullStrategy string
	DryRun       bool
	Services     []string
}

type RunOptions struct {
	JobName string
	Follow  bool
	DryRun  bool
}
