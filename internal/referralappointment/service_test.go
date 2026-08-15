package referralappointment

import "testing"

func TestTransition(t *testing.T) {
	tests := []struct {
		name    string
		current Status
		command Command
		want    Status
		wantErr bool
	}{
		{"schedule", StatusRequested, CommandSchedule, StatusScheduled, false},
		{"arrive", StatusScheduled, CommandArrive, StatusArrived, false},
		{"complete", StatusArrived, CommandComplete, StatusCompleted, false},
		{"abandon requested", StatusRequested, CommandAbandon, StatusAbandoned, false},
		{"cannot complete requested", StatusRequested, CommandComplete, "", true},
		{"cannot change completed", StatusCompleted, CommandAbandon, "", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := transition(test.current, test.command)
			if (err != nil) != test.wantErr {
				t.Fatalf("error=%v, wantErr=%v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestValidCreateRejectsClinicalOrUnboundedInput(t *testing.T) {
	base := CreateRequest{SyntheticPatientID: "11111111-1111-4111-8111-111111111111", SyntheticPatientLabel: "Demo referral", RecipientRole: "demo_gp", ClinicCode: "demo_general_eye_clinic", AppointmentDate: "2026-08-18", Priority: "routine"}
	if !validCreate(base) {
		t.Fatal("expected demo request to be valid")
	}
	base.Notes = "<script>not allowed</script>"
	if validCreate(base) {
		t.Fatal("expected markup to be rejected")
	}
}
