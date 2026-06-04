package fixtures

import flow "cue-flow-poc/cue/flow"

_tf: {
	id:      "taskfunc.basic"
	adapter: "local-go"

	classifiesCueValue: true
	createsRunner:     true
	ownsPolicy:        false
}

_echoRunner: {
	id:      "runner.echo"
	adapter: "local-go"

	executesTask: true
	mayFill:      true
	ownsPolicy:   false
}


badExplicitEdgeAuthority: flow.#FlowRunContract & {
	config: {
		root: "flow"
	}

	taskFunc: _tf

	tasks: {
		first: {
			id: "first"
			kind: "run_step"
			path: "flow.first"
			input: {message: "hello"}
			output: {message: "hello"}
			taskFunc: _tf
			runner: _echoRunner
			state: "Terminated"
			referenceDependencies: []
		}
		second: {
			id: "second"
			kind: "run_step"
			path: "flow.second"
			input: {message: "hello"}
			output: {message: "hello"}
			taskFunc: _tf
			runner: _echoRunner
			state: "Terminated"
			referenceDependencies: []
			dependsOnProjection: ["first"]
			dependsOnProjectionAuthority: "authoritative"
		}
	}

	referenceGraph: {
		cyclic: false
		edges: []
	}

	steps: [{
		id: "step.first"
		task: tasks.first
		ambiguity: []
		flowTerminated: true
		outputAccepted: true
		authorityAccepted: true
		clear: true
	}, {
		id: "step.second"
		task: tasks.second
		ambiguity: []
		flowTerminated: true
		outputAccepted: true
		authorityAccepted: true
		clear: true
	}]
}
