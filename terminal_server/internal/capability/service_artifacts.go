package capability

import (
	"strings"
)

// CreateArtifact creates a new artifact of the given kind and title.
func (s *Service) CreateArtifact(kind, title string) Artifact {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	item := Artifact{
		ID:        s.nextIDLocked("art"),
		Kind:      defaultIfBlank(kind, "document"),
		Title:     strings.TrimSpace(title),
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.artifacts = append(s.artifacts, item)
	s.appendArtifactVersionLocked(item, "create")
	s.appendRecentLocked("artifact", item.ID+" "+item.Title)
	return item
}

// PatchArtifact updates the title of the artifact with the given ID.
func (s *Service) PatchArtifact(artifactID, title string) (Artifact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	artifactID = strings.TrimSpace(artifactID)
	title = strings.TrimSpace(title)
	for i := range s.artifacts {
		if s.artifacts[i].ID != artifactID {
			continue
		}
		if title == "" {
			return s.artifacts[i], true
		}
		s.artifacts[i].Title = title
		s.artifacts[i].Version++
		s.artifacts[i].UpdatedAt = s.now()
		s.appendArtifactVersionLocked(s.artifacts[i], "patch")
		s.appendRecentLocked("artifact", s.artifacts[i].ID+" patch "+s.artifacts[i].Title)
		return s.artifacts[i], true
	}
	return Artifact{}, false
}

// ReplaceArtifact replaces the artifact title and records a full replacement version.
func (s *Service) ReplaceArtifact(artifactID, title string) (Artifact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	artifactID = strings.TrimSpace(artifactID)
	title = strings.TrimSpace(title)
	for i := range s.artifacts {
		if s.artifacts[i].ID != artifactID {
			continue
		}
		s.artifacts[i].Title = title
		s.artifacts[i].Version++
		s.artifacts[i].UpdatedAt = s.now()
		s.appendArtifactVersionLocked(s.artifacts[i], "replace")
		s.appendRecentLocked("artifact", s.artifacts[i].ID+" replace "+s.artifacts[i].Title)
		return s.artifacts[i], true
	}
	return Artifact{}, false
}

// SaveArtifactTemplate stores one template by name using an existing artifact as source.
func (s *Service) SaveArtifactTemplate(name, sourceArtifactID string) (ArtifactTemplate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	sourceArtifactID = strings.TrimSpace(sourceArtifactID)
	if name == "" || sourceArtifactID == "" {
		return ArtifactTemplate{}, false
	}
	artifact, ok := s.getArtifactLocked(sourceArtifactID)
	if !ok {
		return ArtifactTemplate{}, false
	}
	now := s.now()
	template := ArtifactTemplate{
		Name:             name,
		SourceArtifactID: artifact.ID,
		SourceKind:       artifact.Kind,
		SourceTitle:      artifact.Title,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if existing, ok := s.templates[name]; ok {
		template.CreatedAt = existing.CreatedAt
	}
	s.templates[name] = template
	s.appendRecentLocked("artifact", "template save "+template.Name+" -> "+template.SourceArtifactID)
	return template, true
}

// ApplyArtifactTemplate applies a saved template to an existing target artifact.
func (s *Service) ApplyArtifactTemplate(name, targetArtifactID string) (Artifact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	targetArtifactID = strings.TrimSpace(targetArtifactID)
	template, ok := s.templates[name]
	if !ok || targetArtifactID == "" {
		return Artifact{}, false
	}
	for i := range s.artifacts {
		if s.artifacts[i].ID != targetArtifactID {
			continue
		}
		s.artifacts[i].Kind = template.SourceKind
		s.artifacts[i].Title = template.SourceTitle
		s.artifacts[i].Version++
		s.artifacts[i].UpdatedAt = s.now()
		s.appendArtifactVersionLocked(s.artifacts[i], "template.apply:"+template.Name)
		s.appendRecentLocked("artifact", s.artifacts[i].ID+" template "+template.Name)
		return s.artifacts[i], true
	}
	return Artifact{}, false
}

// GetArtifact returns one artifact by ID.
func (s *Service) GetArtifact(artifactID string) (Artifact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	artifactID = strings.TrimSpace(artifactID)
	for _, item := range s.artifacts {
		if item.ID == artifactID {
			return item, true
		}
	}
	return Artifact{}, false
}

// ListArtifacts returns all stored artifacts.
func (s *Service) ListArtifacts() []Artifact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Artifact(nil), s.artifacts...)
}

func (s *Service) getArtifactLocked(artifactID string) (Artifact, bool) {
	for _, item := range s.artifacts {
		if item.ID == artifactID {
			return item, true
		}
	}
	return Artifact{}, false
}

// ArtifactHistory returns version history for an artifact in creation order.
func (s *Service) ArtifactHistory(artifactID string) ([]ArtifactVersion, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	artifactID = strings.TrimSpace(artifactID)
	versions, ok := s.versions[artifactID]
	if !ok {
		return nil, false
	}
	return append([]ArtifactVersion(nil), versions...), true
}
func (s *Service) appendArtifactVersionLocked(item Artifact, action string) {
	s.versions[item.ID] = append(s.versions[item.ID], ArtifactVersion{
		ArtifactID: item.ID,
		Version:    item.Version,
		Kind:       item.Kind,
		Title:      item.Title,
		Action:     strings.TrimSpace(action),
		CreatedAt:  item.UpdatedAt,
	})
}
