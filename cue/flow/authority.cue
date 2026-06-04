package flow

authoritySummary: {
	root: {
		kind: "flow-task-graph-lifecycle-schema"
		path: "cue/flow/schema.cue"
	}

	source: {
		module:  "cuelang.org/go"
		package: "tools/flow"
		version: "v0.6.0"
	}

	boundaries: {
		cueOwnsContracts:           true
		cueReferencesOwnEdges:      true
		flowOwnsLifecycleMechanics: true
		taskFuncClassifiesOnly:     true
		runnerExecutesOnly:         true
		goMCPIsAdapterOnly:         true
	}
}
