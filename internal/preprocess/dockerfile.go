package preprocess

import (
	"bytes"
	"strings"
)

func PatchDockerfile(content []byte) []byte {
	lines := bytes.Split(content, []byte("\n"))
	var result [][]byte
	aliases := make(map[string]bool)

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		upperLine := bytes.ToUpper(trimmed)

		if !bytes.HasPrefix(upperLine, []byte("FROM ")) && !bytes.HasPrefix(upperLine, []byte("FROM\t")) {
			result = append(result, line)
			continue
		}

		indent := line[:len(line)-len(bytes.TrimLeft(line, " \t"))]
		image, platform, stage := parseFromLine(trimmed)

		if stageName := extractStageName(stage); stageName != "" {
			aliases[strings.ToLower(stageName)] = true
		}

		if image != "scratch" && !aliases[strings.ToLower(image)] {
			image = normalizeImage(image)
		}

		patched := buildFromLine(image, platform, stage)
		result = append(result, append(indent, patched...))
	}

	return bytes.Join(result, []byte("\n"))
}

func parseFromLine(line []byte) (image string, platform []byte, stage []byte) {
	rest := line
	if len(rest) > 5 {
		rest = rest[5:]
	}
	rest = bytes.TrimSpace(rest)

	if bytes.HasPrefix(rest, []byte("--platform")) {
		idx := bytes.IndexByte(rest, ' ')
		if idx >= 0 {
			platform = rest[:idx]
			rest = bytes.TrimSpace(rest[idx+1:])
		}
	}

	upperRest := bytes.ToUpper(rest)
	asIdx := -1
	for i := 0; i < len(upperRest)-3; i++ {
		if upperRest[i] == ' ' && upperRest[i+1] == 'A' && upperRest[i+2] == 'S' && upperRest[i+3] == ' ' {
			asIdx = i
			break
		}
	}
	if asIdx >= 0 {
		stage = bytes.TrimSpace(rest[asIdx:])
		rest = bytes.TrimSpace(rest[:asIdx])
	}

	image = string(rest)
	return
}

func extractStageName(stage []byte) string {
	s := string(stage)
	upper := strings.ToUpper(s)
	if strings.HasPrefix(upper, "AS ") {
		return strings.TrimSpace(s[3:])
	}
	return ""
}

func buildFromLine(image string, platform []byte, stage []byte) []byte {
	var parts []string
	parts = append(parts, "FROM")
	if len(platform) > 0 {
		parts = append(parts, string(platform))
	}
	parts = append(parts, image)
	if len(stage) > 0 {
		parts = append(parts, string(stage))
	}
	return []byte(strings.Join(parts, " "))
}
