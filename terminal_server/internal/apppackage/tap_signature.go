package apppackage

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/fxamacker/cbor/v2"
)

func verifySignatureBundle(sigBytes []byte, expectedPackageID string, packageHash []byte, expectedManifestName string, expectedManifestVersion string) ([]VerifiedStatement, error) {
	if len(sigBytes) == 0 || len(sigBytes) > signatureBundleMaxBytes {
		return nil, ErrInvalidSignatureBundle
	}

	var bundle signatureBundle
	if _, err := toml.Decode(string(sigBytes), &bundle); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSignatureBundle, err)
	}

	if bundle.Schema != signatureBundleSchema {
		return nil, ErrInvalidSignatureBundle
	}
	if err := requireMaxStringLen(bundle.PackageID); err != nil {
		return nil, err
	}
	if bundle.PackageID != expectedPackageID {
		return nil, ErrSignaturePackageIDMismatch
	}
	if len(bundle.Statement) == 0 || len(bundle.Statement) > statementMaxCount {
		return nil, ErrInvalidSignatureBundle
	}

	seen := make(map[string]struct{}, len(bundle.Statement))
	authorSeen := false
	verified := make([]VerifiedStatement, 0, len(bundle.Statement))

	for _, stmt := range bundle.Statement {
		statement, err := verifySignatureStatement(stmt, expectedPackageID, packageHash, expectedManifestName, expectedManifestVersion, seen)
		if err != nil {
			return nil, err
		}
		if stmt.Role == "author" {
			authorSeen = true
		}
		verified = append(verified, statement)
	}

	if !authorSeen {
		return nil, ErrMissingAuthorSignature
	}

	return verified, nil
}

func verifySignatureStatement(
	stmt signatureStatement,
	expectedPackageID string,
	packageHash []byte,
	expectedManifestName string,
	expectedManifestVersion string,
	seen map[string]struct{},
) (VerifiedStatement, error) {
	if err := validateStatementFields(stmt); err != nil {
		return VerifiedStatement{}, err
	}
	nonceRaw, sigRaw, publicKey, scope, err := parseSignatureStatementParts(stmt)
	if err != nil {
		return VerifiedStatement{}, err
	}
	payload, err := encodeStatementCBOR(stmt, scope, packageHash, nonceRaw)
	if err != nil {
		return VerifiedStatement{}, err
	}
	if !ed25519.Verify(publicKey, payload, sigRaw) {
		return VerifiedStatement{}, ErrSignatureVerificationFailed
	}
	if stmt.ManifestName != expectedManifestName || stmt.ManifestVersion != expectedManifestVersion {
		return VerifiedStatement{}, ErrInvalidSignatureStatement
	}
	if err := rememberSignatureStatement(stmt, expectedPackageID, nonceRaw, seen); err != nil {
		return VerifiedStatement{}, err
	}
	return VerifiedStatement{
		Role:            stmt.Role,
		KeyID:           stmt.KeyID,
		CreatedUnix:     stmt.CreatedUnix,
		ManifestName:    stmt.ManifestName,
		ManifestVersion: stmt.ManifestVersion,
		Nonce:           stmt.Nonce,
		Scope:           scope,
	}, nil
}

func parseSignatureStatementParts(stmt signatureStatement) ([]byte, []byte, ed25519.PublicKey, map[string]any, error) {
	nonceRaw, err := parsePrefixedBase64URL(stmt.Nonce, "base64url:", statementNonceLen)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	sigRaw, err := parsePrefixedBase64(stmt.Sig, "base64:")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	publicKey, err := parseKeyID(stmt.KeyID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	scope, err := normalizeScope(stmt.Role, stmt.Scope)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return nonceRaw, sigRaw, publicKey, scope, nil
}

func rememberSignatureStatement(stmt signatureStatement, expectedPackageID string, nonceRaw []byte, seen map[string]struct{}) error {
	nonceKey := base64.RawURLEncoding.EncodeToString(nonceRaw)
	triple := stmt.KeyID + "\x00" + expectedPackageID + "\x00" + nonceKey
	if _, ok := seen[triple]; ok {
		return ErrInvalidSignatureStatement
	}
	seen[triple] = struct{}{}
	return nil
}

func validateStatementFields(stmt signatureStatement) error {
	if stmt.Role == "" || stmt.KeyID == "" || stmt.ManifestName == "" || stmt.ManifestVersion == "" || stmt.Nonce == "" || stmt.Sig == "" {
		return ErrInvalidSignatureStatement
	}
	if err := requireMaxStringLen(stmt.Role, stmt.KeyID, stmt.ManifestName, stmt.ManifestVersion, stmt.Nonce, stmt.Sig); err != nil {
		return err
	}
	return nil
}

func requireMaxStringLen(values ...string) error {
	for _, value := range values {
		if len(value) > stringFieldMaxBytes {
			return ErrInvalidSignatureStatement
		}
	}
	return nil
}

func parsePrefixedBase64URL(raw string, prefix string, expectedLen int) ([]byte, error) {
	if !strings.HasPrefix(raw, prefix) {
		return nil, ErrInvalidSignatureStatement
	}
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(raw, prefix))
	if err != nil {
		return nil, ErrInvalidSignatureStatement
	}
	if len(decoded) != expectedLen {
		return nil, ErrInvalidSignatureStatement
	}
	return decoded, nil
}

func parsePrefixedBase64(raw string, prefix string) ([]byte, error) {
	if !strings.HasPrefix(raw, prefix) {
		return nil, ErrInvalidSignatureStatement
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(raw, prefix))
	if err != nil {
		return nil, ErrInvalidSignatureStatement
	}
	if len(decoded) == 0 {
		return nil, ErrInvalidSignatureStatement
	}
	return decoded, nil
}

func parseKeyID(keyID string) (ed25519.PublicKey, error) {
	const prefix = "ed25519:"
	if !strings.HasPrefix(keyID, prefix) {
		return nil, ErrInvalidSignatureStatement
	}
	encoded := strings.TrimPrefix(keyID, prefix)
	publicKey, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		publicKey, err = base64.StdEncoding.DecodeString(encoded)
	}
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return nil, ErrInvalidSignatureStatement
	}
	return ed25519.PublicKey(publicKey), nil
}

func normalizeScope(role string, input map[string]any) (map[string]any, error) {
	if input == nil {
		input = map[string]any{}
	}

	switch role {
	case "author":
		return normalizeAuthorScope(input)
	case "voucher":
		return normalizeVoucherScope(input)
	case "publisher":
		return normalizePublisherScope(input)
	default:
		return nil, ErrInvalidSignatureStatement
	}
}

func normalizeAuthorScope(input map[string]any) (map[string]any, error) {
	if len(input) != 0 {
		return nil, ErrInvalidSignatureStatement
	}
	return map[string]any{}, nil
}

func normalizeVoucherScope(input map[string]any) (map[string]any, error) {
	allowed := map[string]struct{}{"tier": {}, "reviewed": {}, "tested_under": {}, "notes": {}, "expires_unix": {}}
	if err := requireOnlyScopeKeys(input, allowed); err != nil {
		return nil, err
	}
	tier, err := normalizeVoucherTier(input["tier"])
	if err != nil {
		return nil, err
	}
	reviewed, err := normalizeVoucherReviewed(input["reviewed"])
	if err != nil {
		return nil, err
	}
	testedUnder, err := normalizeVoucherTestedUnder(input["tested_under"])
	if err != nil {
		return nil, err
	}
	normalized := map[string]any{
		"tier":         tier,
		"reviewed":     reviewed,
		"tested_under": testedUnder,
	}
	if err := addOptionalVoucherScope(normalized, input); err != nil {
		return nil, err
	}
	return normalized, nil
}

func normalizePublisherScope(input map[string]any) (map[string]any, error) {
	allowed := map[string]struct{}{"via": {}}
	if err := requireOnlyScopeKeys(input, allowed); err != nil {
		return nil, err
	}
	via, ok := input["via"].(string)
	if !ok || via == "" || len(via) > publisherViaMaxBytes {
		return nil, ErrInvalidSignatureStatement
	}
	return map[string]any{"via": via}, nil
}

func requireOnlyScopeKeys(input map[string]any, allowed map[string]struct{}) error {
	for key := range input {
		if _, ok := allowed[key]; !ok {
			return ErrInvalidSignatureStatement
		}
	}
	return nil
}

func normalizeVoucherTier(value any) (string, error) {
	tier, ok := value.(string)
	if !ok || (tier != "full" && tier != "quarantine" && tier != "custom") {
		return "", ErrInvalidSignatureStatement
	}
	return tier, nil
}

func normalizeVoucherReviewed(value any) ([]string, error) {
	reviewedRaw, ok := value.([]any)
	if !ok {
		return nil, ErrInvalidSignatureStatement
	}
	reviewed := make([]string, 0, len(reviewedRaw))
	for _, value := range reviewedRaw {
		v, ok := value.(string)
		if !ok {
			return nil, ErrInvalidSignatureStatement
		}
		switch v {
		case "manifest", "tal", "tests", "kernels", "models", "assets":
			reviewed = append(reviewed, v)
		default:
			return nil, ErrInvalidSignatureStatement
		}
	}
	return reviewed, nil
}

func normalizeVoucherTestedUnder(value any) (string, error) {
	testedUnder, ok := value.(string)
	if !ok || (testedUnder != "sim-only" && testedUnder != "hardware" && testedUnder != "production") {
		return "", ErrInvalidSignatureStatement
	}
	return testedUnder, nil
}

func addOptionalVoucherScope(normalized map[string]any, input map[string]any) error {
	if notesRaw, ok := input["notes"]; ok {
		notes, ok := notesRaw.(string)
		if !ok || len(notes) > voucherNotesMaxBytes {
			return ErrInvalidSignatureStatement
		}
		normalized["notes"] = notes
	}
	if expiresRaw, ok := input["expires_unix"]; ok {
		expiresUnix, ok := asUint64(expiresRaw)
		if !ok {
			return ErrInvalidSignatureStatement
		}
		normalized["expires_unix"] = expiresUnix
	}
	return nil
}

func asUint64(value any) (uint64, bool) {
	switch v := value.(type) {
	case int64:
		if v < 0 {
			return 0, false
		}
		return uint64(v), true
	case int:
		if v < 0 {
			return 0, false
		}
		return uint64(v), true
	case uint64:
		return v, true
	case float64:
		if v < 0 || float64(uint64(v)) != v {
			return 0, false
		}
		return uint64(v), true
	default:
		return 0, false
	}
}

func encodeStatementCBOR(stmt signatureStatement, normalizedScope map[string]any, packageHash []byte, nonceRaw []byte) ([]byte, error) {
	encMode, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSignatureStatement, err)
	}

	payload := map[uint64]any{
		0: uint64(1),
		1: append([]byte(nil), packageHash...),
		2: stmt.Role,
		3: stmt.KeyID,
		4: stmt.CreatedUnix,
		5: stmt.ManifestName,
		6: stmt.ManifestVersion,
		7: normalizedScope,
		8: append([]byte(nil), nonceRaw...),
	}

	b, err := encMode.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSignatureStatement, err)
	}
	return b, nil
}
