package trust

import (
	"errors"
	"fmt"

	"github.com/BurntSushi/toml"
)

// PolicyV1 is the minimum v1 policy schema consumed by the distribution
// vetting pipeline (plan §8). Unknown top-level tables are rejected.
type PolicyV1 struct {
	PolicySchema    string          `toml:"policy_schema"`
	Name            string          `toml:"name"`
	Gate3           Gate3Policy     `toml:"gate3"`
	Gate5           Gate5Policy     `toml:"gate5"`
	Gate7           Gate7Policy     `toml:"gate7"`
	Revoke          RevokePolicies  `toml:"revoke"`
	VoucherDefaults VoucherDefaults `toml:"voucher_defaults"`
}

// Gate3Policy governs author / voucher admission.
type Gate3Policy struct {
	Rules []Gate3Rule `toml:"rule"`
}

// Gate3Rule is one admission rule for Gate 3.
type Gate3Rule struct {
	Kind       string   `toml:"kind"`        // "trusted_author" | "voucher_quorum" | "quarantine_admit"
	NameFilter string   `toml:"name_filter"` // glob over manifest_name
	MinCount   int      `toml:"min_count"`   // for voucher_quorum
	MinTier    string   `toml:"min_tier"`    // for voucher_quorum
	TestingIn  []string `toml:"testing_in"`  // for voucher_quorum
	UniqueKeys bool     `toml:"unique_keys"` // for voucher_quorum
	Enabled    *bool    `toml:"enabled"`     // optional; default true
}

// Gate5Policy governs AI reviewers.
type Gate5Policy struct {
	MinActiveProviders int          `toml:"min_active_providers"`
	CooldownTreatment  string       `toml:"cooldown_treatment"`
	Providers          []AIProvider `toml:"provider"`
}

// AIProvider is one AI reviewer provider.
type AIProvider struct {
	ID           string `toml:"id"`
	Model        string `toml:"model"`
	ContextScope string `toml:"context_scope"`
	SubstituteID string `toml:"substitute_id"`
}

// Gate7Policy governs risk thresholds.
type Gate7Policy struct {
	BlockAboveScore int          `toml:"block_above_score"`
	WarnAboveScore  int          `toml:"warn_above_score"`
	Weights         []PermWeight `toml:"weight"`
}

// PermWeight assigns a risk weight to one permission.
type PermWeight struct {
	Permission string `toml:"permission"`
	Weight     int    `toml:"weight"`
}

// RevokePolicies describes default app consequences when keys are revoked.
type RevokePolicies struct {
	AuthorDefault   string `toml:"author_default"`
	VoucherDefault  string `toml:"voucher_default"`
	CompromiseFloor string `toml:"compromise_floor"`
}

// VoucherDefaults are used when a voucher key has no ceiling entry.
type VoucherDefaults struct {
	MaxTier        string   `toml:"max_tier"`
	AllowedTesting []string `toml:"allowed_testing"`
	MaxExpiryDays  int      `toml:"max_expiry_days"`
}

// LoadPolicy loads and validates a policy/1 file from path.
// Unknown top-level keys are rejected (per plan §8: "unknown top-level tables
// are rejected at load (not ignored)").
func LoadPolicy(path string) (*PolicyV1, error) {
	var raw map[string]any
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return nil, fmt.Errorf("trust: policy load: %w", err)
	}
	// Validate schema field first.
	schema, _ := raw["policy_schema"].(string)
	if schema == "" {
		return nil, errors.New("trust: policy missing required field policy_schema")
	}
	if schema != "policy/1" {
		return nil, fmt.Errorf("trust: policy schema %q is not supported (only policy/1)", schema)
	}
	// Reject unknown top-level tables.
	allowed := map[string]bool{
		"policy_schema":    true,
		"name":             true,
		"gate3":            true,
		"gate5":            true,
		"gate7":            true,
		"revoke":           true,
		"voucher_defaults": true,
	}
	for k := range raw {
		if !allowed[k] {
			return nil, fmt.Errorf("trust: policy contains unknown top-level key %q (not allowed by policy/1 schema)", k)
		}
	}
	// Full decode into the typed struct.
	var p PolicyV1
	if _, err := toml.DecodeFile(path, &p); err != nil {
		return nil, fmt.Errorf("trust: policy decode: %w", err)
	}
	return &p, nil
}
