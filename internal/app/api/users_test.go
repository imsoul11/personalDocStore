package api

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/imsoul11/personalDocStore/internal/pkg/models"
	"github.com/imsoul11/personalDocStore/restapi/operations"
)

func TestApplyUserProfileUpdatePreservesOmittedFields(t *testing.T) {
	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	user := &models.User{
		ID:        7,
		Email:     "user@example.com",
		Name:      "Existing Name",
		Dob:       "1990-01-01",
		Address:   "Old Address",
		CreatedAt: createdAt,
	}

	updated, changed := applyUserProfileUpdate(user, operations.PostUsersProfileBody{
		Name: "  New Name  ",
	})
	if !changed {
		t.Fatal("expected update to be detected")
	}
	if updated.Name != "New Name" {
		t.Fatalf("expected trimmed name to be applied, got %q", updated.Name)
	}
	if updated.Address != user.Address {
		t.Fatalf("expected address to be preserved, got %q", updated.Address)
	}
	if updated.Dob != user.Dob {
		t.Fatalf("expected dob to be preserved, got %q", updated.Dob)
	}
}

func TestApplyUserProfileUpdateRejectsEmptyPatch(t *testing.T) {
	user := &models.User{
		ID:      7,
		Email:   "user@example.com",
		Name:    "Existing Name",
		Dob:     "1990-01-01",
		Address: "Old Address",
	}

	updated, changed := applyUserProfileUpdate(user, operations.PostUsersProfileBody{})
	if changed {
		t.Fatal("expected empty patch to be ignored")
	}
	if updated.Name != user.Name || updated.Address != user.Address || updated.Dob != user.Dob {
		t.Fatal("expected empty patch to leave user unchanged")
	}
}

func TestUserProfileResponsePayload(t *testing.T) {
	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	user := &models.User{
		ID:        11,
		Email:     "user@example.com",
		Name:      "Existing Name",
		Dob:       "1990-01-01",
		Address:   "Old Address",
		CreatedAt: createdAt,
	}

	payload := userProfileResponsePayload("Profile fetched successfully", user)
	if payload.User == nil {
		t.Fatal("expected user payload")
	}
	if payload.User.Email == nil || string(*payload.User.Email) != user.Email {
		t.Fatalf("expected email %q", user.Email)
	}
	if payload.User.CreatedAt == nil || *payload.User.CreatedAt != strfmt.DateTime(createdAt) {
		t.Fatalf("expected created_at %v", createdAt)
	}
}
