package reconcile

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	c2q "github.com/Inoriol/comquad/compose2quadlet"
	"github.com/Inoriol/comquad/compose2quadlet/serialization"
)

// Conflict describes a directive changed by both the user (via `comquad edit`)
// and by compose.yaml regeneration. The user's value wins; the conflict is
// reported so nothing is silently dropped.
type Conflict struct {
	Unit      string
	Section   string
	Key       string
	User      string
	Generated string
}

// Result summarizes a Reconcile pass.
type Result struct {
	Created    []string // files that did not exist and were written
	Changed    []string // existing files whose content changed and were rewritten
	Removed    []string // stale target files removed (no longer generated)
	NoBaseline []string // files overwritten without a baseline (2-way fallback)
	Conflicts  []Conflict
}

// FileStatus classifies how a generated unit relates to what is on disk.
type FileStatus int

const (
	StatusUnchanged FileStatus = iota
	StatusCreated
	StatusChanged
	StatusRemoved
)

// FilePlan describes what will happen to a single quadlet file.
type FilePlan struct {
	Name            string
	TargetPath      string
	BasePath        string
	Status          FileStatus
	OldContent      string
	NewContent      string
	BaselineContent string
	NoBaseline      bool
	Conflicts       []Conflict
}

// Plan is a read-only preview of a reconcile pass.
type Plan struct {
	Files      []FilePlan
	Conflicts  []Conflict
	NoBaseline []string
}

// HasChanges reports whether applying the plan would modify anything.
func (p Plan) HasChanges() bool {
	for _, f := range p.Files {
		if f.Status == StatusCreated || f.Status == StatusChanged || f.Status == StatusRemoved {
			return true
		}
	}
	return false
}

const removedSentinel = "(removed)"

// MergeUnit performs a directive-level three-way merge of a generated quadlet
// unit and its on-disk counterpart, using the previously generated content as
// the common base. On conflict the user's (disk) value wins and the conflict is
// returned. The merged unit always adopts the type and name of new.
func MergeUnit(base, disk, new c2q.QuadletUnit) (c2q.QuadletUnit, []Conflict) {
	unitName := new.Name + "." + string(new.Type)

	_, baseSec := normalize(base)
	diskNames, diskSec := normalize(disk)
	newNames, newSec := normalize(new)

	result := c2q.QuadletUnit{Type: new.Type, Name: new.Name}
	var conflicts []Conflict

	order := sectionOrder(newNames, diskNames)
	for _, name := range order {
		keys, vals, secConflicts := mergeSection(unitName, name, baseSec[name], diskSec[name], newSec[name])
		conflicts = append(conflicts, secConflicts...)
		if len(keys) > 0 {
			result.Sections = append(result.Sections, c2q.Section{Name: name, Directives: directivesFrom(keys, vals)})
		}
	}

	return result, conflicts
}

// Compute builds a Plan describing the changes needed without touching disk.
func Compute(targetDir, baselineDir, prefix string, units []c2q.QuadletUnit) (Plan, error) {
	var plan Plan
	expected := make(map[string]bool, len(units))

	for _, u := range units {
		filename := u.Name + "." + string(u.Type)
		expected[filename] = true

		newContent := serialization.Marshal(u)
		fp := FilePlan{
			Name:            filename,
			TargetPath:      filepath.Join(targetDir, filename),
			BasePath:        filepath.Join(baselineDir, filename),
			BaselineContent: newContent,
		}

		diskContent, diskExists, err := readFile(fp.TargetPath)
		if err != nil {
			return plan, err
		}
		baseContent, baseExists, err := readFile(fp.BasePath)
		if err != nil {
			return plan, err
		}
		fp.OldContent = diskContent

		switch {
		case !diskExists:
			fp.Status = StatusCreated
			fp.NewContent = newContent
		case !baseExists:
			fp.NewContent = newContent
			if diskContent != newContent {
				fp.Status = StatusChanged
				fp.NoBaseline = true
			}
		default:
			base := serialization.Unmarshal(baseContent, u.Type, u.Name)
			disk := serialization.Unmarshal(diskContent, u.Type, u.Name)
			merged, conflicts := MergeUnit(base, disk, u)
			fp.Conflicts = conflicts
			fp.NewContent = serialization.Marshal(merged)
			if fp.NewContent != diskContent {
				fp.Status = StatusChanged
			}
		}

		plan.Files = append(plan.Files, fp)
		plan.Conflicts = append(plan.Conflicts, fp.Conflicts...)
		if fp.NoBaseline {
			plan.NoBaseline = append(plan.NoBaseline, fp.TargetPath)
		}
	}

	stale := findStaleNames(targetDir, prefix, expected)
	stale = append(stale, findStaleNames(baselineDir, prefix, expected)...)
	for _, name := range dedupe(stale) {
		old, _, err := readFile(filepath.Join(targetDir, name))
		if err != nil {
			return plan, err
		}
		plan.Files = append(plan.Files, FilePlan{
			Name:       name,
			TargetPath: filepath.Join(targetDir, name),
			BasePath:   filepath.Join(baselineDir, name),
			Status:     StatusRemoved,
			OldContent: old,
		})
	}

	sort.Slice(plan.Files, func(i, j int) bool { return plan.Files[i].Name < plan.Files[j].Name })

	return plan, nil
}

// Apply executes a Plan, writing files, updating baselines, and removing stale
// entries. It returns a Result summarizing what changed.
func Apply(targetDir, baselineDir string, plan Plan) (Result, error) {
	var res Result
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return res, err
	}
	if err := os.MkdirAll(baselineDir, 0755); err != nil {
		return res, err
	}

	for _, fp := range plan.Files {
		switch fp.Status {
		case StatusCreated:
			if err := writeFileAtomic(fp.TargetPath, fp.NewContent); err != nil {
				return res, err
			}
			res.Created = append(res.Created, fp.TargetPath)
		case StatusChanged:
			if err := writeFileAtomic(fp.TargetPath, fp.NewContent); err != nil {
				return res, err
			}
			res.Changed = append(res.Changed, fp.TargetPath)
		case StatusRemoved:
			if err := os.Remove(fp.TargetPath); err != nil && !os.IsNotExist(err) {
				return res, err
			}
			if err := os.Remove(fp.BasePath); err != nil && !os.IsNotExist(err) {
				return res, err
			}
			res.Removed = append(res.Removed, fp.TargetPath)
			continue
		}

		if fp.NoBaseline {
			res.NoBaseline = append(res.NoBaseline, fp.TargetPath)
		}
		res.Conflicts = append(res.Conflicts, fp.Conflicts...)
		if err := writeFileAtomic(fp.BasePath, fp.BaselineContent); err != nil {
			return res, err
		}
	}

	return res, nil
}

// Reconcile computes and applies changes in a single step.
func Reconcile(targetDir, baselineDir, prefix string, units []c2q.QuadletUnit) (Result, error) {
	plan, err := Compute(targetDir, baselineDir, prefix, units)
	if err != nil {
		return Result{}, err
	}
	return Apply(targetDir, baselineDir, plan)
}

// section is a normalized view of a unit section: ordered directive keys with
// their flattened (concatenated) values. A key present with zero values is a
// boolean flag directive.
type section struct {
	keys []string
	vals map[string][]string
}

func normalize(u c2q.QuadletUnit) ([]string, map[string]*section) {
	names := make([]string, 0, len(u.Sections))
	sections := make(map[string]*section, len(u.Sections))
	for _, s := range u.Sections {
		sec, ok := sections[s.Name]
		if !ok {
			sec = &section{vals: make(map[string][]string)}
			sections[s.Name] = sec
			names = append(names, s.Name)
		}
		for _, d := range s.Directives {
			if _, ok := sec.vals[d.Key]; !ok {
				sec.keys = append(sec.keys, d.Key)
				sec.vals[d.Key] = []string{}
			}
			sec.vals[d.Key] = append(sec.vals[d.Key], d.Values...)
		}
	}
	return names, sections
}

func sectionOrder(newNames, diskNames []string) []string {
	var order []string
	seen := make(map[string]bool)
	for _, n := range newNames {
		order = append(order, n)
		seen[n] = true
	}
	for _, n := range diskNames {
		if !seen[n] {
			order = append(order, n)
			seen[n] = true
		}
	}
	return order
}

func mergeSection(unitName, name string, base, disk, new *section) ([]string, map[string][]string, []Conflict) {
	if disk == nil {
		disk = &section{}
	}
	if new == nil {
		new = &section{}
	}
	if base == nil {
		base = &section{}
	}

	var keys []string
	seen := make(map[string]bool)
	for _, k := range new.keys {
		keys = append(keys, k)
		seen[k] = true
	}
	for _, k := range disk.keys {
		if !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}

	vals := make(map[string][]string)
	var conflicts []Conflict

	for _, k := range keys {
		inBase, bv := lookup(base, k)
		inDisk, dv := lookup(disk, k)
		inNew, nv := lookup(new, k)

		result, conflict := mergeValue(unitName, name, k, inBase, inDisk, inNew, bv, dv, nv)
		if result != nil {
			vals[k] = result
		}
		if conflict != nil {
			conflicts = append(conflicts, *conflict)
		}
	}

	var out []string
	for _, k := range keys {
		if _, ok := vals[k]; ok {
			out = append(out, k)
		}
	}
	return out, vals, conflicts
}

func mergeValue(unit, sec, key string, inBase, inDisk, inNew bool, bv, dv, nv []string) ([]string, *Conflict) {
	conflict := func() *Conflict {
		return &Conflict{Unit: unit, Section: sec, Key: key, User: joinVals(dv), Generated: joinVals(nv)}
	}

	switch {
	case inDisk && inNew:
		if inBase {
			diskEq, newEq := equalVals(dv, bv), equalVals(nv, bv)
			switch {
			case diskEq && newEq:
				return bv, nil
			case !diskEq && newEq:
				return dv, nil
			case diskEq && !newEq:
				return nv, nil
			default:
				if equalVals(dv, nv) {
					return nv, nil
				}
				return dv, conflict()
			}
		}
		if equalVals(dv, nv) {
			return nv, nil
		}
		return dv, conflict()

	case inDisk && !inNew:
		if inBase {
			if equalVals(dv, bv) {
				return nil, nil
			}
			c := conflict()
			c.Generated = removedSentinel
			return dv, c
		}
		return dv, nil

	case !inDisk && inNew:
		if inBase {
			if equalVals(nv, bv) {
				return nil, nil
			}
			c := conflict()
			c.User = removedSentinel
			return nil, c
		}
		return nv, nil

	default:
		return nil, nil
	}
}

func lookup(s *section, key string) (bool, []string) {
	if s == nil {
		return false, nil
	}
	v, ok := s.vals[key]
	return ok, v
}

func equalVals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func joinVals(v []string) string {
	return strings.Join(v, " ")
}

func directivesFrom(keys []string, vals map[string][]string) []c2q.Directive {
	dirs := make([]c2q.Directive, 0, len(keys))
	for _, k := range keys {
		v := vals[k]
		if len(v) == 0 {
			v = nil
		}
		dirs = append(dirs, c2q.Directive{Key: k, Values: v})
	}
	return dirs
}

func readFile(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(data), true, nil
}

func writeFileAtomic(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".comquad-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

func findStaleNames(dir, prefix string, expected map[string]bool) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, prefix) || expected[name] || !isQuadletFile(name) {
			continue
		}
		names = append(names, name)
	}
	return names
}

func dedupe(names []string) []string {
	seen := make(map[string]bool, len(names))
	out := make([]string, 0, len(names))
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

func isQuadletFile(name string) bool {
	for _, ext := range []string{".container", ".network", ".volume", ".image", ".build"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}
