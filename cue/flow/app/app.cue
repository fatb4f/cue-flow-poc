package app

flow: {
	first: {
		kind: "echo"
		input: message: "hello"
		output: {
			message: string
			ok:      bool
		}
	}

	second: {
		kind: "echo"
		// This reference is what tools/flow should discover as a dependency.
		input: message: flow.first.output.message
		output: {
			message: string
			ok:      bool
		}
	}
}

report: {
	first:  flow.first.output
	second: flow.second.output
}
