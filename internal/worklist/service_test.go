package worklist

import (
	"errors"
	"testing"
)

func TestTransitionEnforcesApprovedOwnershipAndStates(t *testing.T) {
	actor := int64(7)
	other := int64(8)

	tests := []struct {
		name       string
		current    Status
		assigneeID *int64
		command    Command
		wantState  Status
		wantOwner  *int64
		wantErr    error
	}{
		{name: "arrive waiting", current: StatusWaiting, command: CommandArrive, wantState: StatusArrived},
		{name: "claim arrived", current: StatusArrived, command: CommandClaim, wantState: StatusInProgress, wantOwner: &actor},
		{name: "release owned", current: StatusInProgress, assigneeID: &actor, command: CommandRelease, wantState: StatusArrived},
		{name: "complete owned", current: StatusInProgress, assigneeID: &actor, command: CommandComplete, wantState: StatusCompleted, wantOwner: &actor},
		{name: "release other owner is hidden", current: StatusInProgress, assigneeID: &other, command: CommandRelease, wantErr: ErrNotFound},
		{name: "cannot claim waiting", current: StatusWaiting, command: CommandClaim, wantErr: ErrInvalidTransition},
		{name: "completed cannot transition", current: StatusCompleted, assigneeID: &actor, command: CommandComplete, wantErr: ErrInvalidTransition},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, owner, err := transition(test.current, test.assigneeID, actor, test.command)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("transition() error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr != nil {
				return
			}
			if state != test.wantState {
				t.Fatalf("state = %q, want %q", state, test.wantState)
			}
			if (owner == nil) != (test.wantOwner == nil) || owner != nil && *owner != *test.wantOwner {
				t.Fatalf("owner = %v, want %v", owner, test.wantOwner)
			}
		})
	}
}

func TestValidTicketIDRequiresUUIDv4Shape(t *testing.T) {
	for _, value := range []string{"77777777-7777-4777-8777-777777777777", "11111111-1111-4111-8111-111111111111"} {
		if !validTicketID(value) {
			t.Fatalf("validTicketID(%q) = false", value)
		}
	}
	for _, value := range []string{"", "not-a-ticket", "77777777-7777-3777-8777-777777777777", "77777777-7777-4777-7777-777777777777"} {
		if validTicketID(value) {
			t.Fatalf("validTicketID(%q) = true", value)
		}
	}
}
