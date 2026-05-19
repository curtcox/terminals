package trust

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

func (s *Service) verifyLogEntry(i int, e *LogEntry) error {
	want := computeLogHash(e.Seq, e.At, e.Actor, e.Op, e.Args, e.PrevHash)
	if e.ThisHash != want {
		return fmt.Errorf("trust: log entry %d hash mismatch: got %s want %s", i+1, e.ThisHash, want)
	}
	if i > 0 && e.PrevHash != s.log[i-1].ThisHash {
		return fmt.Errorf("trust: log chain broken at entry %d: prev_hash mismatch", i+1)
	}
	sigBytes, err := base64.StdEncoding.DecodeString(e.InstallerSig)
	if err != nil {
		return fmt.Errorf("trust: log entry %d installer_sig is not valid base64: %w", i+1, err)
	}
	verifyPub, err := s.installerPubForLogEntry(e, i)
	if err != nil {
		return err
	}
	if !ed25519.Verify(verifyPub, []byte(e.ThisHash), sigBytes) {
		return fmt.Errorf("trust: log entry %d installer signature verification failed", i+1)
	}
	return nil
}

func (s *Service) installerPubForLogEntry(e *LogEntry, index int) (ed25519.PublicKey, error) {
	if e.InstallerKeyID == "" {
		return s.installerPub, nil
	}
	rec, ok := s.keys[e.InstallerKeyID]
	if ok {
		return rec.PubKey, nil
	}
	parsed, parseErr := parseKeyID(e.InstallerKeyID)
	if parseErr != nil {
		return nil, fmt.Errorf("trust: log entry %d references unknown installer key %s", index+1, e.InstallerKeyID)
	}
	return parsed, nil
}
