package auth

import "testing"

func TestPasswordHashAndVerify(t *testing.T) {
	manager := PasswordManager{}
	hash, err := manager.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	valid, err := manager.Verify(hash, "correct horse battery staple")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !valid {
		t.Fatal("Verify() = false, want true")
	}

	valid, err = manager.Verify(hash, "wrong password")
	if err != nil {
		t.Fatalf("Verify() wrong password error = %v", err)
	}
	if valid {
		t.Fatal("Verify() wrong password = true, want false")
	}
}

func TestPasswordVerifyRejectsMalformedOrExcessiveParameters(t *testing.T) {
	tests := []string{
		"not-a-phc-hash",
		"$argon2id$v=19$m=999999999,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXphYmNkZWY",
	}
	for _, encoded := range tests {
		if _, err := (PasswordManager{}).Verify(encoded, "password"); err == nil {
			t.Fatalf("Verify(%q) error = nil, want malformed hash error", encoded)
		}
	}
}
