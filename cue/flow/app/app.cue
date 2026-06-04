package app

root: {
	discover_root: {
		$id: "discover_root"
		input: files: [
			"AGENTS.cue",
			"cue/flow/schema.cue",
			"cue/flow/authority.cue",
		]
		output: {
			rootAuthorityFile: string
			lifecycleSchema:   string
			rootAuthorityKind: string
			ambiguity: [...string]
			accepted: bool
		}
	}

	scan_surfaces: {
		$id: "scan_surfaces"

		rootAuthority: discover_root.output

		input: {
			includePatterns: [
				"authority",
				"owns",
				"ownedBy",
				"policy",
				"contract",
				"root",
				"must",
				"clear",
				"admissible",
				"lifecycle",
				"runner",
				"agent",
				"Task.Fill",
				"$id",
				"$after",
			]
			excludePaths: [
				".git",
				"vendor",
				"node_modules",
				"go.sum",
				"artifacts",
			]
		}

		output: {
			surfaces: [...{
				path:    string
				line:    int
				pattern: string
				claim:   string
			}]
			accepted: bool
			ambiguity: [...string]
		}
	}

	classify_surfaces: {
		$id: "classify_surfaces"

		rootAuthority: discover_root.output
		surfaces:      scan_surfaces.output.surfaces

		output: {
			authoritative: [...string]
			projectionOnly: [...string]
			legacy: [...string]
			adapterBoundary: [...string]
			ambiguous: [...{
				path:   string
				claim:  string
				reason: string
			}]
			accepted: bool
			ambiguity: [...string]
		}
	}

	assess_ssot: {
		$id: "assess_ssot"

		rootAuthority:  discover_root.output
		classification: classify_surfaces.output

		output: {
			ssotRoot:       string
			ambiguityCount: int
			blockingAmbiguity: [...string]
			verdict:  string
			accepted: bool
			ambiguity: [...string]
		}
	}

	emit_report: {
		$id: "emit_report"

		rootAuthority:  discover_root.output
		surfaces:       scan_surfaces.output
		classification: classify_surfaces.output
		assessment:     assess_ssot.output

		output: {
			reportPath: string
			summary:    string
			complete:   bool
			accepted:   bool
			ambiguity: [...string]
		}
	}
}
