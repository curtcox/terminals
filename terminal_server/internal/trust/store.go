// Package trust implements the server-local trust store for signing keys,
// voucher authority, app lineage, the installer-key log chain, and the
// policy/1 admission schema.
package trust

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Key states.
const (
	StateCandidate = "candidate" // added, pending operator confirmation; no authority yet
	StateActive    = "active"    // normal working state
	StateRotated   = "rotated"   // superseded; statements accepted only before AcceptedSeq
	StateRevoked   = "revoked"   // rejected unconditionally; consequence state machine runs
	StateArchived  = "archived"  // not trusted for new statements; historical statements valid
)

// Key roles.
const (
	RoleAuthor    = "author"
	RoleVoucher   = "voucher"
	RolePublisher = "publisher"
	RoleInstaller = "installer"
	RoleOperator  = "operator"
)

// App install states driven by key lifecycle events.
const (
	AppStateActive             = "active"
	AppStatePendingRevet       = "pending-revet/no-new-activations"
	AppStateDisabled           = "disabled"
	AppStateQuarantinedRevoked = "quarantined-revoked" // pre-sandbox: same as disabled
)

// VoucherCeiling constrains what a voucher key may authorize.
type VoucherCeiling struct {
	MaxTier        string   // "quarantine" | "custom" | "full"
	AllowedTesting []string // e.g. ["sim-only"]
	MaxExpiryDays  int      // 0 → use system default (14)
}

// KeyRecord is one key known to the trust store.
type KeyRecord struct {
	KeyID           string
	Roles           []string
	State           string
	Ceiling         *VoucherCeiling // non-nil only when RoleVoucher is present
	FirstObservedAt int64           // unix seconds
	Note            string
	PubKey          ed25519.PublicKey
}

// HasRole reports whether the record holds the given role.
func (r *KeyRecord) HasRole(role string) bool {
	for _, rr := range r.Roles {
		if rr == role {
			return true
		}
	}
	return false
}

// LineageEdge is one author key in an app's lineage chain, oldest first.
type LineageEdge struct {
	AuthorKeyID string
	AddedAt     int64 // unix seconds
}

// AppLineage holds the ordered author key history for one app.
type AppLineage struct {
	AppID string
	Name  string        // manifest_name
	Edges []LineageEdge // oldest first
	State string        // one of the AppState* constants
}

// CurrentAuthorKeyID returns the most-recently-added author key for this lineage.
func (l *AppLineage) CurrentAuthorKeyID() string {
	if len(l.Edges) == 0 {
		return ""
	}
	return l.Edges[len(l.Edges)-1].AuthorKeyID
}

// RotationRecord captures an accepted rotation for later cutoff lookups.
type RotationRecord struct {
	OldKeyID    string
	NewKeyID    string
	NameScope   []string
	AcceptedSeq int64 // trust-log sequence at acceptance (the authoritative cutoff)
	AcceptedAt  int64 // unix seconds at acceptance
}

// LogEntry is one appended entry in the trust-mutation log.
type LogEntry struct {
	Seq            int64          `json:"seq"`
	At             int64          `json:"at"`
	Actor          string         `json:"actor"`
	Op             string         `json:"op"`
	Args           map[string]any `json:"args"`
	PrevHash       string         `json:"prev_hash"`
	ThisHash       string         `json:"this_hash"`
	InstallerSig   string         `json:"installer_sig"`
	InstallerKeyID string         `json:"installer_key_id"` // which installer key signed this entry
}

// OldKeyRotationStatement carries the outgoing author's signed rotation intent.
// Serialised as JSON for signature verification; the plan specifies CBOR for
// wire transport but the service layer operates on the parsed struct.
type OldKeyRotationStatement struct {
	Schema     string   `json:"schema"` // must be "rotation-stmt/1"
	OldKey     string   `json:"old_key"`
	NewKey     string   `json:"new_key"`
	ProposedAt int64    `json:"proposed_at"` // advisory only
	NameScope  []string `json:"name_scope"`
	Reason     string   `json:"reason,omitempty"`
	SigOld     []byte   `json:"-"` // not included in the signed payload
}

// NewKeyRotationStatement carries the incoming author's countersignature.
type NewKeyRotationStatement struct {
	Schema           string `json:"schema"`              // must be "rotation-stmt/1"
	OldKeyStmtDigest string `json:"old_key_stmt_digest"` // sha256 of serialised OldKeyRotationStatement payload
	NewKey           string `json:"new_key"`
	AcceptAt         int64  `json:"accept_at"` // advisory only
	SigNew           []byte `json:"-"`         // not included in the signed payload
}

// ComputeAppID derives the stable app lineage identifier per the plan §1.4.
//
//	app_id_bytes = sha256(first_author_key_id || "\0" || manifest_name)
//	app_id       = "app:sha256:" + hex(app_id_bytes)
func ComputeAppID(firstAuthorKeyID, manifestName string) string {
	h := sha256.Sum256([]byte(firstAuthorKeyID + "\x00" + manifestName))
	return "app:sha256:" + hex.EncodeToString(h[:])
}

// computeLogHash computes this_hash for a log entry over the canonical fields.
func computeLogHash(seq, at int64, actor, op string, args map[string]any, prevHash string) string {
	type canonical struct {
		Seq      int64          `json:"seq"`
		At       int64          `json:"at"`
		Actor    string         `json:"actor"`
		Op       string         `json:"op"`
		Args     map[string]any `json:"args"`
		PrevHash string         `json:"prev_hash"`
	}
	b, _ := json.Marshal(canonical{seq, at, actor, op, args, prevHash})
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}

// parseKeyID parses "ed25519:<base64url>" and returns the raw public key bytes.
func parseKeyID(keyID string) (ed25519.PublicKey, error) {
	const prefix = "ed25519:"
	if !strings.HasPrefix(keyID, prefix) {
		return nil, &KeyIDError{keyID, "unsupported prefix (only ed25519: is accepted)"}
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(keyID, prefix))
	if err != nil {
		return nil, &KeyIDError{keyID, "invalid base64url: " + err.Error()}
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("trust: key_id %q has wrong length %d (want %d)", keyID, len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}

// KeyIDError is returned when a key_id string is malformed.
type KeyIDError struct {
	KeyID  string
	Reason string
}

func (e *KeyIDError) Error() string {
	return "trust: invalid key_id " + e.KeyID + ": " + e.Reason
}
