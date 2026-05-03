package transport

import (
	"strings"
	"sync"

	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// UISessionState owns the per-device UI session bookkeeping that the
// reconnect/heartbeat replay paths and command handlers consult: the most
// recent SetUI sent to a device, the count of host UI events already
// delivered, the active main-UI activation ID, and any pending
// multi-window resume snapshot. It owns its own mutex; StreamHandler does
// not share locks with it.
type UISessionState struct {
	mu                    sync.Mutex
	lastSetUIByDevice     map[string]ui.Descriptor
	lastUIHostEventByDev  map[string]int
	mainUIActivationByDev map[string]string
	multiWindowResume     map[string]multiWindowResumeState
}

// NewUISessionState returns an empty UI session state store.
func NewUISessionState() *UISessionState {
	return &UISessionState{
		lastSetUIByDevice:     map[string]ui.Descriptor{},
		lastUIHostEventByDev:  map[string]int{},
		mainUIActivationByDev: map[string]string{},
		multiWindowResume:     map[string]multiWindowResumeState{},
	}
}

// RememberSetUI scans the outgoing message slice for SetUI messages
// destined for deviceID (i.e. not relayed elsewhere) and stores the most
// recent one as the last UI for that device. It must be called with the
// slice that is about to be sent on the wire so the remembered value
// matches what the client actually receives.
func (s *UISessionState) RememberSetUI(deviceID string, responses []ServerMessage) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || len(responses) == 0 {
		return
	}
	for _, response := range responses {
		if response.SetUI == nil {
			continue
		}
		if relayTarget := strings.TrimSpace(response.RelayToDeviceID); relayTarget != "" {
			continue
		}
		s.mu.Lock()
		s.lastSetUIByDevice[deviceID] = *response.SetUI
		s.mu.Unlock()
	}
}

// LastSetUI returns the last remembered SetUI descriptor for deviceID and
// whether one was found. The returned descriptor is a copy.
func (s *UISessionState) LastSetUI(deviceID string) (ui.Descriptor, bool) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return ui.Descriptor{}, false
	}
	s.mu.Lock()
	stored, ok := s.lastSetUIByDevice[deviceID]
	s.mu.Unlock()
	return stored, ok
}

// SwapMainUIActivation atomically replaces the main-UI activation ID for
// deviceID, returning the prior value (empty if none). It is a no-op
// returning "" when either argument is blank.
func (s *UISessionState) SwapMainUIActivation(deviceID, activationID string) string {
	deviceID = strings.TrimSpace(deviceID)
	activationID = strings.TrimSpace(activationID)
	if deviceID == "" || activationID == "" {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	prior := strings.TrimSpace(s.mainUIActivationByDev[deviceID])
	s.mainUIActivationByDev[deviceID] = activationID
	return prior
}

// ForgetMainUIActivation removes the main-UI activation entry for
// deviceID. Used when a device disconnects.
func (s *UISessionState) ForgetMainUIActivation(deviceID string) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}
	s.mu.Lock()
	delete(s.mainUIActivationByDev, deviceID)
	s.mu.Unlock()
}

// CaptureMultiWindowResume snapshots the device's last UI under
// priorScenario unless a snapshot already exists for the device or
// priorScenario is itself "multi_window". The lookup of the prior UI and
// the insertion are atomic relative to other store operations.
func (s *UISessionState) CaptureMultiWindowResume(deviceID, priorScenario string) {
	deviceID = strings.TrimSpace(deviceID)
	priorScenario = strings.TrimSpace(priorScenario)
	if deviceID == "" || priorScenario == "multi_window" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.multiWindowResume[deviceID]; exists {
		return
	}
	storedUI, hasUI := s.lastSetUIByDevice[deviceID]
	s.multiWindowResume[deviceID] = multiWindowResumeState{
		PriorScenario: priorScenario,
		PriorUI:       storedUI,
		HasPriorUI:    hasUI,
	}
}

// TakeMultiWindowResume removes any pending multi-window resume snapshot
// for deviceID and returns its components: the prior scenario name, the
// prior UI descriptor (zero-valued if hasPriorUI is false), whether a
// prior UI was captured, and whether any snapshot existed at all (taken).
func (s *UISessionState) TakeMultiWindowResume(deviceID string) (priorScenario string, priorUI ui.Descriptor, hasPriorUI, taken bool) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return "", ui.Descriptor{}, false, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, exists := s.multiWindowResume[deviceID]
	if !exists {
		return "", ui.Descriptor{}, false, false
	}
	delete(s.multiWindowResume, deviceID)
	return state.PriorScenario, state.PriorUI, state.HasPriorUI, true
}

// UIHostBeforeCountAndAdvance returns the previously delivered UI host
// event count for deviceID and stores totalCount as the new high-water
// mark. The two operations are atomic.
func (s *UISessionState) UIHostBeforeCountAndAdvance(deviceID string, totalCount int) int {
	deviceID = strings.TrimSpace(deviceID)
	s.mu.Lock()
	before := s.lastUIHostEventByDev[deviceID]
	s.lastUIHostEventByDev[deviceID] = totalCount
	s.mu.Unlock()
	return before
}

// MarkUIHostDelivered records that each non-empty deviceID in the set has
// been delivered up through totalCount.
func (s *UISessionState) MarkUIHostDelivered(deviceIDs map[string]struct{}, totalCount int) {
	if len(deviceIDs) == 0 {
		return
	}
	s.mu.Lock()
	for deviceID := range deviceIDs {
		if deviceID == "" {
			continue
		}
		s.lastUIHostEventByDev[deviceID] = totalCount
	}
	s.mu.Unlock()
}
