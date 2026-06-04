# cue-flow-poc

Minimal proof-of-concept for a CUE-owned `tools/flow` authority model.

## Purpose

This repo is intentionally small. It proves the control law before any migration back into a larger dotfiles or agentflow repository:

```text
CUE owns the task and step contracts.
cuelang.org/go/tools/flow owns lifecycle mechanics.
TaskFunc classifies CUE values into runnable tasks.
Runner executes only a bound task and may call Task.Fill.
Task.Fill output is accepted only through the CUE step contract.
go-mcp is an adapter boundary, not a policy authority.
```

## Non-goals

- No JSON-LD.
- No MCP server implementation yet.
- No dotfiles import.
- No Hookrail import.
- No historical authority documents.
- No broad task registry.

## Layout

```text
.
├── AGENTS.cue                         # root authority pointer
├── cue.mod/module.cue                 # CUE module
├── cue/flow/schema.cue                # root PoC contract model
├── cue/flow/authority.cue             # authority summary value
├── cue/flow/fixtures/*.cue            # good/bad contract proofs
├── cue/flow/app/app.cue               # tiny runnable CUE app for Go proof
├── internal/flowpoc/*.go              # local tools/flow adapter proof
└── cmd/flow-poc/main.go               # emits flow proof report
```

## Phase 0: CUE-only proof

```sh
just fmt
just vet-good
just vet-bad
```

The bad fixtures are expected to fail. The `vet-bad` recipe uses shell negation.

## Phase 1: local Go flow proof

```sh
go mod tidy
go run ./cmd/flow-poc
```

The Go proof loads `cue/flow/app`, runs `tools/flow`, records task states/dependencies, and prints the final CUE value.

## Control law

```text
flow.Terminated is a native lifecycle state.
contract.clear is an authority overlay.

A step can clear only when:
  flowTerminated == true
  outputAccepted == true
  authorityAccepted == true
  ambiguity == []
```

