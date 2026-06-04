set shell := ["bash", "-cu"]

CUE := "cue"
GO := "go"

fmt:
    {{CUE}} fmt ./...

vet-good:
    {{CUE}} eval -c ./cue/flow/fixtures/good_single_step.cue >/dev/null
    {{CUE}} eval -c ./cue/flow/fixtures/good_two_step_reference_chain.cue >/dev/null

vet-bad:
    ! {{CUE}} eval -c ./cue/flow/fixtures/bad_ambiguity.cue >/dev/null
    ! {{CUE}} eval -c ./cue/flow/fixtures/bad_runner_policy.cue >/dev/null
    ! {{CUE}} eval -c ./cue/flow/fixtures/bad_unaccepted_fill.cue >/dev/null
    ! {{CUE}} eval -c ./cue/flow/fixtures/bad_explicit_edge_authority.cue >/dev/null
    ! {{CUE}} eval -c ./cue/flow/fixtures/bad_infer_tasks_enabled.cue >/dev/null
    ! {{CUE}} eval -c ./cue/flow/fixtures/bad_unbound_runner.cue >/dev/null

go-proof:
    {{GO}} run ./cmd/flow-poc

check-cue: fmt vet-good vet-bad

check: check-cue go-proof
