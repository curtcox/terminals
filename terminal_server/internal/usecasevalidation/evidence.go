package usecasevalidation

import (
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Evidence writes the evidence bundle for this run and returns a summary.
// The bundle is always written under artifacts/usecase-validation/<run-id>/.
// The full bundle (including assertions.jsonl) is written when any assertion
// failed or USECASE_ARTIFACTS=1 is set. Otherwise only manifest.json is written.
func (h *Harness) Evidence(usecaseID string) *EvidenceBundle {
	h.t.Helper()
	// Collect TTS captures from the FakeTTS before taking the mutex snapshot so
	// that audio recorded during the run is included even if CaptureAudio was
	// not called explicitly.
	if h.tts != nil {
		for i, cap := range h.tts.Captures() {
			label := fmt.Sprintf("tts-%d", i)
			h.mu.Lock()
			h.audioClips = append(h.audioClips, AudioRecord{
				Label:     fmt.Sprintf("%s: %s", label, cap.Text),
				Path:      filepath.ToSlash(filepath.Join("audio", safeArtifactName(label)+".wav")),
				PCM:       cap.PCM,
				Timestamp: cap.Timestamp,
			})
			h.mu.Unlock()
		}
		// Reset so repeated Evidence() calls don't double-count.
		h.tts.mu.Lock()
		h.tts.captures = nil
		h.tts.mu.Unlock()
	}

	h.mu.Lock()
	assertions := make([]AssertionRecord, len(h.assertions))
	copy(assertions, h.assertions)
	interactions := make([]InteractionRecord, len(h.interactions))
	copy(interactions, h.interactions)
	frames := make([]FrameRecord, len(h.frames))
	copy(frames, h.frames)
	audioClips := make([]AudioRecord, len(h.audioClips))
	copy(audioClips, h.audioClips)
	h.mu.Unlock()

	end := time.Now().UTC()
	pass := true
	var failingIDs []string
	for _, a := range assertions {
		if !a.Pass {
			pass = false
			failingIDs = append(failingIDs, a.ID)
		}
	}

	// Strip PCM bytes from audio records before serialising into the manifest.
	audioMeta := make([]AudioRecord, len(audioClips))
	for i, a := range audioClips {
		audioMeta[i] = AudioRecord{Label: a.Label, Path: a.Path, Timestamp: a.Timestamp}
	}

	media := MediaManifest{
		Frames: frames,
		Audio:  audioMeta,
	}
	bundle := &EvidenceBundle{
		Manifest: Manifest{
			RunID:             h.runID,
			UseCaseID:         usecaseID,
			ScenarioName:      h.t.Name(),
			GitCommit:         gitCommit(),
			TimestampStart:    h.start,
			TimestampEnd:      end,
			Pass:              pass,
			FailingAssertions: failingIDs,
			InteractionTrace:  interactions,
			Media:             media,
		},
		Assertions:   assertions,
		Interactions: interactions,
		Frames:       frames,
		Audio:        audioClips,
	}

	writeArtifacts := os.Getenv("USECASE_ARTIFACTS") == "1" || !pass
	dir := filepath.Join(artifactsRoot(), "usecase-validation", h.runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		h.t.Logf("usecasevalidation: could not create artifacts dir %s: %v", dir, err)
		return bundle
	}

	resultDir := filepath.Join(artifactsRoot(), "usecases", usecaseID)
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		h.t.Logf("usecasevalidation: could not create result dir %s: %v", resultDir, err)
	} else {
		if len(frames) > 0 {
			if err := writeFramePNGs(resultDir, frames); err != nil {
				h.t.Logf("usecasevalidation: could not write frame artifacts: %v", err)
			} else if video, err := writeFrameVideo(resultDir, frames); err != nil {
				h.t.Logf("usecasevalidation: could not write video artifact: %v", err)
			} else if video != nil {
				bundle.Manifest.Media.Videos = append(bundle.Manifest.Media.Videos, *video)
				bundle.Videos = append(bundle.Videos, *video)
			}
		}
		if len(audioClips) > 0 {
			if err := writeAudioWAVs(resultDir, audioClips); err != nil {
				h.t.Logf("usecasevalidation: could not write audio artifacts: %v", err)
			}
		}
	}
	if len(frames) > 0 {
		if err := writeFramePNGs(dir, frames); err != nil {
			h.t.Logf("usecasevalidation: could not write evidence frame artifacts: %v", err)
		} else if _, err := writeFrameVideo(dir, frames); err != nil {
			h.t.Logf("usecasevalidation: could not write evidence video artifact: %v", err)
		}
	}
	if len(audioClips) > 0 {
		if err := writeAudioWAVs(dir, audioClips); err != nil {
			h.t.Logf("usecasevalidation: could not write evidence audio artifacts: %v", err)
		}
	}
	if err := writeJSON(filepath.Join(dir, "manifest.json"), bundle.Manifest); err != nil {
		h.t.Logf("usecasevalidation: could not write manifest.json: %v", err)
	}
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		h.t.Logf("usecasevalidation: could not create result dir %s: %v", resultDir, err)
	} else if err := writeJSON(filepath.Join(resultDir, "result.json"), bundle.Manifest); err != nil {
		h.t.Logf("usecasevalidation: could not write result.json: %v", err)
	}

	if writeArtifacts {
		if err := writeJSONL(filepath.Join(dir, "assertions.jsonl"), assertionsToAny(assertions)); err != nil {
			h.t.Logf("usecasevalidation: could not write assertions.jsonl: %v", err)
		}
		if err := writeJSONL(filepath.Join(dir, "interaction_trace.jsonl"), interactionsToAny(interactions)); err != nil {
			h.t.Logf("usecasevalidation: could not write interaction_trace.jsonl: %v", err)
		}
		if err := writeSummaryMD(filepath.Join(dir, "summary.md"), bundle); err != nil {
			h.t.Logf("usecasevalidation: could not write summary.md: %v", err)
		}
		h.t.Logf("usecasevalidation: full evidence bundle at %s", dir)
	} else {
		h.t.Logf("usecasevalidation: manifest at %s/manifest.json (set USECASE_ARTIFACTS=1 for full bundle)", dir)
	}

	return bundle
}

func writeSummaryMD(path string, b *EvidenceBundle) error {
	m := b.Manifest
	result := "PASS"
	if !m.Pass {
		result = "FAIL"
	}

	passed := 0
	failed := 0
	for _, a := range b.Assertions {
		if a.Pass {
			passed++
		} else {
			failed++
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Use Case %s — Validation Summary\n\n", m.UseCaseID)
	fmt.Fprintf(&sb, "**Run ID:** %s  \n", m.RunID)
	fmt.Fprintf(&sb, "**Scenario:** %s  \n", m.ScenarioName)
	fmt.Fprintf(&sb, "**Result:** %s  \n", result)
	fmt.Fprintf(&sb, "**Start:** %s  \n", m.TimestampStart.Format(time.RFC3339))
	fmt.Fprintf(&sb, "**End:** %s  \n", m.TimestampEnd.Format(time.RFC3339))
	if m.GitCommit != "" {
		fmt.Fprintf(&sb, "**Git commit:** %s  \n", m.GitCommit)
	}
	fmt.Fprintf(&sb, "\n## Assertions (%d passed, %d failed)\n\n", passed, failed)
	fmt.Fprintf(&sb, "| ID | Description | Result | Detail |\n")
	fmt.Fprintf(&sb, "|---|---|---|---|\n")
	for _, a := range b.Assertions {
		mark := "✓ PASS"
		if !a.Pass {
			mark = "✗ FAIL"
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n", a.ID, a.Description, mark, a.Detail)
	}
	if len(m.FailingAssertions) > 0 {
		fmt.Fprintf(&sb, "\n**Failing assertions:** %s\n", strings.Join(m.FailingAssertions, ", "))
	}
	if len(b.Interactions) > 0 {
		fmt.Fprintf(&sb, "\n## Interaction trace\n\n")
		for i, interaction := range b.Interactions {
			fmt.Fprintf(&sb, "%d. %s\n", i+1, interaction.Summary)
		}
	}
	if len(b.Frames) > 0 {
		fmt.Fprintf(&sb, "\n## Visual frames\n\n")
		for _, frame := range b.Frames {
			fmt.Fprintf(&sb, "- [%s](%s): %s\n", frame.Label, frame.Path, frame.Summary)
		}
	}
	if len(b.Videos) > 0 {
		fmt.Fprintf(&sb, "\n## Video\n\n")
		for _, video := range b.Videos {
			fmt.Fprintf(&sb, "- [%s](%s)\n", video.Label, video.Path)
		}
	}
	fmt.Fprintf(&sb, "\n## Replay\n\n```bash\ngo test ./internal/usecasevalidation -run TestReplay -args -bundle %s\n```\n", filepath.Dir(path))
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

func writeFramePNGs(baseDir string, frames []FrameRecord) error {
	for _, frame := range frames {
		path := filepath.Join(baseDir, filepath.FromSlash(frame.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		err = png.Encode(file, renderFrameImage(frame))
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func writeFrameVideo(baseDir string, frames []FrameRecord) (*VideoRecord, error) {
	if len(frames) == 0 {
		return nil, nil
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, nil
	}
	video := VideoRecord{
		Label: "Validation walkthrough",
		Path:  filepath.ToSlash(filepath.Join("video", "validation.mp4")),
	}
	videoPath := filepath.Join(baseDir, filepath.FromSlash(video.Path))
	if err := os.MkdirAll(filepath.Dir(videoPath), 0o755); err != nil {
		return nil, err
	}
	listPath := filepath.Join(filepath.Dir(videoPath), "frames.txt")
	var sb strings.Builder
	for _, frame := range frames {
		framePath, err := filepath.Abs(filepath.Join(baseDir, filepath.FromSlash(frame.Path)))
		if err != nil {
			return nil, err
		}
		escaped := strings.ReplaceAll(framePath, "'", "'\\''")
		fmt.Fprintf(&sb, "file '%s'\n", escaped)
		fmt.Fprintf(&sb, "duration %.2f\n", 1.25)
	}
	lastFramePath, err := filepath.Abs(filepath.Join(baseDir, filepath.FromSlash(frames[len(frames)-1].Path)))
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(&sb, "file '%s'\n", strings.ReplaceAll(lastFramePath, "'", "'\\''"))
	if err := os.WriteFile(listPath, []byte(sb.String()), 0o644); err != nil {
		return nil, err
	}
	cmd := exec.Command(
		ffmpeg,
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", listPath,
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		videoPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return &video, nil
}

// writeAudioWAVs encodes each AudioRecord's raw PCM16 data as a WAV file.
// Records without PCM bytes are skipped (metadata-only records from deserialized manifests).
func writeAudioWAVs(baseDir string, clips []AudioRecord) error {
	for _, clip := range clips {
		if len(clip.PCM) == 0 {
			continue
		}
		path := filepath.Join(baseDir, filepath.FromSlash(clip.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		wavData := encodeWAV(clip.PCM, 24000, 1)
		if err := os.WriteFile(path, wavData, 0o644); err != nil {
			return err
		}
	}
	return nil
}
