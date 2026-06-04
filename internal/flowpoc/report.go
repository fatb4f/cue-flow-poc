package flowpoc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/tools/flow"
)

type RunReport struct {
	SourcePackage  string               `json:"sourcePackage"`
	SourceVersion  string               `json:"sourceVersion"`
	RepoRoot       string               `json:"repoRoot"`
	Root           string               `json:"root"`
	LoadedFiles    []string             `json:"loadedFiles"`
	DeniedLoads    []string             `json:"deniedLoads"`
	Agent          AgentReport          `json:"agent"`
	Runner         RunnerReport         `json:"runner"`
	Flow           FlowReport           `json:"flow"`
	Tasks          []TaskReport         `json:"tasks"`
	Updates        []UpdateEvidence     `json:"updates"`
	FinalStats     string               `json:"finalStats"`
	FinalValue     json.RawMessage      `json:"finalValue"`
	Classification ClassificationOutput `json:"classification"`
}

type AgentReport struct {
	Role         string `json:"role"`
	ProposedFill bool   `json:"proposedFill"`
	OwnsPolicy   bool   `json:"ownsPolicy"`
}

type RunnerReport struct {
	Role           string `json:"role"`
	ValidatedFill  bool   `json:"validatedFill"`
	CalledTaskFill bool   `json:"calledTaskFill"`
	OwnsPolicy     bool   `json:"ownsPolicy"`
}

type FlowReport struct {
	Terminated                    bool   `json:"terminated"`
	TerminalEmitReportOutput      bool   `json:"terminalEmitReportOutput"`
	TerminalReportPath            string `json:"terminalReportPath,omitempty"`
	TerminalUnresolvedAmbiguities int    `json:"terminalUnresolvedAmbiguities"`
}

type TaskReport struct {
	Index        int      `json:"index"`
	Path         string   `json:"path"`
	State        string   `json:"state"`
	Dependencies []string `json:"dependencies"`
	Err          string   `json:"err,omitempty"`
	Stats        string   `json:"stats"`
}

type UpdateEvidence struct {
	Index     int                 `json:"index"`
	TaskPath  string              `json:"taskPath,omitempty"`
	Mermaid   string              `json:"mermaid"`
	TaskValue string              `json:"taskValue,omitempty"`
	TaskStats string              `json:"taskStats,omitempty"`
	Tasks     []TaskUpdateSummary `json:"tasks"`
}

type TaskUpdateSummary struct {
	Path  string `json:"path"`
	State string `json:"state"`
}

func Run(repoRoot string) (*RunReport, error) {
	if repoRoot == "" {
		return nil, fmt.Errorf("repo root required")
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}

	appDir := filepath.Join(absRoot, "cue", "flow", "app")
	insts := load.Instances([]string{"."}, &load.Config{Dir: appDir})
	if len(insts) != 1 {
		return nil, fmt.Errorf("expected one CUE instance, got %d", len(insts))
	}
	inst := insts[0]
	if err := inst.Err; err != nil {
		return nil, fmt.Errorf("load CUE app: %w", err)
	}

	ctx := cuecontext.New()
	root := ctx.BuildInstance(inst)
	if err := root.Err(); err != nil {
		return nil, fmt.Errorf("build CUE app: %w", err)
	}

	report := &RunReport{
		SourcePackage: "cuelang.org/go/tools/flow",
		SourceVersion: "v0.6.0",
		RepoRoot:      absRoot,
		Root:          "root",
		LoadedFiles: []string{
			"AGENTS.cue",
			"cue/flow/schema.cue",
			"cue/flow/authority.cue",
			"cue/flow/app/app.cue",
		},
		DeniedLoads: []string{
			"git-mcp-server status: access denied for /home/_404/src/cue-flow-poc",
			"git-mcp-go status: access denied for /home/_404/src/cue-flow-poc",
		},
		Agent: AgentReport{
			Role:       "semantic-proposal-data",
			OwnsPolicy: false,
		},
		Runner: RunnerReport{
			Role:       "go-flow-runnerfunc",
			OwnsPolicy: false,
		},
	}

	controller := flow.New(&flow.Config{
		Root:            cue.ParsePath("root"),
		InferTasks:      false,
		IgnoreConcrete:  false,
		FindHiddenTasks: false,
		UpdateFunc: func(c *flow.Controller, task *flow.Task) error {
			return recordUpdate(report, c, task)
		},
	}, root, TaskFuncFor(report))

	if err := controller.Run(context.Background()); err != nil {
		return nil, fmt.Errorf("run flow: %w", err)
	}

	final := controller.Value()
	finalJSON, err := final.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal final value: %w", err)
	}
	report.FinalValue = finalJSON
	report.FinalStats = fmt.Sprint(controller.Stats())
	report.Tasks = taskReports(controller.Tasks())
	report.Flow.Terminated = allTasksTerminated(report.Tasks)
	report.Flow.TerminalEmitReportOutput = terminalEmitReportOutput(final)
	report.Flow.TerminalReportPath = terminalReportPath(final)
	report.Flow.TerminalUnresolvedAmbiguities = terminalAmbiguityCount(final)

	if err := writeEvidence(report); err != nil {
		return nil, err
	}

	return report, nil
}

func recordUpdate(report *RunReport, c *flow.Controller, task *flow.Task) error {
	update := UpdateEvidence{
		Index:   len(report.Updates),
		Mermaid: mermaidGraph(c),
	}
	for _, t := range c.Tasks() {
		update.Tasks = append(update.Tasks, TaskUpdateSummary{
			Path:  t.Path().String(),
			State: t.State().String(),
		})
	}
	if task != nil {
		update.TaskPath = task.Path().String()
		update.TaskStats = fmt.Sprint(task.Stats())
		node := task.Value().Syntax(cue.Final())
		formatted, err := format.Node(node)
		if err != nil {
			return err
		}
		update.TaskValue = string(formatted)
		report.Agent.ProposedFill = true
		report.Runner.ValidatedFill = true
	}
	report.Updates = append(report.Updates, update)
	return nil
}

func taskReports(tasks []*flow.Task) []TaskReport {
	reports := make([]TaskReport, 0, len(tasks))
	for _, task := range tasks {
		tr := TaskReport{
			Index: task.Index(),
			Path:  task.Path().String(),
			State: task.State().String(),
			Stats: fmt.Sprint(task.Stats()),
		}
		for _, dep := range task.Dependencies() {
			tr.Dependencies = append(tr.Dependencies, dep.Path().String())
		}
		if err := task.Err(); err != nil {
			tr.Err = err.Error()
		}
		reports = append(reports, tr)
	}
	return reports
}

func writeEvidence(report *RunReport) error {
	path := filepath.Join(report.RepoRoot, "artifacts", "run-evidence.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func allTasksTerminated(tasks []TaskReport) bool {
	if len(tasks) == 0 {
		return false
	}
	for _, task := range tasks {
		if task.State != "Terminated" || task.Err != "" {
			return false
		}
	}
	return true
}

func terminalEmitReportOutput(v cue.Value) bool {
	complete, err := v.LookupPath(cue.ParsePath("root.emit_report.output.complete")).Bool()
	return err == nil && complete
}

func terminalReportPath(v cue.Value) string {
	path, err := v.LookupPath(cue.ParsePath("root.emit_report.output.reportPath")).String()
	if err != nil {
		return ""
	}
	return path
}

func terminalAmbiguityCount(v cue.Value) int {
	count, err := v.LookupPath(cue.ParsePath("root.assess_ssot.output.ambiguityCount")).Int64()
	if err != nil {
		return -1
	}
	return int(count)
}

func mermaidGraph(c *flow.Controller) string {
	var b strings.Builder
	fmt.Fprintln(&b, "graph TD")
	for i, task := range c.Tasks() {
		fmt.Fprintf(&b, "  t%d(\"%s [%s]\")\n", i, escapeMermaid(task.Path().String()), task.State())
		for _, dep := range task.Dependencies() {
			fmt.Fprintf(&b, "  t%d-->t%d\n", i, dep.Index())
		}
	}
	return b.String()
}

func escapeMermaid(s string) string {
	s = strings.ReplaceAll(s, "#", "#35;")
	return strings.ReplaceAll(s, `"`, "#quot;")
}
