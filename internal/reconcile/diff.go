package reconcile

import (
	"fmt"
	"strings"
)

// Diff renders a unified diff of the plan for user display. It covers created,
// changed, and removed files; unchanged files produce no output.
func (p Plan) Diff() string {
	var b strings.Builder
	for _, fp := range p.Files {
		b.WriteString(fp.Diff())
	}
	return b.String()
}

// Diff renders the unified diff for a single file (empty if unchanged).
func (fp FilePlan) Diff() string {
	switch fp.Status {
	case StatusCreated:
		return unifiedDiff(fp.Name, "", fp.NewContent)
	case StatusChanged:
		return unifiedDiff(fp.Name, fp.OldContent, fp.NewContent)
	case StatusRemoved:
		return unifiedDiff(fp.Name, fp.OldContent, "")
	}
	return ""
}

type diffOp struct {
	kind byte
	line string
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func diffLines(oldLines, newLines []string) []diffOp {
	n, m := len(oldLines), len(newLines)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case oldLines[i] == newLines[j]:
				dp[i][j] = dp[i+1][j+1] + 1
			case dp[i+1][j] >= dp[i][j+1]:
				dp[i][j] = dp[i+1][j]
			default:
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case oldLines[i] == newLines[j]:
			ops = append(ops, diffOp{' ', oldLines[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, diffOp{'-', oldLines[i]})
			i++
		default:
			ops = append(ops, diffOp{'+', newLines[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{'-', oldLines[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{'+', newLines[j]})
	}
	return ops
}

func hasChange(ops []diffOp) bool {
	for _, op := range ops {
		if op.kind != ' ' {
			return true
		}
	}
	return false
}

func unifiedDiff(name, old, new string) string {
	ops := diffLines(splitLines(old), splitLines(new))
	if !hasChange(ops) {
		return ""
	}

	var b strings.Builder
	if old == "" {
		b.WriteString("--- /dev/null\n")
	} else {
		fmt.Fprintf(&b, "--- a/%s\n", name)
	}
	if new == "" {
		b.WriteString("+++ /dev/null\n")
	} else {
		fmt.Fprintf(&b, "+++ b/%s\n", name)
	}

	const ctx = 3

	var changes []int
	for i, op := range ops {
		if op.kind != ' ' {
			changes = append(changes, i)
		}
	}

	type span struct{ lo, hi int }
	var spans []span
	for _, c := range changes {
		if len(spans) > 0 && c <= spans[len(spans)-1].hi+2*ctx {
			spans[len(spans)-1].hi = c
		} else {
			spans = append(spans, span{c, c})
		}
	}

	for _, s := range spans {
		lo := s.lo - ctx
		if lo < 0 {
			lo = 0
		}
		hi := s.hi + ctx
		if hi >= len(ops) {
			hi = len(ops) - 1
		}

		oldStart, newStart := 1, 1
		for k := 0; k < lo; k++ {
			switch ops[k].kind {
			case ' ':
				oldStart++
				newStart++
			case '-':
				oldStart++
			case '+':
				newStart++
			}
		}

		oldCount, newCount := 0, 0
		for k := lo; k <= hi; k++ {
			switch ops[k].kind {
			case ' ':
				oldCount++
				newCount++
			case '-':
				oldCount++
			case '+':
				newCount++
			}
		}

		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount)
		for k := lo; k <= hi; k++ {
			b.WriteByte(ops[k].kind)
			b.WriteString(ops[k].line)
			b.WriteByte('\n')
		}
	}

	return b.String()
}
