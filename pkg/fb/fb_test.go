package fb

import (
	"errors"
	"fmt"
	"net/http"
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

func TestErrorTreeContainsWrappedError(t *testing.T) {
	target := errors.New("target")
	err := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", target))

	if !errorTreeContains(err, func(candidate error) bool {
		return candidate == target
	}) {
		t.Fatal("errorTreeContains() = false, want true for wrapped target")
	}
}

func TestErrorTreeContainsJoinedError(t *testing.T) {
	target := errors.New("target")
	err := errors.Join(errors.New("other"), fmt.Errorf("wrapped: %w", target))

	if !errorTreeContains(err, func(candidate error) bool {
		return candidate == target
	}) {
		t.Fatal("errorTreeContains() = false, want true for joined target")
	}
}

func TestIsUnregisteredUsesStructuredSendClassification(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", &SendError{
		Message:      "NotRegistered",
		Unregistered: true,
	})

	if !IsUnregistered(err) {
		t.Fatal("IsUnregistered() = false, want true")
	}
}

func TestErrorMessageAloneDoesNotClassifyTokenAsUnregistered(t *testing.T) {
	err := &SendError{
		HTTPStatus:   http.StatusNotFound,
		Status:       "NOT_FOUND",
		FCMErrorCode: "",
		Message:      "NotRegistered",
		Unregistered: false,
	}

	if IsUnregistered(err) {
		t.Fatal("IsUnregistered() = true for message-only error")
	}
}
