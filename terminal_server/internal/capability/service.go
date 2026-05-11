// Package capability provides typed in-memory services for REPL capability closure.
package capability

import (
	"sync"
	"time"
)

// Identity represents a user or system principal in the capability service.
type Identity struct {
	ID          string            `json:"id"`
	DisplayName string            `json:"display_name,omitempty"`
	Groups      []string          `json:"groups,omitempty"`
	Aliases     []string          `json:"aliases,omitempty"`
	Preferences map[string]string `json:"preferences,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// InteractiveSession represents a collaborative session between participants.
type InteractiveSession struct {
	ID              string                  `json:"id"`
	Kind            string                  `json:"kind"`
	Target          string                  `json:"target"`
	Participants    []SessionParticipant    `json:"participants,omitempty"`
	AttachedDevices []string                `json:"attached_devices,omitempty"`
	ControlRequests []SessionControlRequest `json:"control_requests,omitempty"`
	ControlGrants   []SessionControlGrant   `json:"control_grants,omitempty"`
	Audit           []SessionAuditEvent     `json:"audit,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

// SessionParticipant records a single identity's membership in a session.
type SessionParticipant struct {
	IdentityID string    `json:"identity_id"`
	JoinedAt   time.Time `json:"joined_at"`
}

// SessionControlRequest records a request from one participant to take control.
type SessionControlRequest struct {
	ParticipantID string    `json:"participant_id"`
	ControlType   string    `json:"control_type"`
	RequestedAt   time.Time `json:"requested_at"`
}

// SessionControlGrant records an approved control grant for one participant.
type SessionControlGrant struct {
	ParticipantID string    `json:"participant_id"`
	GrantedBy     string    `json:"granted_by"`
	ControlType   string    `json:"control_type"`
	GrantedAt     time.Time `json:"granted_at"`
}

// SessionAuditEvent records one control/share lifecycle event.
type SessionAuditEvent struct {
	Action    string    `json:"action"`
	Actor     string    `json:"actor,omitempty"`
	Target    string    `json:"target,omitempty"`
	Meta      string    `json:"meta,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// MessageRoom represents a durable room for conversation history.
type MessageRoom struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Audience      string    `json:"audience,omitempty"`
	RetentionDays int       `json:"retention_days,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Message represents a posted message within a room.
type Message struct {
	ID              string    `json:"id"`
	Room            string    `json:"room"`
	TargetRef       string    `json:"target_ref,omitempty"`
	Text            string    `json:"text"`
	ThreadRootRef   string    `json:"thread_root_ref,omitempty"`
	ThreadParentRef string    `json:"thread_parent_ref,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// BoardItem represents a pinned item on a named board.
type BoardItem struct {
	ID        string    `json:"id"`
	Board     string    `json:"board"`
	Pinned    bool      `json:"pinned,omitempty"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// Artifact represents a stored artifact such as a document or media object.
type Artifact struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ArtifactVersion records one durable version entry for an artifact.
type ArtifactVersion struct {
	ArtifactID string    `json:"artifact_id"`
	Version    int       `json:"version"`
	Kind       string    `json:"kind"`
	Title      string    `json:"title"`
	Action     string    `json:"action"`
	CreatedAt  time.Time `json:"created_at"`
}

// ArtifactTemplate records a reusable artifact template keyed by name.
type ArtifactTemplate struct {
	Name             string    `json:"name"`
	SourceArtifactID string    `json:"source_artifact_id"`
	SourceKind       string    `json:"source_kind"`
	SourceTitle      string    `json:"source_title"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Annotation represents a user annotation attached to a canvas.
type Annotation struct {
	ID        string    `json:"id"`
	Canvas    string    `json:"canvas"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// SearchResult represents a single item returned by a search query.
type SearchResult struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// MemoryEntry represents a stored memory item scoped to a named context.
type MemoryEntry struct {
	ID        string    `json:"id"`
	Scope     string    `json:"scope"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// Acknowledgement records that an identity has acknowledged a subject reference.
type Acknowledgement struct {
	IdentityID     string    `json:"identity_id"`
	SubjectRef     string    `json:"subject_ref"`
	ActorRef       string    `json:"actor_ref,omitempty"`
	Mode           string    `json:"mode,omitempty"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
}

// RecentItem represents a recent activity entry in the capability service.
type RecentItem struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type searchableItem struct {
	ID        string
	Kind      string
	Text      string
	CreatedAt time.Time
}

// StoreRecord represents a key/value entry in a named namespace store.
type StoreRecord struct {
	Namespace string     `json:"namespace"`
	Key       string     `json:"key"`
	Value     string     `json:"value"`
	Binding   string     `json:"binding,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// StoreNamespaceSummary represents aggregate inventory for one store namespace.
type StoreNamespaceSummary struct {
	Name        string `json:"name"`
	RecordCount int    `json:"record_count"`
}

// DeviceCohort represents a reusable named selector set for device targeting.
type DeviceCohort struct {
	Name      string    `json:"name"`
	Selectors []string  `json:"selectors,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UIView represents one authored UI view record.
type UIView struct {
	ViewID     string    `json:"view_id"`
	RootID     string    `json:"root_id,omitempty"`
	Descriptor string    `json:"descriptor,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// UISnapshot represents current server-authored UI state for one device.
type UISnapshot struct {
	DeviceID                  string    `json:"device_id"`
	RootID                    string    `json:"root_id,omitempty"`
	Descriptor                string    `json:"descriptor,omitempty"`
	LastPatchComponentID      string    `json:"last_patch_component_id,omitempty"`
	LastPatchDescriptor       string    `json:"last_patch_descriptor,omitempty"`
	LastTransitionComponentID string    `json:"last_transition_component_id,omitempty"`
	LastTransition            string    `json:"last_transition,omitempty"`
	LastTransitionDurationMS  int       `json:"last_transition_duration_ms,omitempty"`
	Subscriptions             []string  `json:"subscriptions,omitempty"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

// UIBroadcast represents one fan-out UI operation targeting a named cohort.
type UIBroadcast struct {
	Cohort     string    `json:"cohort"`
	Descriptor string    `json:"descriptor,omitempty"`
	PatchID    string    `json:"patch_id,omitempty"`
	Devices    []string  `json:"devices,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// BusEvent represents a named event emitted on the internal event bus.
type BusEvent struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	Payload   string    `json:"payload,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// HandlerRegistration represents one runtime input/event routing rule.
type HandlerRegistration struct {
	ID          string    `json:"id"`
	Selector    string    `json:"selector"`
	Action      string    `json:"action"`
	RunCommand  string    `json:"run_command,omitempty"`
	EmitKind    string    `json:"emit_kind,omitempty"`
	EmitName    string    `json:"emit_name,omitempty"`
	EmitPayload string    `json:"emit_payload,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// InlineScenarioEventHook binds one event kind to a REPL command fragment.
type InlineScenarioEventHook struct {
	Kind    string `json:"kind"`
	Command string `json:"command"`
}

// InlineScenarioDefinition stores one REPL-authored scenario definition.
type InlineScenarioDefinition struct {
	Name         string                    `json:"name"`
	MatchIntents []string                  `json:"match_intents,omitempty"`
	MatchEvents  []string                  `json:"match_events,omitempty"`
	Priority     string                    `json:"priority"`
	OnStart      string                    `json:"on_start,omitempty"`
	OnInput      string                    `json:"on_input,omitempty"`
	OnEvents     []InlineScenarioEventHook `json:"on_events,omitempty"`
	OnSuspend    string                    `json:"on_suspend,omitempty"`
	OnResume     string                    `json:"on_resume,omitempty"`
	OnStop       string                    `json:"on_stop,omitempty"`
	UpdatedAt    time.Time                 `json:"updated_at"`
}

// SimDevice represents one virtual device registered for simulation workflows.
type SimDevice struct {
	DeviceID  string    `json:"device_id"`
	Caps      []string  `json:"caps,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SimInputRecord captures one synthetic input event delivered to a sim device.
type SimInputRecord struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"device_id"`
	ComponentID string    `json:"component_id"`
	Action      string    `json:"action"`
	Value       string    `json:"value,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// SimExpectationResult captures one expectation check against simulation state.
type SimExpectationResult struct {
	DeviceID  string    `json:"device_id"`
	Kind      string    `json:"kind"`
	Selector  string    `json:"selector,omitempty"`
	Within    string    `json:"within,omitempty"`
	Matched   bool      `json:"matched"`
	Reason    string    `json:"reason,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// SimRecordResult captures simulation state over a requested recording window.
type SimRecordResult struct {
	DeviceID  string           `json:"device_id"`
	Duration  string           `json:"duration,omitempty"`
	StartedAt time.Time        `json:"started_at"`
	EndedAt   time.Time        `json:"ended_at"`
	Snapshot  UISnapshot       `json:"snapshot"`
	Inputs    []SimInputRecord `json:"inputs,omitempty"`
	Messages  []BusEvent       `json:"messages,omitempty"`
}

// ScriptDryRunResult summarizes parsed commands from a scripts dry-run call.
type ScriptDryRunResult struct {
	Path         string    `json:"path"`
	CommandCount int       `json:"command_count"`
	SkippedCount int       `json:"skipped_count"`
	Commands     []string  `json:"commands,omitempty"`
	Issues       []string  `json:"issues,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ScriptRunResult summarizes command execution from a scripts run call.
type ScriptRunResult struct {
	Path          string    `json:"path"`
	CommandCount  int       `json:"command_count"`
	SkippedCount  int       `json:"skipped_count"`
	ExecutedCount int       `json:"executed_count"`
	FailedCount   int       `json:"failed_count"`
	Commands      []string  `json:"commands,omitempty"`
	Issues        []string  `json:"issues,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Service provides typed in-memory storage for capability closure tools.
type Service struct {
	mu sync.RWMutex

	now func() time.Time
	seq uint64

	identities   []Identity
	sessions     []InteractiveSession
	messageRooms []MessageRoom
	messages     []Message
	boardItems   []BoardItem
	artifacts    []Artifact
	versions     map[string][]ArtifactVersion
	templates    map[string]ArtifactTemplate
	annotations  []Annotation
	memories     []MemoryEntry
	recent       []RecentItem
	store        map[string]StoreRecord
	bus          []BusEvent
	handlers     map[string]HandlerRegistration
	cohorts      map[string]DeviceCohort
	uiViews      map[string]UIView
	uiSnapshots  map[string]UISnapshot
	uiSubs       map[string][]string
	scenarios    map[string]InlineScenarioDefinition
	simDevices   map[string]SimDevice
	simInputs    map[string][]SimInputRecord
	acks         map[string]Acknowledgement
}

// NewService creates a new Service with default seed data.
func NewService() *Service {
	now := time.Now
	s := &Service{
		now:         func() time.Time { return now().UTC() },
		store:       map[string]StoreRecord{},
		handlers:    map[string]HandlerRegistration{},
		cohorts:     map[string]DeviceCohort{},
		uiViews:     map[string]UIView{},
		uiSnapshots: map[string]UISnapshot{},
		uiSubs:      map[string][]string{},
		scenarios:   map[string]InlineScenarioDefinition{},
		simDevices:  map[string]SimDevice{},
		simInputs:   map[string][]SimInputRecord{},
		acks:        map[string]Acknowledgement{},
		versions:    map[string][]ArtifactVersion{},
		templates:   map[string]ArtifactTemplate{},
		identities: []Identity{
			{
				ID:          "system",
				DisplayName: "System",
				Groups:      []string{"family", "operators"},
				Aliases:     []string{"admin", "house"},
				Preferences: map[string]string{"notifications": "normal", "default_zone": "house"},
				CreatedAt:   now().UTC(),
			},
		},
	}
	s.createMessageRoomLocked("general")
	return s
}
