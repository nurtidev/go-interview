package main

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/nurtilek/go-interview/internal/auth"
	"github.com/nurtilek/go-interview/internal/store"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "seed-test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestSeedUsers_CreatesMissingAccounts(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	seedUsers(ctx, st, "one@goprep.dev:password123;two@goprep.dev:password456", testLogger())

	u1, err := st.GetUserByEmail(ctx, "one@goprep.dev")
	if err != nil {
		t.Fatalf("expected one@goprep.dev to be created: %v", err)
	}
	if !auth.CheckPassword(u1.PasswordHash, "password123") {
		t.Error("password for one@goprep.dev does not match the seeded password")
	}

	u2, err := st.GetUserByEmail(ctx, "two@goprep.dev")
	if err != nil {
		t.Fatalf("expected two@goprep.dev to be created: %v", err)
	}
	if !auth.CheckPassword(u2.PasswordHash, "password456") {
		t.Error("password for two@goprep.dev does not match the seeded password")
	}
}

func TestSeedUsers_DoesNotTouchExistingAccounts(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	existingHash, err := auth.HashPassword("original-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := st.CreateUser(ctx, "existing@goprep.dev", existingHash); err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	seedUsers(ctx, st, "existing@goprep.dev:some-other-password", testLogger())

	u, err := st.GetUserByEmail(ctx, "existing@goprep.dev")
	if err != nil {
		t.Fatalf("get existing user: %v", err)
	}
	if u.PasswordHash != existingHash {
		t.Error("seedUsers overwrote the password hash of an already-existing account")
	}
	if auth.CheckPassword(u.PasswordHash, "some-other-password") {
		t.Error("existing account's password was replaced by the seed password")
	}
	if !auth.CheckPassword(u.PasswordHash, "original-password") {
		t.Error("existing account's original password no longer matches")
	}
}

func TestSeedUsers_MixOfExistingAndMissing(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	existingHash, err := auth.HashPassword("keep-me")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := st.CreateUser(ctx, "existing@goprep.dev", existingHash); err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	seedUsers(ctx, st, "existing@goprep.dev:ignored;fresh@goprep.dev:brandnew123", testLogger())

	u, err := st.GetUserByEmail(ctx, "existing@goprep.dev")
	if err != nil {
		t.Fatalf("get existing user: %v", err)
	}
	if u.PasswordHash != existingHash {
		t.Error("existing account was modified")
	}

	fresh, err := st.GetUserByEmail(ctx, "fresh@goprep.dev")
	if err != nil {
		t.Fatalf("expected fresh@goprep.dev to be created: %v", err)
	}
	if !auth.CheckPassword(fresh.PasswordHash, "brandnew123") {
		t.Error("password for fresh@goprep.dev does not match the seeded password")
	}
}

func TestSeedUsers_EmptySpecIsNoop(t *testing.T) {
	st := openTestStore(t)
	seedUsers(context.Background(), st, "", testLogger())
	// No assertion beyond "does not panic or error" — nothing to seed.
}

func TestSeedUsers_MalformedEntrySkipped(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	// "no-colon-here" has no ':' separator and must be skipped without
	// affecting the well-formed entry alongside it.
	seedUsers(ctx, st, "no-colon-here;good@goprep.dev:password123", testLogger())

	if _, err := st.GetUserByEmail(ctx, "good@goprep.dev"); err != nil {
		t.Fatalf("expected good@goprep.dev to be created despite a malformed sibling entry: %v", err)
	}
}
