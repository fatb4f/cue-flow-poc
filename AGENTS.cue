package root

rootAuthority: {
	id:   "cue-flow-poc.root"
	kind: "flow-task-graph-lifecycle-schema"
	path: "cue/flow/schema.cue"

	invariants: {
		cueOwnsContracts:               true
		flowOwnsLifecycleMechanics:     true
		goMCPIsAdapterOnly:             true
		runnersDoNotOwnPolicy:          true
		clearanceRequiresZeroAmbiguity: true
	}
}
