package appruntime

import (
	"fmt"
	"strings"
)

type runtimeMigrationHostEffects struct {
	ArtifactPatches []runtimeMigrationArtifactPatchEffect
}

type runtimeMigrationArtifactPatchEffect struct {
	ArtifactID string
	OwnerAppID string
	Sequence   int
}

func collectRuntimeMigrationHostEffects(pkg Package, scriptSource []byte) (runtimeMigrationHostEffects, error) {
	var effects runtimeMigrationHostEffects
	patchAliases := artifactSelfPatchAliases(scriptSource)
	if len(patchAliases) == 0 {
		return effects, nil
	}
	patchCount := 0
	lines := strings.Split(string(scriptSource), "\n")
	for lineNumber, line := range lines {
		line = strings.TrimSpace(stripTALLineComment(line))
		if line == "" || strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "load(") || strings.HasPrefix(line, "return ") || line == "pass" {
			continue
		}
		match := migrateCallPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		if _, ok := patchAliases[match[1]]; !ok {
			continue
		}
		patchCount++
		if patchCount > migrationMaxArtifactPatches {
			return effects, fmt.Errorf("%w: artifact.self.patch count exceeds hard cap (%d > %d)", ErrMigrationResourceLimit, patchCount, migrationMaxArtifactPatches)
		}
		appID := strings.TrimSpace(pkg.Manifest.AppID)
		if appID == "" {
			return effects, fmt.Errorf("%w: artifact.self.patch requires manifest app_id at line %d", ErrMigrationArtifactOwnership, lineNumber+1)
		}
		artifactID := migrationStringArgument(match[2])
		if artifactID == "" {
			return effects, fmt.Errorf("%w: artifact.self.patch missing artifact_id at line %d", ErrMigrationArtifactOwnership, lineNumber+1)
		}
		ownerAppID := migrationKeywordStringArgument(migrateOwnerAppIDPattern, match[2])
		if ownerAppID == "" {
			return effects, fmt.Errorf("%w: artifact %q patch missing owner_app_id at line %d", ErrMigrationArtifactOwnership, artifactID, lineNumber+1)
		}
		if ownerAppID != appID {
			return effects, fmt.Errorf("%w: artifact %q owner_app_id %q does not match app_id %q at line %d", ErrMigrationArtifactOwnership, artifactID, ownerAppID, appID, lineNumber+1)
		}
		effects.ArtifactPatches = append(effects.ArtifactPatches, runtimeMigrationArtifactPatchEffect{
			ArtifactID: artifactID,
			OwnerAppID: ownerAppID,
			Sequence:   patchCount,
		})
	}
	return effects, nil
}

func artifactSelfPatchAliases(scriptSource []byte) map[string]struct{} {
	aliases := make(map[string]struct{})
	for _, match := range migrateArtifactSelfLoadPattern.FindAllSubmatch(scriptSource, -1) {
		if len(match) < 2 {
			continue
		}
		for _, aliasMatch := range migrateLoadAliasPattern.FindAllSubmatch(match[1], -1) {
			if len(aliasMatch) < 2 {
				continue
			}
			aliases[string(aliasMatch[1])] = struct{}{}
		}
	}
	return aliases
}
