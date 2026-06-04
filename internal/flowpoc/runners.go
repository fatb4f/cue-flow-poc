package flowpoc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/tools/flow"
)

type taskRunner func(cue.Value, *RunReport) (map[string]any, error)

type Surface struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Pattern string `json:"pattern"`
	Claim   string `json:"claim"`
}

type AmbiguousSurface struct {
	Path   string `json:"path"`
	Claim  string `json:"claim"`
	Reason string `json:"reason"`
}

type ClassificationOutput struct {
	Authoritative   []string           `json:"authoritative"`
	ProjectionOnly  []string           `json:"projectionOnly"`
	Legacy          []string           `json:"legacy"`
	AdapterBoundary []string           `json:"adapterBoundary"`
	Ambiguous       []AmbiguousSurface `json:"ambiguous"`
	Accepted        bool               `json:"accepted"`
	Ambiguity       []string           `json:"ambiguity"`
}

func runTask(t *flow.Task, run *RunReport, f taskRunner) error {
	output, err := f(t.Value(), run)
	if err != nil {
		return err
	}
	if err := validateOutput(output); err != nil {
		return err
	}
	if err := t.Fill(map[string]any{"output": output}); err != nil {
		return err
	}
	if run != nil {
		run.Runner.CalledTaskFill = true
	}
	return nil
}

func discoverRoot(v cue.Value, run *RunReport) (map[string]any, error) {
	files, err := stringList(v.LookupPath(cue.ParsePath("input.files")))
	if err != nil {
		return nil, err
	}

	ambiguity := []string{}
	for _, name := range files {
		if _, err := os.Stat(filepath.Join(run.RepoRoot, name)); err != nil {
			ambiguity = append(ambiguity, fmt.Sprintf("missing required authority file %s: %v", name, err))
		}
	}

	return map[string]any{
		"rootAuthorityFile": "AGENTS.cue",
		"lifecycleSchema":   "cue/flow/schema.cue",
		"rootAuthorityKind": "flow-task-graph-lifecycle-schema",
		"ambiguity":         ambiguity,
		"accepted":          len(ambiguity) == 0,
	}, nil
}

func scanSurfaces(v cue.Value, run *RunReport) (map[string]any, error) {
	patterns, err := stringList(v.LookupPath(cue.ParsePath("input.includePatterns")))
	if err != nil {
		return nil, err
	}
	excludes, err := stringList(v.LookupPath(cue.ParsePath("input.excludePaths")))
	if err != nil {
		return nil, err
	}

	surfaces := []Surface{}
	err = filepath.WalkDir(run.RepoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(run.RepoRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if excluded(rel, excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !scanCandidate(rel) {
			return nil
		}
		fileSurfaces, err := scanFile(path, rel, patterns)
		if err != nil {
			return err
		}
		surfaces = append(surfaces, fileSurfaces...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(surfaces, func(i, j int) bool {
		if surfaces[i].Path != surfaces[j].Path {
			return surfaces[i].Path < surfaces[j].Path
		}
		return surfaces[i].Line < surfaces[j].Line
	})

	return map[string]any{
		"surfaces":  surfaces,
		"accepted":  true,
		"ambiguity": []string{},
	}, nil
}

func classifySurfaces(v cue.Value, run *RunReport) (map[string]any, error) {
	surfaces, err := surfacesFromValue(v.LookupPath(cue.ParsePath("surfaces")))
	if err != nil {
		return nil, err
	}

	classification := ClassificationOutput{}
	for _, surface := range surfaces {
		switch classifyPath(surface.Path) {
		case "authoritative":
			classification.Authoritative = appendUnique(classification.Authoritative, surface.Path)
		case "adapterBoundary":
			classification.AdapterBoundary = appendUnique(classification.AdapterBoundary, surface.Path)
		case "legacy":
			classification.Legacy = appendUnique(classification.Legacy, surface.Path)
			classification.Ambiguous = append(classification.Ambiguous, AmbiguousSurface{
				Path:   surface.Path,
				Claim:  surface.Claim,
				Reason: "legacy authority-shaped surface is outside the selected root CUE node",
			})
		default:
			classification.ProjectionOnly = appendUnique(classification.ProjectionOnly, surface.Path)
			if ambiguityCandidate(surface) {
				classification.Ambiguous = append(classification.Ambiguous, AmbiguousSurface{
					Path:   surface.Path,
					Claim:  surface.Claim,
					Reason: "authority-shaped claim is projection-only unless accepted by AGENTS.cue or cue/flow/schema.cue",
				})
			}
		}
	}
	sort.Strings(classification.Authoritative)
	sort.Strings(classification.ProjectionOnly)
	sort.Strings(classification.Legacy)
	sort.Strings(classification.AdapterBoundary)
	classification.Accepted = len(classification.Ambiguous) == 0
	classification.Ambiguity = ambiguityMessages(classification.Ambiguous)
	run.Classification = classification

	return map[string]any{
		"authoritative":   classification.Authoritative,
		"projectionOnly":  classification.ProjectionOnly,
		"legacy":          classification.Legacy,
		"adapterBoundary": classification.AdapterBoundary,
		"ambiguous":       classification.Ambiguous,
		"accepted":        classification.Accepted,
		"ambiguity":       classification.Ambiguity,
	}, nil
}

func assessSSOT(v cue.Value, run *RunReport) (map[string]any, error) {
	classification, err := classificationFromValue(v.LookupPath(cue.ParsePath("classification")))
	if err != nil {
		return nil, err
	}
	blocking := ambiguityMessages(classification.Ambiguous)
	verdict := "clear"
	if len(blocking) > 0 {
		verdict = "ambiguous"
	}

	return map[string]any{
		"ssotRoot":          "cue/flow/schema.cue",
		"ambiguityCount":    len(blocking),
		"blockingAmbiguity": blocking,
		"verdict":           verdict,
		"accepted":          len(blocking) == 0,
		"ambiguity":         blocking,
	}, nil
}

func emitReport(v cue.Value, run *RunReport) (map[string]any, error) {
	assessment, err := assessmentFromValue(v.LookupPath(cue.ParsePath("assessment")))
	if err != nil {
		return nil, err
	}
	classification, err := classificationFromValue(v.LookupPath(cue.ParsePath("classification")))
	if err != nil {
		return nil, err
	}
	surfaces, err := surfacesFromValue(v.LookupPath(cue.ParsePath("surfaces.surfaces")))
	if err != nil {
		return nil, err
	}

	reportPath := filepath.Join("artifacts", "ssot-ambiguity-report.json")
	absPath := filepath.Join(run.RepoRoot, reportPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, err
	}

	body := map[string]any{
		"summary":          summary(assessment, len(surfaces)),
		"ssotRoot":         assessment["ssotRoot"],
		"verdict":          assessment["verdict"],
		"ambiguityCount":   assessment["ambiguityCount"],
		"classification":   classification,
		"classifiedCount":  len(surfaces),
		"rootAuthority":    valueAsJSON(v.LookupPath(cue.ParsePath("rootAuthority"))),
		"blockingMessages": assessment["blockingAmbiguity"],
	}
	data, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(absPath, append(data, '\n'), 0o644); err != nil {
		return nil, err
	}

	return map[string]any{
		"reportPath": reportPath,
		"summary":    body["summary"],
		"complete":   true,
		"accepted":   assessment["accepted"],
		"ambiguity":  assessment["ambiguity"],
	}, nil
}

func validateOutput(output map[string]any) error {
	if output == nil {
		return fmt.Errorf("task output proposal must not be nil")
	}
	return nil
}

func stringList(v cue.Value) ([]string, error) {
	iter, err := v.List()
	if err != nil {
		return nil, err
	}
	var values []string
	for iter.Next() {
		value, err := iter.Value().String()
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func surfacesFromValue(v cue.Value) ([]Surface, error) {
	iter, err := v.List()
	if err != nil {
		return nil, err
	}
	var surfaces []Surface
	for iter.Next() {
		item := iter.Value()
		path, err := item.LookupPath(cue.ParsePath("path")).String()
		if err != nil {
			return nil, err
		}
		line, err := item.LookupPath(cue.ParsePath("line")).Int64()
		if err != nil {
			return nil, err
		}
		pattern, err := item.LookupPath(cue.ParsePath("pattern")).String()
		if err != nil {
			return nil, err
		}
		claim, err := item.LookupPath(cue.ParsePath("claim")).String()
		if err != nil {
			return nil, err
		}
		surfaces = append(surfaces, Surface{
			Path:    path,
			Line:    int(line),
			Pattern: pattern,
			Claim:   claim,
		})
	}
	return surfaces, nil
}

func classificationFromValue(v cue.Value) (ClassificationOutput, error) {
	var out ClassificationOutput
	var err error
	if out.Authoritative, err = stringList(v.LookupPath(cue.ParsePath("authoritative"))); err != nil {
		return out, err
	}
	if out.ProjectionOnly, err = stringList(v.LookupPath(cue.ParsePath("projectionOnly"))); err != nil {
		return out, err
	}
	if out.Legacy, err = stringList(v.LookupPath(cue.ParsePath("legacy"))); err != nil {
		return out, err
	}
	if out.AdapterBoundary, err = stringList(v.LookupPath(cue.ParsePath("adapterBoundary"))); err != nil {
		return out, err
	}
	out.Ambiguous, err = ambiguousFromValue(v.LookupPath(cue.ParsePath("ambiguous")))
	if err != nil {
		return out, err
	}
	out.Accepted, err = v.LookupPath(cue.ParsePath("accepted")).Bool()
	if err != nil {
		return out, err
	}
	out.Ambiguity, err = stringList(v.LookupPath(cue.ParsePath("ambiguity")))
	return out, err
}

func ambiguousFromValue(v cue.Value) ([]AmbiguousSurface, error) {
	iter, err := v.List()
	if err != nil {
		return nil, err
	}
	var values []AmbiguousSurface
	for iter.Next() {
		item := iter.Value()
		path, err := item.LookupPath(cue.ParsePath("path")).String()
		if err != nil {
			return nil, err
		}
		claim, err := item.LookupPath(cue.ParsePath("claim")).String()
		if err != nil {
			return nil, err
		}
		reason, err := item.LookupPath(cue.ParsePath("reason")).String()
		if err != nil {
			return nil, err
		}
		values = append(values, AmbiguousSurface{Path: path, Claim: claim, Reason: reason})
	}
	return values, nil
}

func assessmentFromValue(v cue.Value) (map[string]any, error) {
	root, err := v.LookupPath(cue.ParsePath("ssotRoot")).String()
	if err != nil {
		return nil, err
	}
	count, err := v.LookupPath(cue.ParsePath("ambiguityCount")).Int64()
	if err != nil {
		return nil, err
	}
	blocking, err := stringList(v.LookupPath(cue.ParsePath("blockingAmbiguity")))
	if err != nil {
		return nil, err
	}
	verdict, err := v.LookupPath(cue.ParsePath("verdict")).String()
	if err != nil {
		return nil, err
	}
	accepted, err := v.LookupPath(cue.ParsePath("accepted")).Bool()
	if err != nil {
		return nil, err
	}
	ambiguity, err := stringList(v.LookupPath(cue.ParsePath("ambiguity")))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ssotRoot":          root,
		"ambiguityCount":    int(count),
		"blockingAmbiguity": blocking,
		"verdict":           verdict,
		"accepted":          accepted,
		"ambiguity":         ambiguity,
	}, nil
}

func scanFile(absPath, relPath string, patterns []string) ([]Surface, error) {
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var surfaces []Surface
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		for _, pattern := range patterns {
			if containsPattern(line, pattern) {
				surfaces = append(surfaces, Surface{
					Path:    relPath,
					Line:    lineNo,
					Pattern: pattern,
					Claim:   line,
				})
				break
			}
		}
	}
	return surfaces, scanner.Err()
}

func containsPattern(line, pattern string) bool {
	if strings.HasPrefix(pattern, "$") || strings.Contains(pattern, ".") {
		return strings.Contains(line, pattern)
	}
	return strings.Contains(strings.ToLower(line), strings.ToLower(pattern))
}

func excluded(path string, excludes []string) bool {
	for _, exclude := range excludes {
		if path == exclude || strings.HasPrefix(path, exclude+"/") || strings.Contains(path, "/"+exclude+"/") {
			return true
		}
	}
	return false
}

func scanCandidate(path string) bool {
	switch filepath.Ext(path) {
	case ".cue", ".go", ".md", ".txt", ".just":
		return true
	default:
		return path == "justfile" || path == "AGENTS.cue" || path == "AGENTS.md"
	}
}

func classifyPath(path string) string {
	switch {
	case path == "AGENTS.cue", path == "cue/flow/schema.cue", path == "cue/flow/authority.cue":
		return "authoritative"
	case strings.HasPrefix(path, "internal/flowpoc/"), strings.HasPrefix(path, "cmd/flow-poc/"):
		return "adapterBoundary"
	case path == "AGENTS.md":
		return "legacy"
	default:
		return "projectionOnly"
	}
}

func ambiguityCandidate(surface Surface) bool {
	pattern := strings.ToLower(surface.Pattern)
	return pattern == "authority" ||
		pattern == "owns" ||
		pattern == "ownedby" ||
		pattern == "policy" ||
		pattern == "must" ||
		pattern == "clear" ||
		pattern == "admissible" ||
		pattern == "lifecycle"
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func ambiguityMessages(values []AmbiguousSurface) []string {
	messages := make([]string, 0, len(values))
	for _, value := range values {
		messages = append(messages, fmt.Sprintf("%s: %s", value.Path, value.Reason))
	}
	return messages
}

func summary(assessment map[string]any, surfaceCount int) string {
	return fmt.Sprintf("SSOT assessment %s with %d unresolved ambiguity findings across %d authority-shaped surfaces",
		assessment["verdict"], assessment["ambiguityCount"], surfaceCount)
}

func valueAsJSON(v cue.Value) any {
	data, err := v.MarshalJSON()
	if err != nil {
		return map[string]string{"error": err.Error()}
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]string{"error": err.Error()}
	}
	return out
}
