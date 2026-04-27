package trust

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Service is the in-memory trust service with a freshly-generated installer key.
// All public methods are safe for concurrent use.
type Service struct {
	mu           sync.RWMutex
	keys         map[string]*KeyRecord  // keyed by key_id
	lineage      map[string]*AppLineage // keyed by app_id
	rotations    []*RotationRecord
	log          []*LogEntry
	installerPub ed25519.PublicKey
	installerPrv ed25519.PrivateKey
	now          func() time.Time
}

// NewService creates an in-memory trust service with a freshly-generated installer key.
func NewService() *Service {
	pub, prv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic("trust: failed to generate installer key: " + err.Error())
	}
	return &Service{
		keys:         make(map[string]*KeyRecord),
		lineage:      make(map[string]*AppLineage),
		installerPub: pub,
		installerPrv: prv,
		now:          time.Now,
	}
}

// InstallerKeyID returns the key_id for the installer key.
func (s *Service) InstallerKeyID() string {
	return "ed25519:" + base64.RawURLEncoding.EncodeToString(s.installerPub)
}

// appendLog appends a signed log entry and returns it. Must be called with s.mu held.
func (s *Service) appendLog(actor, op string, args map[string]any) *LogEntry {
	seq := int64(len(s.log) + 1)
	at := s.now().Unix()
	prevHash := "sha256:" + hex.EncodeToString(sha256.New().Sum(nil)) // genesis sentinel
	if len(s.log) > 0 {
		prevHash = s.log[len(s.log)-1].ThisHash
	}
	thisHash := computeLogHash(seq, at, actor, op, args, prevHash)
	sig := ed25519.Sign(s.installerPrv, []byte(thisHash))
	e := &LogEntry{
		Seq:            seq,
		At:             at,
		Actor:          actor,
		Op:             op,
		Args:           args,
		PrevHash:       prevHash,
		ThisHash:       thisHash,
		InstallerSig:   base64.StdEncoding.EncodeToString(sig),
		InstallerKeyID: "ed25519:" + base64.RawURLEncoding.EncodeToString(s.installerPub),
	}
	s.log = append(s.log, e)
	return e
}

// AddKey adds a key to the trust store in state "candidate".
// The caller must call ConfirmKey to move it to "active".
func (s *Service) AddKey(keyID string, roles []string, ceiling *VoucherCeiling, note string) error {
	pub, err := parseKeyID(keyID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.keys[keyID]; exists {
		return fmt.Errorf("trust: key %s is already in the store", keyID)
	}
	rec := &KeyRecord{
		KeyID:           keyID,
		Roles:           append([]string{}, roles...),
		State:           StateCandidate,
		Ceiling:         ceiling,
		FirstObservedAt: s.now().Unix(),
		Note:            note,
		PubKey:          pub,
	}
	s.keys[keyID] = rec
	s.appendLog("operator", "keys.add", map[string]any{"key_id": keyID, "roles": roles})
	return nil
}

// ConfirmKey moves a candidate key to active. It is a mutating operation; for
// author keys it is critical_mutating (the caller is responsible for enforcing that).
func (s *Service) ConfirmKey(keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.keys[keyID]
	if !ok {
		return fmt.Errorf("trust: key %s not found", keyID)
	}
	if rec.State != StateCandidate {
		return fmt.Errorf("trust: key %s is in state %q, not candidate", keyID, rec.State)
	}
	rec.State = StateActive
	s.appendLog("operator", "keys.confirm", map[string]any{"key_id": keyID})
	return nil
}

// RevokeKey moves an active key to revoked and runs the consequence state
// machine for all dependent app lineages.
// Returns a list of app_ids that were moved to pending-revet.
func (s *Service) RevokeKey(keyID, reason string) (affectedAppIDs []string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("trust: key %s not found", keyID)
	}
	if rec.State == StateRevoked {
		return nil, fmt.Errorf("trust: key %s is already revoked", keyID)
	}
	rec.State = StateRevoked
	s.appendLog("operator", "keys.revoke", map[string]any{"key_id": keyID, "reason": reason})

	// Run consequence state machine for all affected lineages.
	for _, lin := range s.lineage {
		if lin.CurrentAuthorKeyID() == keyID {
			// §5.2: author revocation → quarantined-revoked (v1: disabled)
			lin.State = AppStatePendingRevet
			affectedAppIDs = append(affectedAppIDs, lin.AppID)
			s.appendLog("system", "lineage.pending_revet", map[string]any{
				"app_id": lin.AppID, "trigger_key": keyID,
			})
		}
	}
	return affectedAppIDs, nil
}

// ArchiveKey moves a non-active key to archived.
func (s *Service) ArchiveKey(keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.keys[keyID]
	if !ok {
		return fmt.Errorf("trust: key %s not found", keyID)
	}
	if rec.State == StateActive {
		return fmt.Errorf("trust: archiving an active key is critical_mutating; use RevokeKey first or set state explicitly")
	}
	if rec.State == StateArchived {
		return fmt.Errorf("trust: key %s is already archived", keyID)
	}
	rec.State = StateArchived
	s.appendLog("operator", "keys.archive", map[string]any{"key_id": keyID})
	return nil
}

// RotateAccept accepts a pair of rotation statements and records the rotation.
// Both statements must be cryptographically valid and the old key must be active.
// Returns an error if either statement is invalid or only one is presented.
//
// This is a critical_mutating operation; the caller is responsible for enforcing
// operator confirmation before calling.
func (s *Service) RotateAccept(oldStmt OldKeyRotationStatement, newStmt NewKeyRotationStatement) (*RotationRecord, error) {
	if oldStmt.Schema != "rotation-stmt/1" {
		return nil, errors.New("trust: OldKeyRotationStatement has wrong schema (want rotation-stmt/1)")
	}
	if newStmt.Schema != "rotation-stmt/1" {
		return nil, errors.New("trust: NewKeyRotationStatement has wrong schema (want rotation-stmt/1)")
	}
	if oldStmt.NewKey != newStmt.NewKey {
		return nil, errors.New("trust: new_key mismatch between old and new rotation statements")
	}

	// Parse and verify old key signature.
	oldPub, err := parseKeyID(oldStmt.OldKey)
	if err != nil {
		return nil, fmt.Errorf("trust: rotation old_key: %w", err)
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
		return nil, fmt.Errorf("trust: marshal old rotation statement: %w", err)
	}
	if !ed25519.Verify(oldPub, oldPayload, oldStmt.SigOld) {
		return nil, errors.New("trust: old rotation statement signature verification failed")
	}

	// Verify the new statement's digest matches the old statement payload.
	oldDigest := "sha256:" + hex.EncodeToString(func() []byte { h := sha256.Sum256(oldPayload); return h[:] }())
	if newStmt.OldKeyStmtDigest != oldDigest {
		return nil, fmt.Errorf("trust: new rotation statement old_key_stmt_digest mismatch: got %s want %s",
			newStmt.OldKeyStmtDigest, oldDigest)
	}

	// Parse and verify new key signature.
	newPub, err := parseKeyID(newStmt.NewKey)
	if err != nil {
		return nil, fmt.Errorf("trust: rotation new_key: %w", err)
	}
	newPayload, err := json.Marshal(struct {
		Schema    string `json:"schema"`
		OldDigest string `json:"old_key_stmt_digest"`
		NewKey    string `json:"new_key"`
		AcceptAt  int64  `json:"accept_at"`
	}{newStmt.Schema, newStmt.OldKeyStmtDigest, newStmt.NewKey, newStmt.AcceptAt})
	if err != nil {
		return nil, fmt.Errorf("trust: marshal new rotation statement: %w", err)
	}
	if !ed25519.Verify(newPub, newPayload, newStmt.SigNew) {
		return nil, errors.New("trust: new rotation statement signature verification failed (only old key signature is not sufficient)")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldRec, ok := s.keys[oldStmt.OldKey]
	if !ok {
		return nil, fmt.Errorf("trust: old key %s not found in trust store", oldStmt.OldKey)
	}
	if oldRec.State != StateActive {
		return nil, fmt.Errorf("trust: old key %s is in state %q, not active", oldStmt.OldKey, oldRec.State)
	}

	// Add new key as active (inheriting author role from old key).
	if _, exists := s.keys[newStmt.NewKey]; !exists {
		s.keys[newStmt.NewKey] = &KeyRecord{
			KeyID:           newStmt.NewKey,
			Roles:           append([]string{}, oldRec.Roles...),
			State:           StateActive,
			FirstObservedAt: s.now().Unix(),
			PubKey:          newPub,
		}
	}

	// Move old key to rotated.
	oldRec.State = StateRotated

	// Record the rotation with the current log sequence as the cutoff.
	acceptSeq := int64(len(s.log) + 1) // will be this entry's seq
	acceptAt := s.now().Unix()
	s.appendLog("operator", "keys.rotate.accept", map[string]any{
		"old_key":    oldStmt.OldKey,
		"new_key":    newStmt.NewKey,
		"name_scope": oldStmt.NameScope,
	})

	rot := &RotationRecord{
		OldKeyID:    oldStmt.OldKey,
		NewKeyID:    newStmt.NewKey,
		NameScope:   append([]string{}, oldStmt.NameScope...),
		AcceptedSeq: acceptSeq,
		AcceptedAt:  acceptAt,
	}
	s.rotations = append(s.rotations, rot)

	// Update lineage map: for each named app, append new key edge. §4.3.
	for _, lin := range s.lineage {
		for _, name := range oldStmt.NameScope {
			if lin.Name == name && lin.CurrentAuthorKeyID() == oldStmt.OldKey {
				lin.Edges = append(lin.Edges, LineageEdge{
					AuthorKeyID: newStmt.NewKey,
					AddedAt:     acceptAt,
				})
				break
			}
		}
	}

	return rot, nil
}

// IsKeyAccepted reports whether a statement from keyID is accepted at the given
// log sequence number. A rotated key is only accepted for statements that were
// first observed before its rotation-acceptance sequence (per §1.3 / §4.4).
//
// observedSeq should be the log sequence at which the statement was first seen.
// Pass -1 to use the current sequence (i.e. "just received").
func (s *Service) IsKeyAccepted(keyID string, observedSeq int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.keys[keyID]
	if !ok {
		return false
	}
	switch rec.State {
	case StateActive:
		return true
	case StateRotated:
		if observedSeq < 0 {
			// Just-received: current seq is after rotation → reject.
			return false
		}
		for _, rot := range s.rotations {
			if rot.OldKeyID == keyID && observedSeq < rot.AcceptedSeq {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// GetKey returns a copy of the key record, or an error if not found.
func (s *Service) GetKey(keyID string) (*KeyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("trust: key %s not found", keyID)
	}
	recCopy := *rec
	return &recCopy, nil
}

// ListKeys returns copies of all key records.
func (s *Service) ListKeys() []*KeyRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*KeyRecord, 0, len(s.keys))
	for _, rec := range s.keys {
		c := *rec
		out = append(out, &c)
	}
	return out
}

// RecordLineage records the first install of an app and binds its app_id.
// The firstAuthorKeyID must be active in the store.
func (s *Service) RecordLineage(manifestName, firstAuthorKeyID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.keys[firstAuthorKeyID]
	if !ok {
		return "", fmt.Errorf("trust: first author key %s not found", firstAuthorKeyID)
	}
	if rec.State != StateActive {
		return "", fmt.Errorf("trust: first author key %s is not active (state=%s)", firstAuthorKeyID, rec.State)
	}
	appID := ComputeAppID(firstAuthorKeyID, manifestName)
	if _, exists := s.lineage[appID]; exists {
		return appID, nil // idempotent
	}
	s.lineage[appID] = &AppLineage{
		AppID: appID,
		Name:  manifestName,
		Edges: []LineageEdge{{AuthorKeyID: firstAuthorKeyID, AddedAt: s.now().Unix()}},
		State: AppStateActive,
	}
	s.appendLog("system", "lineage.created", map[string]any{
		"app_id": appID, "manifest_name": manifestName, "author_key": firstAuthorKeyID,
	})
	return appID, nil
}

// GetLineage returns the lineage for an app_id.
func (s *Service) GetLineage(appID string) (*AppLineage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lin, ok := s.lineage[appID]
	if !ok {
		return nil, fmt.Errorf("trust: lineage for app_id %s not found", appID)
	}
	c := *lin
	c.Edges = append([]LineageEdge{}, lin.Edges...)
	return &c, nil
}

// VerifyChain verifies the integrity of the trust log hash chain and all
// installer signatures. Returns nil if the chain is intact.
func (s *Service) VerifyChain() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.log) == 0 {
		return nil
	}
	for i, e := range s.log {
		// Recompute this_hash.
		want := computeLogHash(e.Seq, e.At, e.Actor, e.Op, e.Args, e.PrevHash)
		if e.ThisHash != want {
			return fmt.Errorf("trust: log entry %d hash mismatch: got %s want %s", i+1, e.ThisHash, want)
		}
		// Verify prev_hash chain.
		if i > 0 && e.PrevHash != s.log[i-1].ThisHash {
			return fmt.Errorf("trust: log chain broken at entry %d: prev_hash mismatch", i+1)
		}
		// Verify installer signature over this_hash using the key that was active
		// when the entry was appended (identified by InstallerKeyID).
		sigBytes, err := base64.StdEncoding.DecodeString(e.InstallerSig)
		if err != nil {
			return fmt.Errorf("trust: log entry %d installer_sig is not valid base64: %w", i+1, err)
		}
		// Determine the public key for this entry.
		var verifyPub ed25519.PublicKey
		if e.InstallerKeyID == "" {
			// Legacy entries without a key ID: use the current installer key.
			verifyPub = s.installerPub
		} else {
			rec, ok := s.keys[e.InstallerKeyID]
			if !ok {
				// Fall back to parsing from the key ID itself.
				parsed, parseErr := parseKeyID(e.InstallerKeyID)
				if parseErr != nil {
					return fmt.Errorf("trust: log entry %d references unknown installer key %s", i+1, e.InstallerKeyID)
				}
				verifyPub = parsed
			} else {
				verifyPub = rec.PubKey
			}
		}
		if !ed25519.Verify(verifyPub, []byte(e.ThisHash), sigBytes) {
			return fmt.Errorf("trust: log entry %d installer signature verification failed", i+1)
		}
	}
	return nil
}

// LogEntries returns a copy of all log entries.
func (s *Service) LogEntries() []*LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*LogEntry, len(s.log))
	for i, e := range s.log {
		c := *e
		out[i] = &c
	}
	return out
}

// RotateInstallerKey generates a new installer key pair, adds the old key to the
// store as archived, and begins signing all future log entries with the new key.
// The new installer key ID is returned.
// This is a critical_mutating operation; the caller is responsible for enforcing
// operator confirmation before calling.
func (s *Service) RotateInstallerKey() (newKeyID string, err error) {
	newPub, newPrv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("trust: failed to generate new installer key: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	oldKeyID := "ed25519:" + base64.RawURLEncoding.EncodeToString(s.installerPub)
	newKeyIDStr := "ed25519:" + base64.RawURLEncoding.EncodeToString(newPub)

	// Archive the old installer key in the key store.
	if _, exists := s.keys[oldKeyID]; !exists {
		s.keys[oldKeyID] = &KeyRecord{
			KeyID:           oldKeyID,
			Roles:           []string{"installer"},
			State:           StateArchived,
			FirstObservedAt: s.now().Unix(),
			PubKey:          s.installerPub,
		}
	} else {
		s.keys[oldKeyID].State = StateArchived
	}

	// Register the new installer key.
	s.keys[newKeyIDStr] = &KeyRecord{
		KeyID:           newKeyIDStr,
		Roles:           []string{"installer"},
		State:           StateActive,
		FirstObservedAt: s.now().Unix(),
		PubKey:          newPub,
	}

	// Log the rotation before switching keys (signed by the old key).
	s.appendLog("operator", "installer.rotate", map[string]any{
		"old_key": oldKeyID,
		"new_key": newKeyIDStr,
	})

	// Switch to the new key for all future log entries.
	s.installerPub = newPub
	s.installerPrv = newPrv

	return newKeyIDStr, nil
}

// RollbackRotation reverses an author-key rotation that was accepted at the given
// log sequence number. The new key is moved to revoked and the old key is restored
// to active, provided the old key is currently in state rotated and was accepted
// at exactly acceptedSeq.
// This is a critical_mutating operation; the caller is responsible for enforcing
// operator confirmation before calling.
func (s *Service) RollbackRotation(acceptedSeq int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var rot *RotationRecord
	for _, r := range s.rotations {
		if r.AcceptedSeq == acceptedSeq {
			rot = r
			break
		}
	}
	if rot == nil {
		return fmt.Errorf("trust: no rotation found with accepted_seq %d", acceptedSeq)
	}
	oldRec, ok := s.keys[rot.OldKeyID]
	if !ok {
		return fmt.Errorf("trust: old key %s not found after rollback lookup", rot.OldKeyID)
	}
	if oldRec.State != StateRotated {
		return fmt.Errorf("trust: old key %s is in state %q, expected rotated; rollback not safe", rot.OldKeyID, oldRec.State)
	}
	newRec, ok := s.keys[rot.NewKeyID]
	if !ok {
		return fmt.Errorf("trust: new key %s not found after rollback lookup", rot.NewKeyID)
	}

	// Restore old key to active; revoke the new key.
	oldRec.State = StateActive
	newRec.State = StateRevoked

	// Remove the lineage edges that were added by this rotation.
	for _, lin := range s.lineage {
		edges := lin.Edges[:0]
		for _, e := range lin.Edges {
			if e.AuthorKeyID == rot.NewKeyID && e.AddedAt == rot.AcceptedAt {
				continue // remove this edge
			}
			edges = append(edges, e)
		}
		lin.Edges = edges
	}

	// Remove the rotation record.
	filtered := s.rotations[:0]
	for _, r := range s.rotations {
		if r.AcceptedSeq != acceptedSeq {
			filtered = append(filtered, r)
		}
	}
	s.rotations = filtered

	s.appendLog("operator", "keys.rotate.rollback", map[string]any{
		"rolled_back_seq": acceptedSeq,
		"old_key":         rot.OldKeyID,
		"new_key":         rot.NewKeyID,
	})
	return nil
}

// Rotations returns a copy of all rotation records.
func (s *Service) Rotations() []*RotationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*RotationRecord, len(s.rotations))
	for i, r := range s.rotations {
		c := *r
		out[i] = &c
	}
	return out
}
