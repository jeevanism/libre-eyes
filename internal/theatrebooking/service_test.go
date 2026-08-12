package theatrebooking

import (
	"errors"
	"testing"
)

func TestValidateTransitionAcceptsOnlyApprovedLifecycle(t *testing.T) {
	sessionID := "b1111111-1111-4111-8111-111111111111"
	tests := []struct {
		name    string
		current BookingRequest
		request CommandRequest
		want    error
	}{
		{"schedule waiting", BookingRequest{Status: StatusWaiting}, CommandRequest{Command: CommandSchedule, TargetSessionID: sessionID}, nil},
		{"reschedule scheduled", BookingRequest{Status: StatusScheduled, AssignedSessionID: &sessionID}, CommandRequest{Command: CommandReschedule, TargetSessionID: "c1111111-1111-4111-8111-111111111111"}, nil},
		{"cancel waiting", BookingRequest{Status: StatusWaiting}, CommandRequest{Command: CommandCancel}, nil},
		{"cancel scheduled", BookingRequest{Status: StatusScheduled, AssignedSessionID: &sessionID}, CommandRequest{Command: CommandCancel}, nil},
		{"cannot schedule cancelled", BookingRequest{Status: StatusCancelled}, CommandRequest{Command: CommandSchedule, TargetSessionID: sessionID}, ErrConflict},
		{"cannot reschedule same session", BookingRequest{Status: StatusScheduled, AssignedSessionID: &sessionID}, CommandRequest{Command: CommandReschedule, TargetSessionID: sessionID}, ErrConflict},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateTransition(test.current, test.request)
			if !errors.Is(err, test.want) {
				t.Fatalf("validateTransition() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestInvolvedSessionIDsAreSortedAndDistinct(t *testing.T) {
	prior := "c1111111-1111-4111-8111-111111111111"
	ids := involvedSessionIDs(&prior, "b1111111-1111-4111-8111-111111111111", CommandReschedule)
	if len(ids) != 2 || ids[0] != "b1111111-1111-4111-8111-111111111111" || ids[1] != prior {
		t.Fatalf("ids = %#v", ids)
	}
	if got := involvedSessionIDs(&prior, "", CommandCancel); len(got) != 1 || got[0] != prior {
		t.Fatalf("cancel ids = %#v", got)
	}
}

func TestValidUUIDv4RejectsUnsafeShapes(t *testing.T) {
	if !validUUIDv4("b1111111-1111-4111-8111-111111111111") {
		t.Fatal("validUUIDv4() rejected a valid UUIDv4")
	}
	for _, value := range []string{"", "not-an-id", "b1111111-1111-3111-8111-111111111111", "b1111111-1111-4111-7111-111111111111"} {
		if validUUIDv4(value) {
			t.Fatalf("validUUIDv4(%q) = true", value)
		}
	}
}

func TestSameBookingStateRequiresSameVersionStatusAndAssignment(t *testing.T) {
	sessionA := "b1111111-1111-4111-8111-111111111111"
	sessionB := "c1111111-1111-4111-8111-111111111111"
	observed := BookingRequest{Version: 3, Status: StatusScheduled, AssignedSessionID: &sessionA}

	for _, test := range []struct {
		name    string
		current BookingRequest
		want    bool
	}{
		{"identical", observed, true},
		{"version changed", BookingRequest{Version: 4, Status: StatusScheduled, AssignedSessionID: &sessionA}, false},
		{"status changed", BookingRequest{Version: 3, Status: StatusCancelled, AssignedSessionID: nil}, false},
		{"assignment changed", BookingRequest{Version: 3, Status: StatusScheduled, AssignedSessionID: &sessionB}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := sameBookingState(test.current, observed); got != test.want {
				t.Fatalf("sameBookingState() = %t, want %t", got, test.want)
			}
		})
	}
}
