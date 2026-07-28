package fb

import (
	"strings"
	"testing"
)

func TestFirebaseServiceAccountValidate(t *testing.T) {
	valid := FirebaseServiceAccount{
		Type:        "service_account",
		ProjectID:   "yevmiye",
		PrivateKey:  "private-key",
		ClientEmail: "firebase-admin@example.iam.gserviceaccount.com",
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestFirebaseServiceAccountRejectsAuthorizedUser(t *testing.T) {
	creds := FirebaseServiceAccount{
		Type:        "authorized_user",
		ProjectID:   "yevmiye",
		PrivateKey:  "private-key",
		ClientEmail: "user@example.com",
	}

	err := creds.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want credential type error")
	}
	if !strings.Contains(err.Error(), "service_account") {
		t.Fatalf("Validate() error = %q, want service_account type error", err)
	}
}
