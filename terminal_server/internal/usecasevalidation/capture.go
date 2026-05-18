package usecasevalidation

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// CaptureFrame records a deterministic server-side visual snapshot for docs.
func (h *Harness) CaptureFrame(stepID, terminal string, messages []transport.ProtoServerEnvelope) {
	summary := summarizeLatestUI(messages)
	descriptor := extractLatestDescriptor(messages)
	h.mu.Lock()
	h.frames = append(h.frames, FrameRecord{
		StepID:     stepID,
		Terminal:   terminal,
		Label:      stepID,
		Summary:    summary,
		Descriptor: descriptor,
		Path:       filepath.ToSlash(filepath.Join("frames", safeArtifactName(stepID)+".png")),
		Timestamp:  time.Now().UTC(),
	})
	h.mu.Unlock()
}

// CaptureHostFrame records a frame from the current server-side UI state for
// a device by replaying MemoryHost events. Use this to capture UI state
// produced by server operations that bypass the terminal stream (e.g.
// ProcessDueTimers patches the banner via env.UI.Patch without sending a
// proto message to the terminal).
func (h *Harness) CaptureHostFrame(stepID, deviceID string) {
	if h.Runtime == nil || h.Runtime.Env == nil || h.Runtime.Env.UI == nil {
		return
	}
	type eventer interface {
		Events() []ui.HostEvent
	}
	host, ok := h.Runtime.Env.UI.(eventer)
	if !ok {
		return
	}
	current, ok := replayHostEvents(host.Events(), deviceID)
	if !ok {
		return
	}
	summary := "UI state: " + summarizeDescriptor(current)
	h.mu.Lock()
	h.frames = append(h.frames, FrameRecord{
		StepID:     stepID,
		Terminal:   deviceID,
		Label:      stepID,
		Summary:    summary,
		Descriptor: &current,
		Path:       filepath.ToSlash(filepath.Join("frames", safeArtifactName(stepID)+".png")),
		Timestamp:  time.Now().UTC(),
	})
	h.mu.Unlock()
}

// replayHostEvents replays UI host events for a device to produce its current
// UI state including all applied patches.
func replayHostEvents(events []ui.HostEvent, deviceID string) (ui.Descriptor, bool) {
	var root ui.Descriptor
	found := false
	for _, ev := range events {
		if ev.DeviceID != deviceID {
			continue
		}
		switch ev.Kind {
		case "set":
			root = ev.Node
			found = true
		case "patch":
			if found {
				root = applyDescriptorPatch(root, ev.ComponentID, ev.Node)
			}
		case "clear":
			root = ui.Descriptor{}
			found = false
		}
	}
	return root, found
}

// applyDescriptorPatch replaces the node whose Props["id"] matches componentID.
// The transport layer may scope plain IDs to "act:<deviceID>/<componentID>" when
// sending to terminals, which mutates the shared Props map stored in MemoryHost
// events. Both plain ("banner") and scoped ("act:kitchen/banner") forms are matched.
func applyDescriptorPatch(root ui.Descriptor, componentID string, node ui.Descriptor) ui.Descriptor {
	id := root.Props["id"]
	if id == componentID || strings.HasSuffix(id, "/"+componentID) {
		return node
	}
	if len(root.Children) == 0 {
		return root
	}
	patched := make([]ui.Descriptor, len(root.Children))
	for i, child := range root.Children {
		patched[i] = applyDescriptorPatch(child, componentID, node)
	}
	root.Children = patched
	return root
}

// CaptureAudio extracts any PlayAudio messages from messages and records them
// as audio artifacts for the doc site.
func (h *Harness) CaptureAudio(stepID, _ string, messages []transport.ProtoServerEnvelope) {
	var index int
	for _, env := range messages {
		resp, ok := env.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		pa := resp.GetPlayAudio()
		if pa == nil {
			continue
		}
		pcm, ok := pa.Source.(*iov1.PlayAudio_PcmData)
		if !ok || len(pcm.PcmData) == 0 {
			continue
		}
		label := fmt.Sprintf("%s-audio-%d", stepID, index)
		index++
		h.mu.Lock()
		h.audioClips = append(h.audioClips, AudioRecord{
			Label:      label,
			Path:       filepath.ToSlash(filepath.Join("audio", safeArtifactName(label)+".wav")),
			Source:     "play-audio-pcm",
			RightsNote: "Captured from a PlayAudio protobuf message emitted during validation.",
			PCM:        append([]byte(nil), pcm.PcmData...),
			Timestamp:  time.Now().UTC(),
		})
		h.mu.Unlock()
	}
}
