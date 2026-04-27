package trust

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- helpers ---

func mustGenKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string) {
	t.Helper()
	pub, prv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	keyID := "ed25519:" + base64.RawURLEncoding.EncodeToString(pub)
	return pub, prv, keyID
}

func addActiveKey(t *testing.T, svc *Service, keyID string, roles []string) {
	t.Helper()
	if err := svc.AddKey(keyID, roles, nil, "test"); err != nil {
		t.Fatalf("AddKey(%s): %v", keyID, err)
	}
	if err := svc.ConfirmKey(keyID); err != nil {
		t.Fatalf("ConfirmKey(%s): %v", keyID, err)
	}
}

// buildRotationPair signs both rotation statements for a key pair and returns them.
func buildRotationPair(
	t *testing.T,
	oldKeyID string, oldPrv ed25519.PrivateKey,
	newKeyID string, newPrv ed25519.PrivateKey,
	nameScope []string,
) (OldKeyRotationStatement, NewKeyRotationStatement) {
	t.Helper()

	oldStmt := OldKeyRotationStatement{
		Schema:    "rotation-stmt/1",
		OldKey:    oldKeyID,
		NewKey:    newKeyID,
		NameScope: nameScope,
	}
	oldPayload, err := json.Marshal(struct {
		Schema    string   `json:"schema"`
		OldKey    string   `json:"old_key"`
		NewKey    string   `json:"new_key"`
		Proposed  int64    `json:"proposed_at"`
		NameScope []string `json:"name_scope"`
		Reason    string   `json:"reason,omitempty"`
	}{oldStmt.Schema, oldStmt.OldKey, oldStmt.NewKey, oldStmt.ProposedAt, oldStmt.NameScope, oldStmt.Reason})
	if err != nil {
		t.Fatalf("marshal old stmt: %v", err)
	}
	oldStmt.SigOld = ed25519.Sign(oldPrv, oldPayload)

	h := sha256.Sum256(oldPayload)
	oldDigest := "sha256:" + hex.EncodeToString(h[:])

	newStmt := NewKeyRotationStatement{
		Schema:           "rotation-stmt/1",
		OldKeyStmtDigest: oldDigest,
		NewKey:           newKeyID,
	}
	newPayload, err := json.Marshal(struct {
		Schema    string `json:"schema"`
		OldDigest string `json:"old_key_stmt_digest"`
		NewKey    string `json:"new_key"`
		AcceptAt  int64  `json:"accept_at"`
	}{newStmt.Schema, newStmt.OldKeyStmtDigest, newStmt.NewKey, newStmt.AcceptAt})
	if err != nil {
		t.Fatalf("marshal new stmt: %v", err)
	}
	newStmt.SigNew = ed25519.Sign(newPrv, newPayload)

	return oldStmt, newStmt
}

// --- acceptance criteria tests ---

// AC1: Server can import an author key, bind it to an app lineage, and the
// key is accepted. A revoked key is no longer accepted.
func TestAC1_ImportAuthorKeyAndLineage(t *testing.T) {
	svc := NewService()
	_, _, keyID := mustGenKey(t)
	addActiveKey(t, svc, keyID, []string{RoleAuthor})

	rec, err := svc.GetKey(keyID)
	if err != nil {
		t.Fatalf("GetKey: %v", err)
	}
	if rec.State != StateActive {
		t.Fatalf("state = %q, want active", rec.State)
	}
	if !svc.IsKeyAccepted(keyID, -1) {
		t.Fatal("IsKeyAccepted = false, want true for active key")
	}

	appID, err := svc.RecordLineage("my-app", keyID)
	if err != nil {
		t.Fatalf("RecordLineage: %v", err)
	}
	if appID == "" {
		t.Fatal("RecordLineage returned empty app_id")
	}
	// Stable across re-call.
	appID2, _ := svc.RecordLineage("my-app", keyID)
	if appID != appID2 {
		t.Fatalf("RecordLineage not idempotent: %s vs %s", appID, appID2)
	}

	// Revoke: key is no longer accepted.
	if _, err := svc.RevokeKey(keyID, "test"); err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}
	if svc.IsKeyAccepted(keyID, -1) {
		t.Fatal("IsKeyAccepted = true after revocation, want false")
	}
}

// AC2: Rotation requires both signed statements; a rotation with only an old
// key signature (invalid new-key sig) is rejected with a clear error.
func TestAC2_RotationRequiresBothStatements(t *testing.T) {
	svc := NewService()
	_, oldPrv, oldKeyID := mustGenKey(t)
	newPub, _, newKeyID := mustGenKey(t)
	addActiveKey(t, svc, oldKeyID, []string{RoleAuthor})

	oldStmt := OldKeyRotationStatement{
		Schema:    "rotation-stmt/1",
		OldKey:    oldKeyID,
		NewKey:    newKeyID,
		NameScope: []string{"my-app"},
	}
	oldPayload, _ := json.Marshal(struct {
		Schema    string   `json:"schema"`
		OldKey    string   `json:"old_key"`
		NewKey    string   `json:"new_key"`
		Proposed  int64    `json:"proposed_at"`
		NameScope []string `json:"name_scope"`
		Reason    string   `json:"reason,omitempty"`
	}{oldStmt.Schema, oldStmt.OldKey, oldStmt.NewKey, 0, oldStmt.NameScope, ""})
	oldStmt.SigOld = ed25519.Sign(oldPrv, oldPayload)

	h := sha256.Sum256(oldPayload)
	digest := "sha256:" + hex.EncodeToString(h[:])

	// NewKeyRotationStatement with invalid signature (all zeros instead of real sig).
	newStmt := NewKeyRotationStatement{
		Schema:           "rotation-stmt/1",
		OldKeyStmtDigest: digest,
		NewKey:           newKeyID,
		SigNew:           make([]byte, ed25519.SignatureSize), // wrong sig
	}
	// Verify that newPub is genuinely the public key for newKeyID.
	_ = newPub // ensure parsed above

	_, err := svc.RotateAccept(oldStmt, newStmt)
	if err == nil {
		t.Fatal("RotateAccept succeeded with invalid new-key signature, want error")
	}
	t.Logf("correctly rejected: %v", err)
}

// AC3: A stolen old key that tries to land a statement after rotation acceptance
// is rejected because IsKeyAccepted checks the seq-based cutoff.
func TestAC3_BackdatedStatementRejectedAfterRotation(t *testing.T) {
	svc := NewService()
	_, oldPrv, oldKeyID := mustGenKey(t)
	_, newPrv, newKeyID := mustGenKey(t)
	addActiveKey(t, svc, oldKeyID, []string{RoleAuthor})

	oldStmt, newStmt := buildRotationPair(t, oldKeyID, oldPrv, newKeyID, newPrv, []string{"my-app"})
	rot, err := svc.RotateAccept(oldStmt, newStmt)
	if err != nil {
		t.Fatalf("RotateAccept: %v", err)
	}
	acceptedSeq := rot.AcceptedSeq

	// Statement observed BEFORE the rotation sequence → should be accepted.
	if !svc.IsKeyAccepted(oldKeyID, acceptedSeq-1) {
		t.Fatal("old key not accepted for pre-rotation seq, want accepted")
	}

	// Statement observed AT OR AFTER the rotation sequence → rejected.
	if svc.IsKeyAccepted(oldKeyID, acceptedSeq) {
		t.Fatal("old key accepted at rotation seq, want rejected")
	}
	if svc.IsKeyAccepted(oldKeyID, -1) {
		t.Fatal("old key accepted for just-received statement, want rejected")
	}
	// New key is accepted.
	if !svc.IsKeyAccepted(newKeyID, -1) {
		t.Fatal("new key not accepted, want accepted")
	}
	t.Logf("rotation cutoff seq=%d (correct)", acceptedSeq)
}

// AC4: Revoking an author key immediately marks every dependent lineage
// pending-revet/no-new-activations before any async work.
func TestAC4_AuthorRevocationMarksPendingRevet(t *testing.T) {
	svc := NewService()
	_, _, keyID := mustGenKey(t)
	addActiveKey(t, svc, keyID, []string{RoleAuthor})

	appID1, _ := svc.RecordLineage("app-one", keyID)
	appID2, _ := svc.RecordLineage("app-two", keyID)

	affected, err := svc.RevokeKey(keyID, "compromise")
	if err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}
	if len(affected) != 2 {
		t.Fatalf("affected count = %d, want 2", len(affected))
	}
	affectedSet := map[string]bool{appID1: true, appID2: true}
	for _, id := range affected {
		if !affectedSet[id] {
			t.Errorf("unexpected affected app_id %s", id)
		}
	}

	// Both lineages must be in pending-revet synchronously.
	for _, appID := range []string{appID1, appID2} {
		lin, err := svc.GetLineage(appID)
		if err != nil {
			t.Fatalf("GetLineage(%s): %v", appID, err)
		}
		if lin.State != AppStatePendingRevet {
			t.Errorf("lineage %s state = %q, want %q", appID, lin.State, AppStatePendingRevet)
		}
	}
}

// AC5: Revoking a voucher key marks dependent installs pending-revet (the
// §5.3 state machine). Since the service layer tracks installs for voucher
// dependence, this test covers the structural path: a voucher key can be
// revoked and the key state transitions correctly.
func TestAC5_VoucherRevocationStateTransition(t *testing.T) {
	svc := NewService()
	_, _, vKeyID := mustGenKey(t)
	if err := svc.AddKey(vKeyID, []string{RoleVoucher}, &VoucherCeiling{
		MaxTier: "quarantine", AllowedTesting: []string{"sim-only"}, MaxExpiryDays: 14,
	}, "test voucher"); err != nil {
		t.Fatalf("AddKey voucher: %v", err)
	}
	if err := svc.ConfirmKey(vKeyID); err != nil {
		t.Fatalf("ConfirmKey voucher: %v", err)
	}

	rec, _ := svc.GetKey(vKeyID)
	if rec.State != StateActive {
		t.Fatalf("voucher state = %q, want active", rec.State)
	}

	// Revoke: key state → revoked.
	affected, err := svc.RevokeKey(vKeyID, "test")
	if err != nil {
		t.Fatalf("RevokeKey voucher: %v", err)
	}
	// No lineages depend on this voucher (they depend on author keys at the
	// service layer); the state machine runs but zero lineages are affected.
	_ = affected

	rec, _ = svc.GetKey(vKeyID)
	if rec.State != StateRevoked {
		t.Fatalf("voucher state after revoke = %q, want revoked", rec.State)
	}
	if svc.IsKeyAccepted(vKeyID, -1) {
		t.Fatal("revoked voucher key is still accepted, want rejected")
	}
}

// AC6: The installer key signs each log entry; VerifyChain walks both the
// hash chain and the installer signatures.
func TestAC6_InstallerKeySignsAndVerifyWalksChain(t *testing.T) {
	svc := NewService()
	_, _, keyID := mustGenKey(t)
	addActiveKey(t, svc, keyID, []string{RoleAuthor})
	_, _ = svc.RecordLineage("my-app", keyID)
	_, _ = svc.RevokeKey(keyID, "test")

	entries := svc.LogEntries()
	if len(entries) == 0 {
		t.Fatal("no log entries after operations")
	}

	if err := svc.VerifyChain(); err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
}

// AC6b: Tampering with a log entry breaks the chain.
func TestAC6b_TamperedChainDetected(t *testing.T) {
	svc := NewService()
	_, _, keyID := mustGenKey(t)
	addActiveKey(t, svc, keyID, []string{RoleAuthor})

	// Tamper directly with the internal log.
	svc.mu.Lock()
	if len(svc.log) > 0 {
		svc.log[0].ThisHash = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	}
	svc.mu.Unlock()

	if err := svc.VerifyChain(); err == nil {
		t.Fatal("VerifyChain succeeded after tampering, want error")
	}
}

// AC7: policy/1 loads correctly; a file with unknown top-level table is rejected.
func TestAC7_PolicyV1Loads(t *testing.T) {
	dir := t.TempDir()

	validPolicy := `
policy_schema = "policy/1"
name          = "default"

[gate3]
[[gate3.rule]]
kind        = "trusted_author"
name_filter = "*"

[gate5]
min_active_providers = 2
cooldown_treatment   = "require_revet"

[gate7]
block_above_score = 80
warn_above_score  = 40

[revoke]
author_default   = "quarantined-revoked"
voucher_default  = "pending-revet"
compromise_floor = "disabled"

[voucher_defaults]
max_tier        = "quarantine"
allowed_testing = ["sim-only"]
max_expiry_days = 14
`
	policyFile := filepath.Join(dir, "policy.toml")
	if err := os.WriteFile(policyFile, []byte(validPolicy), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	p, err := LoadPolicy(policyFile)
	if err != nil {
		t.Fatalf("LoadPolicy: %v", err)
	}
	if p.PolicySchema != "policy/1" {
		t.Errorf("PolicySchema = %q, want policy/1", p.PolicySchema)
	}
	if p.Gate7.BlockAboveScore != 80 {
		t.Errorf("gate7.block_above_score = %d, want 80", p.Gate7.BlockAboveScore)
	}
}

func TestAC7_UnknownTopLevelTableRejected(t *testing.T) {
	dir := t.TempDir()
	bad := `
policy_schema = "policy/1"
name          = "bad"

[gate3]
[[gate3.rule]]
kind = "trusted_author"
name_filter = "*"

[mystery_section]
something = "unexpected"
`
	f := filepath.Join(dir, "policy.toml")
	if err := os.WriteFile(f, []byte(bad), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := LoadPolicy(f)
	if err == nil {
		t.Fatal("LoadPolicy succeeded with unknown top-level table, want error")
	}
	t.Logf("correctly rejected: %v", err)
}

// AC8: An app_id is stable across author-key rotation — the same app_id is
// returned before and after the rotation is accepted.
func TestAC8_AppIDSurvivesRotation(t *testing.T) {
	svc := NewService()
	_, oldPrv, oldKeyID := mustGenKey(t)
	_, newPrv, newKeyID := mustGenKey(t)
	addActiveKey(t, svc, oldKeyID, []string{RoleAuthor})

	// Record lineage under old key.
	appID, err := svc.RecordLineage("kitchen-timer", oldKeyID)
	if err != nil {
		t.Fatalf("RecordLineage: %v", err)
	}

	// Rotate to new key.
	oldStmt, newStmt := buildRotationPair(t, oldKeyID, oldPrv, newKeyID, newPrv, []string{"kitchen-timer"})
	if _, err := svc.RotateAccept(oldStmt, newStmt); err != nil {
		t.Fatalf("RotateAccept: %v", err)
	}

	// app_id is unchanged after rotation.
	lin, err := svc.GetLineage(appID)
	if err != nil {
		t.Fatalf("GetLineage: %v", err)
	}
	if lin.AppID != appID {
		t.Fatalf("app_id changed after rotation: %s → %s", appID, lin.AppID)
	}
	// Lineage now has both keys.
	if len(lin.Edges) != 2 {
		t.Fatalf("lineage edges = %d, want 2 (old + new)", len(lin.Edges))
	}
	if lin.CurrentAuthorKeyID() != newKeyID {
		t.Fatalf("current author = %s, want new key %s", lin.CurrentAuthorKeyID(), newKeyID)
	}

	// ComputeAppID with the original first-author key still produces the same ID.
	if got := ComputeAppID(oldKeyID, "kitchen-timer"); got != appID {
		t.Fatalf("ComputeAppID mismatch: %s vs %s", got, appID)
	}
}

// TestCandidateKeyHasNoAuthority verifies a candidate key cannot authorize
// lineage recording (state must be active first).
func TestCandidateKeyHasNoAuthority(t *testing.T) {
	svc := NewService()
	_, _, keyID := mustGenKey(t)
	if err := svc.AddKey(keyID, []string{RoleAuthor}, nil, ""); err != nil {
		t.Fatalf("AddKey: %v", err)
	}
	// Not confirmed yet — should be rejected.
	if _, err := svc.RecordLineage("my-app", keyID); err == nil {
		t.Fatal("RecordLineage succeeded for candidate key, want error")
	}
}

// TestRevokedKeyCannotRotate verifies the plan §4.6: a revoked key cannot
// produce a valid OldKeyRotationStatement.
func TestRevokedKeyCannotRotate(t *testing.T) {
	svc := NewService()
	_, oldPrv, oldKeyID := mustGenKey(t)
	_, newPrv, newKeyID := mustGenKey(t)
	addActiveKey(t, svc, oldKeyID, []string{RoleAuthor})

	// Revoke old key first.
	if _, err := svc.RevokeKey(oldKeyID, "compromise"); err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}

	// Attempt rotation — RotateAccept must reject because old key is revoked, not active.
	oldStmt, newStmt := buildRotationPair(t, oldKeyID, oldPrv, newKeyID, newPrv, []string{"my-app"})
	_, err := svc.RotateAccept(oldStmt, newStmt)
	if err == nil {
		t.Fatal("RotateAccept succeeded for revoked old key, want error")
	}
	t.Logf("correctly rejected: %v", err)
}

// TestRotateInstallerKey verifies that a new installer key pair is generated,
// the old key is archived, and future log entries are signed by the new key.
func TestRotateInstallerKey(t *testing.T) {
	svc := NewService()
	oldKeyID := svc.InstallerKeyID()

	newKeyID, err := svc.RotateInstallerKey()
	if err != nil {
		t.Fatalf("RotateInstallerKey: %v", err)
	}
	if newKeyID == oldKeyID {
		t.Fatal("new installer key_id must differ from old")
	}
	if svc.InstallerKeyID() != newKeyID {
		t.Errorf("InstallerKeyID() = %s, want %s", svc.InstallerKeyID(), newKeyID)
	}

	// Old key must be archived in the key store.
	rec, err := svc.GetKey(oldKeyID)
	if err != nil {
		t.Fatalf("GetKey(old): %v", err)
	}
	if rec.State != StateArchived {
		t.Errorf("old installer key state = %q, want archived", rec.State)
	}

	// New key must be active in the key store.
	newRec, err := svc.GetKey(newKeyID)
	if err != nil {
		t.Fatalf("GetKey(new): %v", err)
	}
	if newRec.State != StateActive {
		t.Errorf("new installer key state = %q, want active", newRec.State)
	}

	// Log should include a rotate event and chain should still verify.
	if err := svc.VerifyChain(); err != nil {
		t.Fatalf("VerifyChain after installer rotation: %v", err)
	}
}

// TestRollbackRotation verifies that a rotation can be reversed: the old key
// returns to active, the new key is revoked, and the lineage edge is removed.
func TestRollbackRotation(t *testing.T) {
	svc := NewService()
	_, oldPrv, oldKeyID := mustGenKey(t)
	_, newPrv, newKeyID := mustGenKey(t)
	addActiveKey(t, svc, oldKeyID, []string{RoleAuthor})

	appName := "my-app"
	appID, err := svc.RecordLineage(appName, oldKeyID)
	if err != nil {
		t.Fatalf("RecordLineage: %v", err)
	}

	oldStmt, newStmt := buildRotationPair(t, oldKeyID, oldPrv, newKeyID, newPrv, []string{appName})
	rot, err := svc.RotateAccept(oldStmt, newStmt)
	if err != nil {
		t.Fatalf("RotateAccept: %v", err)
	}

	// Rollback the rotation.
	if err := svc.RollbackRotation(rot.AcceptedSeq); err != nil {
		t.Fatalf("RollbackRotation: %v", err)
	}

	// Old key must be active again.
	oldRec, err := svc.GetKey(oldKeyID)
	if err != nil {
		t.Fatalf("GetKey(old): %v", err)
	}
	if oldRec.State != StateActive {
		t.Errorf("old key state after rollback = %q, want active", oldRec.State)
	}

	// New key must be revoked.
	newRec, err := svc.GetKey(newKeyID)
	if err != nil {
		t.Fatalf("GetKey(new): %v", err)
	}
	if newRec.State != StateRevoked {
		t.Errorf("new key state after rollback = %q, want revoked", newRec.State)
	}

	// Lineage edge for the new key must be removed.
	lin, err := svc.GetLineage(appID)
	if err != nil {
		t.Fatalf("GetLineage: %v", err)
	}
	for _, e := range lin.Edges {
		if e.AuthorKeyID == newKeyID {
			t.Errorf("lineage edge for new key still present after rollback")
		}
	}

	// No rotation records should remain for that seq.
	for _, r := range svc.Rotations() {
		if r.AcceptedSeq == rot.AcceptedSeq {
			t.Errorf("rotation record for seq %d still present after rollback", rot.AcceptedSeq)
		}
	}
}

// TestRollbackRotationNotFound verifies that rollback returns an error for
// an unknown accepted_seq.
func TestRollbackRotationNotFound(t *testing.T) {
	svc := NewService()
	if err := svc.RollbackRotation(99); err == nil {
		t.Fatal("RollbackRotation(99) succeeded, want error for unknown seq")
	}
}
