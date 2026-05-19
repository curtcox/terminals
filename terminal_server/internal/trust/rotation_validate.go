package trust

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

func validateRotationStatements(oldStmt OldKeyRotationStatement, newStmt NewKeyRotationStatement) (oldPub, newPub ed25519.PublicKey, err error) {
	if oldStmt.Schema != "rotation-stmt/1" {
		return nil, nil, errors.New("trust: OldKeyRotationStatement has wrong schema (want rotation-stmt/1)")
	}
	if newStmt.Schema != "rotation-stmt/1" {
		return nil, nil, errors.New("trust: NewKeyRotationStatement has wrong schema (want rotation-stmt/1)")
	}
	if oldStmt.NewKey != newStmt.NewKey {
		return nil, nil, errors.New("trust: new_key mismatch between old and new rotation statements")
	}

	oldPub, err = parseKeyID(oldStmt.OldKey)
	if err != nil {
		return nil, nil, fmt.Errorf("trust: rotation old_key: %w", err)
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
		return nil, nil, fmt.Errorf("trust: marshal old rotation statement: %w", err)
	}
	if !ed25519.Verify(oldPub, oldPayload, oldStmt.SigOld) {
		return nil, nil, errors.New("trust: old rotation statement signature verification failed")
	}

	oldDigest := "sha256:" + hex.EncodeToString(sha256Sum(oldPayload))
	if newStmt.OldKeyStmtDigest != oldDigest {
		return nil, nil, fmt.Errorf("trust: new rotation statement old_key_stmt_digest mismatch: got %s want %s",
			newStmt.OldKeyStmtDigest, oldDigest)
	}

	newPub, err = parseKeyID(newStmt.NewKey)
	if err != nil {
		return nil, nil, fmt.Errorf("trust: rotation new_key: %w", err)
	}
	newPayload, err := json.Marshal(struct {
		Schema    string `json:"schema"`
		OldDigest string `json:"old_key_stmt_digest"`
		NewKey    string `json:"new_key"`
		AcceptAt  int64  `json:"accept_at"`
	}{newStmt.Schema, newStmt.OldKeyStmtDigest, newStmt.NewKey, newStmt.AcceptAt})
	if err != nil {
		return nil, nil, fmt.Errorf("trust: marshal new rotation statement: %w", err)
	}
	if !ed25519.Verify(newPub, newPayload, newStmt.SigNew) {
		return nil, nil, errors.New("trust: new rotation statement signature verification failed (only old key signature is not sufficient)")
	}
	return oldPub, newPub, nil
}

func sha256Sum(payload []byte) []byte {
	h := sha256.Sum256(payload)
	return h[:]
}
