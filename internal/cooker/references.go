package cooker

import (
	"sort"
	"strings"
)

// rewriteReferences rewrites quadlet unit references inside a file's content.
func (c *Cooker) rewriteReferences(content string, renameMap map[string]string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !isReferenceDirective(trimmed) {
			continue
		}

		sortedKeys := make([]string, 0, len(renameMap))
		for k := range renameMap {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Slice(sortedKeys, func(i, j int) bool {
			return len(sortedKeys[i]) > len(sortedKeys[j])
		})
		for _, oldName := range sortedKeys {
			newName := renameMap[oldName]
			oldRef := stripQuadletExtension(oldName)
			newRef := stripQuadletExtension(newName)
			if oldRef != newRef {
				lines[i] = c.replaceDirectiveValue(lines[i], oldRef, newRef)
			}
		}
	}
	return strings.Join(lines, "\n")
}

// isReferenceDirective returns true for quadlet directives that reference other units.
func isReferenceDirective(line string) bool {
	directives := []string{
		"Network=",
		"Volume=",
		"Pod=",
		"Wants=",
		"Requires=",
		"Requisite=",
		"BindsTo=",
		"PartOf=",
		"Upholds=",
		"Conflicts=",
		"Before=",
		"After=",
	}
	for _, d := range directives {
		if strings.HasPrefix(line, d) {
			return true
		}
	}
	return false
}

// stripQuadletExtension removes known quadlet extensions from a filename.
func stripQuadletExtension(name string) string {
	for _, ext := range []string{".container", ".service", ".network", ".volume", ".pod", ".kube", ".image", ".build"} {
		if strings.HasSuffix(name, ext) {
			return strings.TrimSuffix(name, ext)
		}
	}
	return name
}

// replaceDirectiveValue replaces oldRef with newRef in a quadlet directive value.
func (c *Cooker) replaceDirectiveValue(line, oldRef, newRef string) string {
	colonIdx := strings.Index(line, "=")
	if colonIdx < 0 {
		return line
	}
	directive := line[:colonIdx]
	value := line[colonIdx+1:]

	unitDirectives := map[string]bool{
		"Wants":     true,
		"Requires":  true,
		"Requisite": true,
		"BindsTo":   true,
		"PartOf":    true,
		"Upholds":   true,
		"Conflicts": true,
		"Before":    true,
		"After":     true,
	}

	if unitDirectives[directive] {
		return c.replaceUnitDirectives(line, directive, value, oldRef, newRef)
	}

	switch directive {
	case "Volume":
		parts := strings.SplitN(value, ":", 2)
		if len(parts) >= 2 && strings.Contains(parts[0], "/") {
			return line
		}
		if strings.Contains(parts[0], oldRef) {
			if strings.Contains(parts[0], newRef) {
				return line
			}
			replaced := strings.Replace(parts[0], oldRef, newRef, 1)
			if len(parts) == 2 {
				return directive + "=" + replaced + ":" + parts[1]
			}
			return directive + "=" + replaced
		}
		return line
	default:
		if c.valueContainsRef(value, oldRef, newRef) {
			return directive + "=" + strings.Replace(value, oldRef, newRef, 1)
		}
		return line
	}
}

// valueContainsRef checks whether value contains oldRef as a standalone reference
// (not as a substring within a longer name). This avoids corrupting already-prefixed
// values like replacing "nginx" inside "cq-nginx-nodejs-redis-default".
func (c *Cooker) valueContainsRef(value, oldRef, newRef string) bool {
	if strings.Contains(value, newRef) {
		return false
	}

	for _, suffix := range []string{".network", ".volume", ".pod", ".image", ".container", ".build", ".kube", ".service"} {
		extValue := oldRef + suffix
		if strings.HasSuffix(value, extValue) {
			return true
		}
		if value == oldRef {
			return true
		}
	}

	parts := strings.Split(value, ":")
	for _, part := range parts {
		if part == oldRef {
			return true
		}
		for _, suffix := range []string{".network", ".volume", ".pod", ".image", ".container", ".build", ".kube", ".service"} {
			if part == oldRef+suffix {
				return true
			}
		}
	}

	return false
}

// replaceUnitDirectives handles [Unit] section directives with multiple space-separated unit references.
func (c *Cooker) replaceUnitDirectives(line, directive, value, oldRef, newRef string) string {
	if oldRef == newRef {
		return line
	}

	tokens := strings.Fields(value)
	if len(tokens) == 0 {
		return line
	}

	changed := false
	for i, token := range tokens {
		tokenName := stripQuadletExtension(token)
		if tokenName == oldRef {
			if strings.Contains(token, newRef) {
				continue
			}
			ext := ""
			if strings.HasSuffix(token, ".service") {
				ext = ".service"
			} else if strings.HasSuffix(token, ".container") {
				ext = ".container"
			}
			tokens[i] = newRef + ext
			changed = true
		}
	}

	if !changed {
		return line
	}

	return directive + "=" + strings.Join(tokens, " ")
}

// splitCombinedLabels splits combined Label= lines into separate Label= lines.
func (c *Cooker) splitCombinedLabels(lines []string) []string {
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Label=") {
			value := strings.TrimPrefix(trimmed, "Label=")
			pairs := labelFields(value)
			if len(pairs) == 0 {
				result = append(result, line)
			} else {
				for _, pair := range pairs {
					result = append(result, "Label="+pair)
				}
			}
		} else {
			result = append(result, line)
		}
	}
	return result
}

// labelFields splits a space-separated key=value string into individual tokens.
func labelFields(s string) []string {
	var fields []string
	s = strings.TrimSpace(s)

	for len(s) > 0 {
		s = strings.TrimLeft(s, " ")
		if len(s) == 0 {
			break
		}

		eqIdx := strings.IndexByte(s, '=')
		if eqIdx < 0 {
			fields = append(fields, s)
			break
		}

		afterEq := eqIdx + 1

		if afterEq < len(s) && (s[afterEq] == '"' || s[afterEq] == '\'') {
			quote := s[afterEq]
			valStart := afterEq + 1
			closeIdx := -1
			for i := valStart; i < len(s); i++ {
				if s[i] == '\\' && i+1 < len(s) {
					i++
					continue
				}
				if s[i] == quote {
					closeIdx = i + 1
					break
				}
			}
			if closeIdx < 0 {
				fields = append(fields, s)
				break
			}
			fields = append(fields, s[:closeIdx])
			s = s[closeIdx:]
		} else {
			spaceIdx := strings.IndexByte(s[afterEq:], ' ')
			if spaceIdx < 0 {
				fields = append(fields, s)
				break
			}
			fields = append(fields, s[:afterEq+spaceIdx])
			s = s[afterEq+spaceIdx:]
		}
	}

	return fields
}
