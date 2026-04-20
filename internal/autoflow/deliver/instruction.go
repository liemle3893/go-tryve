// Package deliver implements the autoflow-deliver workflow orchestrator
// — a 13-step state machine that returns JSON instructions the LLM
// executes. Ports skills/autoflow-deliver/scripts/step-controller.py.
package deliver

import "encoding/json"

// Action is the discriminator for the Instruction struct.
type Action string

const (
	ActionBash             Action = "bash"
	ActionDispatch         Action = "dispatch"
	ActionDispatchParallel Action = "dispatch_parallel"
	ActionAutoComplete     Action = "auto_complete"
	ActionEscalate         Action = "escalate"
	ActionDone             Action = "done"
)

// Instruction is the JSON object returned by `deliver next`. Fields are
// deliberately flattened (no separate types per action) to match the
// Python form one-for-one — parity tests compare the JSON directly.
type Instruction struct {
	Action      Action `json:"action"`
	Step        int    `json:"step,omitempty"`
	Description string `json:"description,omitempty"`

	// bash
	Commands []string `json:"commands,omitempty"`

	// dispatch
	SubagentType string `json:"subagent_type,omitempty"`
	Prompt       string `json:"prompt,omitempty"`
	ParseReturn  string `json:"parse_return,omitempty"`

	// dispatch_parallel
	Dispatches []*Instruction `json:"dispatches,omitempty"`

	// modifiers that can appear on bash or dispatch
	Loop              bool              `json:"loop,omitempty"`
	OnFailure         string            `json:"on_failure,omitempty"`
	OnFixFailedMarker string            `json:"on_fix_failed_marker,omitempty"`
	Note              string            `json:"note,omitempty"`
	Extract           map[string]string `json:"extract,omitempty"`
	PassToComplete    string            `json:"pass_to_complete,omitempty"`
	PostActions       []PostAction      `json:"post_actions,omitempty"`

	// auto_complete / escalate / done
	Reason  string `json:"reason,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// PostAction is the `post_actions` list-item used after step_02 and
// step_13 to trigger a Jira transition after the main action succeeds.
type PostAction struct {
	Action         string `json:"action"` // "jira_transition"
	Ticket         string `json:"ticket"`
	FromStatus     string `json:"from_status"`
	ToStatus       string `json:"to_status"`
	TransitionName string `json:"transition_name"`
}

// autoComplete constructs a pre-filled auto-complete instruction.
func autoComplete(step int, reason string) *Instruction {
	return &Instruction{Action: ActionAutoComplete, Step: step, Reason: reason}
}

// escalate constructs a pre-filled escalate instruction.
func escalate(reason string) *Instruction {
	return &Instruction{Action: ActionEscalate, Reason: reason}
}

// MarshalJSON keeps the output stable: action first, then step, then
// everything else. Go's default struct-order marshal already does this
// given the field order above, so this is a compile-time reminder —
// do not reorder struct fields without updating the parity fixtures.
var _ = json.Marshal
