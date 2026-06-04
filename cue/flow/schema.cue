package flow

// #FlowState mirrors cuelang.org/go/tools/flow v0.6.0 task states.
// Native flow does not have a "clear" state; clear is an authority overlay.
#FlowState: "Waiting" | "Ready" | "Running" | "Terminated"

#AdapterKind: "local-go" | "go-mcp"

#TaskKind:
	"resolve" |
	"bind_contract" |
	"run_step" |
	"validate_fill" |
	"record_evidence"

#FlowConfig: {
	root!: string

	// PoC authority mode intentionally disables ambiguity-expanding discovery.
	inferTasks:      false | *false
	ignoreConcrete: bool | *false
	findHiddenTasks: false | *false
}

#TaskFuncBinding: {
	id!: string

	adapter!: #AdapterKind

	source: {
		module:  "cuelang.org/go"
		package: "tools/flow"
		version: "v0.6.0"
	}

	classifiesCueValue: true
	createsRunner:     true

	// TaskFunc may classify values and create Runners, but CUE owns admissibility.
	ownsPolicy: false
}

#RunnerBinding: {
	id!: string

	adapter!: #AdapterKind

	executesTask: true
	mayFill:      true

	// Runner executes and may Fill. It never owns policy.
	ownsPolicy: false
}

#ReferenceDependency: {
	fromTaskPath!:  string
	fromValuePath!: string
	toTaskPath!:    string

	// In tools/flow, dependencies are derived from CUE references.
	derivedBy: "cue-reference-analysis"
}

#AmbiguityKind:
	"infer_tasks_enabled" |
	"hidden_tasks_enabled" |
	"taskfunc_schema_unbound" |
	"runner_unbound" |
	"explicit_edge_claims_authority" |
	"unaccepted_fill_payload" |
	"runner_claims_policy_authority" |
	"uncleared_dependency" |
	"cycle_detected" |
	"step_contract_unbound"

#AmbiguityFinding: {
	kind!:     #AmbiguityKind
	path!:     string
	reason!:   string
	severity: "blocker"
}

#TaskContract: {
	id!:   string
	kind!: #TaskKind

	path!: string

	input!:  _
	output!: _

	taskFunc!: #TaskFuncBinding
	runner!:   #RunnerBinding

	state!: #FlowState

	referenceDependencies: [...#ReferenceDependency]

	// Optional projection/cache only. It is never graph authority.
	dependsOnProjection?: [...string]

	// If present, this field must say projection-only.
	dependsOnProjectionAuthority?: "projection-only"

	authority: {
		taskShapeOwnedBy: "cue-contract"
		edgeAuthority:   "cue-references"
		executionOwnedBy: "runner"

		adapterOwnsPolicy: false
	}
}

#FillContract: {
	taskPath!: string

	source:         "Task.Fill"
	composition:    "conjunctive"
	effectiveAfter: "Task.Terminated"

	payload!: _
	accepted!: bool
}

#AcceptedFillContract: #FillContract & {
	accepted: true
}

#StepContract: {
	id!: string

	task!: #TaskContract

	fill?: #AcceptedFillContract

	ambiguity!: [...#AmbiguityFinding]

	flowTerminated!:    bool
	outputAccepted!:    bool
	authorityAccepted!: bool

	clear!: bool

	clear: flowTerminated == true
	clear: outputAccepted == true
	clear: authorityAccepted == true
	clear: len(ambiguity) == 0
}

#FlowRunContract: {
	schemaVersion: "cue-flow-poc.v0"

	config!: #FlowConfig

	rootAuthority: {
		kind: "flow-task-graph-lifecycle-schema"
		path: "cue/flow/schema.cue"
	}

	taskFunc!: #TaskFuncBinding

	tasks!: [string]: #TaskContract

	referenceGraph!: {
		edgeAuthority: "cue-references"
		cyclic:        false
		edges:         [...#ReferenceDependency]
	}

	steps!: [...#StepContract]

	invariants: {
		cueOwnsTaskContracts:       true
		cueReferencesOwnEdges:      true
		taskFuncClassifiesOnly:     true
		runnerExecutesOnly:         true
		adapterOwnsPolicy:          false
		fillRequiresOutputContract: true
		clearRequiresZeroAmbiguity: true
		goMCPIsAdapterOnly:         true
	}
}
