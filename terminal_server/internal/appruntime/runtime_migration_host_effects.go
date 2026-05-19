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
		effect, found, err := migrationArtifactPatchFromLine(pkg, line, lineNumber+1, patchAliases, patchCount)
		if err != nil {
			return effects, err
		}
		if !found {
			continue
		}
		patchCount++
		effects.ArtifactPatches = append(effects.ArtifactPatches, effect)
	}
	return effects, nil
}

func migrationArtifactPatchFromLine(
	pkg Package,
	line string,
	lineNumber int,
	patchAliases map[string]struct{},
	patchCount int,
) (runtimeMigrationArtifactPatchEffect, bool, error) {
	line = strings.TrimSpace(stripTALLineComment(line))
	if line == "" || strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "load(") || strings.HasPrefix(line, "return ") || line == "pass" {
		return runtimeMigrationArtifactPatchEffect{}, false, nil
	}
	match := migrateCallPattern.FindStringSubmatch(line)
	if match == nil {
		return runtimeMigrationArtifactPatchEffect{}, false, nil
	}
	if _, ok := patchAliases[match[1]]; !ok {
		return runtimeMigrationArtifactPatchEffect{}, false, nil
	}
	nextCount := patchCount + 1
	if nextCount > migrationMaxArtifactPatches {
		return runtimeMigrationArtifactPatchEffect{}, false, fmt.Errorf("%w: artifact.self.patch count exceeds hard cap (%d > %d)", ErrMigrationResourceLimit, nextCount, migrationMaxArtifactPatches)
	}
	appID := strings.TrimSpace(pkg.Manifest.AppID)
	if appID == "" {
		return runtimeMigrationArtifactPatchEffect{}, false, fmt.Errorf("%w: artifact.self.patch requires manifest app_id at line %d", ErrMigrationArtifactOwnership, lineNumber)
	}
	artifactID := migrationStringArgument(match[2])
	if artifactID == "" {
		return runtimeMigrationArtifactPatchEffect{}, false, fmt.Errorf("%w: artifact.self.patch missing artifact_id at line %d", ErrMigrationArtifactOwnership, lineNumber)
	}
	ownerAppID := migrationKeywordStringArgument(migrateOwnerAppIDPattern, match[2])
	if ownerAppID == "" {
		return runtimeMigrationArtifactPatchEffect{}, false, fmt.Errorf("%w: artifact %q patch missing owner_app_id at line %d", ErrMigrationArtifactOwnership, artifactID, lineNumber)
	}
	if ownerAppID != appID {
		return runtimeMigrationArtifactPatchEffect{}, false, fmt.Errorf("%w: artifact %q owner_app_id %q does not match app_id %q at line %d", ErrMigrationArtifactOwnership, artifactID, ownerAppID, appID, lineNumber)
	}
	return runtimeMigrationArtifactPatchEffect{
		ArtifactID: artifactID,
		OwnerAppID: ownerAppID,
		Sequence:   nextCount,
	}, true, nil
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
