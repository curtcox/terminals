package transport

import (
	"strconv"
	"strings"
	"testing"

	capv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestMergeCornerAffordanceConfig_DefaultWhenUnset(t *testing.T) {
	got := MergeCornerAffordanceConfig(nil, nil)
	if got.Corner != cornerBottomRight {
		t.Fatalf("default corner = %q, want %q", got.Corner, cornerBottomRight)
	}
	if !got.Visible {
		t.Fatalf("default visible = false, want true")
	}
	if got.MinHitDp != minCornerHitDp {
		t.Fatalf("default MinHitDp = %d, want %d", got.MinHitDp, minCornerHitDp)
	}
}

func TestMergeCornerAffordanceConfig_UserPrefWinsWhenAlone(t *testing.T) {
	user := &CornerAffordanceConfig{Corner: cornerTopLeft, Visible: true, MinHitDp: 60, Density: 2.0}
	got := MergeCornerAffordanceConfig(user, nil)
	if got.Corner != cornerTopLeft {
		t.Fatalf("user-pref corner = %q, want top-left", got.Corner)
	}
	if got.MinHitDp != 60 {
		t.Fatalf("user-pref MinHitDp = %d, want 60", got.MinHitDp)
	}
}

func TestMergeCornerAffordanceConfig_ActivityOverrideWinsOverUserPref(t *testing.T) {
	user := &CornerAffordanceConfig{Corner: cornerTopLeft, Visible: true, MinHitDp: 44, Density: 1.0}
	activity := &CornerAffordanceConfig{Corner: cornerBottomLeft, Visible: true, MinHitDp: 56, Density: 2.0}

	got := MergeCornerAffordanceConfig(user, activity)
	if got.Corner != cornerBottomLeft {
		t.Fatalf("override corner = %q, want bottom-left", got.Corner)
	}
	if got.MinHitDp != 56 {
		t.Fatalf("override MinHitDp = %d, want 56", got.MinHitDp)
	}
}

func TestMergeCornerAffordanceConfig_UserPrefRestoredAfterActivityExit(t *testing.T) {
	user := &CornerAffordanceConfig{Corner: cornerTopRight, Visible: true, MinHitDp: 44, Density: 1.0}
	activity := &CornerAffordanceConfig{Corner: cornerBottomLeft, Visible: true, MinHitDp: 44, Density: 1.0}

	withActivity := MergeCornerAffordanceConfig(user, activity)
	if withActivity.Corner != cornerBottomLeft {
		t.Fatalf("with activity, corner = %q, want bottom-left", withActivity.Corner)
	}
	// Activity exits → only user pref remains.
	afterActivity := MergeCornerAffordanceConfig(user, nil)
	if afterActivity.Corner != cornerTopRight {
		t.Fatalf("after activity exit, corner = %q, want top-right (user pref)", afterActivity.Corner)
	}
}

func TestMergeCornerAffordanceConfig_EnforcesMinHitFloor(t *testing.T) {
	user := &CornerAffordanceConfig{Corner: cornerTopLeft, Visible: true, MinHitDp: 12, Density: 1.0}
	got := MergeCornerAffordanceConfig(user, nil)
	if got.MinHitDp < minCornerHitDp {
		t.Fatalf("MinHitDp = %d, want >= %d (floor)", got.MinHitDp, minCornerHitDp)
	}
}

func TestWithCornerAffordanceConfig_PerCornerPlacement(t *testing.T) {
	for _, corner := range []string{cornerTopLeft, cornerTopRight, cornerBottomLeft, cornerBottomRight} {
		corner := corner
		t.Run(corner, func(t *testing.T) {
			root := ui.New("stack", map[string]string{"id": "root"})
			cfg := defaultCornerAffordanceConfig()
			cfg.Corner = corner
			got := withCornerAffordanceConfig(root, "device-1", cfg)
			node := findNodeByID(&got, "act:device-1/__affordance.corner__")
			if node == nil {
				t.Fatalf("missing scoped corner node for corner=%s", corner)
			}
			if node.Props["corner"] != corner {
				t.Fatalf("emitted corner = %q, want %q", node.Props["corner"], corner)
			}
		})
	}
}

func TestWithCornerAffordanceConfig_HitTargetMeetsMinimumAtDensity(t *testing.T) {
	cases := []struct {
		density   float64
		wantMinPx int
	}{
		{density: 1.0, wantMinPx: 44},
		{density: 2.0, wantMinPx: 88},
		{density: 3.0, wantMinPx: 132},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(strconv.FormatFloat(tc.density, 'f', -1, 64), func(t *testing.T) {
			root := ui.New("stack", map[string]string{"id": "root"})
			cfg := defaultCornerAffordanceConfig()
			cfg.Density = tc.density
			got := withCornerAffordanceConfig(root, "device-1", cfg)
			node := findNodeByID(&got, "act:device-1/__affordance.corner__")
			if node == nil {
				t.Fatalf("missing scoped corner node")
			}
			gotDp, err := strconv.Atoi(node.Props["min_hit_dp"])
			if err != nil {
				t.Fatalf("min_hit_dp parse error: %v", err)
			}
			if gotDp < minCornerHitDp {
				t.Fatalf("min_hit_dp = %d, want >= %d", gotDp, minCornerHitDp)
			}
			gotPx, err := strconv.Atoi(node.Props["min_hit_px"])
			if err != nil {
				t.Fatalf("min_hit_px parse error: %v", err)
			}
			if gotPx < tc.wantMinPx {
				t.Fatalf("min_hit_px at density %v = %d, want >= %d", tc.density, gotPx, tc.wantMinPx)
			}
		})
	}
}

func TestWithCornerAffordanceConfig_ZOrderIsLastChildOfContainingStack(t *testing.T) {
	root := ui.New("stack", map[string]string{"id": "root"},
		ui.New("text", map[string]string{"id": "headline"}),
		ui.New("button", map[string]string{"id": "primary"}),
	)
	got := withCornerAffordanceConfig(root, "device-1", defaultCornerAffordanceConfig())
	if len(got.Children) == 0 {
		t.Fatalf("expected children")
	}
	last := got.Children[len(got.Children)-1]
	if last.ID != "act:device-1/__affordance.corner__" && last.Props["id"] != "act:device-1/__affordance.corner__" {
		t.Fatalf("last child id = %q (props.id %q), want corner-affordance scoped id",
			last.ID, last.Props["id"])
	}
}

func TestWithCornerAffordanceConfig_ZOrderWhenWrappingNonStackRoot(t *testing.T) {
	root := ui.New("fullscreen", map[string]string{"id": "app_root"},
		ui.New("text", map[string]string{"id": "body"}),
	)
	got := withCornerAffordanceConfig(root, "device-1", defaultCornerAffordanceConfig())
	if got.Type != "stack" {
		t.Fatalf("non-stack root must be wrapped in stack, got %q", got.Type)
	}
	if len(got.Children) < 2 {
		t.Fatalf("wrapped tree expected >=2 children, got %d", len(got.Children))
	}
	last := got.Children[len(got.Children)-1]
	if last.Props["id"] != "act:device-1/__affordance.corner__" && last.ID != "act:device-1/__affordance.corner__" {
		t.Fatalf("corner affordance must be last child of wrapping stack")
	}
}

func TestWithCornerAffordanceConfig_AsymmetricSafeAreaNonOcclusion(t *testing.T) {
	// Notch on top, fat bottom inset, a sliver on the right.
	safe := &capv1.Insets{Left: 8, Top: 50, Right: 24, Bottom: 80}

	cases := []struct {
		corner                                   string
		wantTop, wantRight, wantBottom, wantLeft int32
	}{
		{cornerTopLeft, 50, 0, 0, 8},
		{cornerTopRight, 50, 24, 0, 0},
		{cornerBottomLeft, 0, 0, 80, 8},
		{cornerBottomRight, 0, 24, 80, 0},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.corner, func(t *testing.T) {
			root := ui.New("stack", map[string]string{"id": "root"})
			cfg := defaultCornerAffordanceConfig()
			cfg.Corner = tc.corner
			cfg.SafeArea = safe
			got := withCornerAffordanceConfig(root, "device-1", cfg)
			node := findNodeByID(&got, "act:device-1/__affordance.corner__")
			if node == nil {
				t.Fatalf("missing scoped corner node")
			}
			gotInset := func(key string) int32 {
				v, err := strconv.Atoi(node.Props[key])
				if err != nil {
					t.Fatalf("inset prop %q parse error: %v", key, err)
				}
				return int32(v)
			}
			if gotInset("inset_top_dp") < tc.wantTop {
				t.Fatalf("inset_top_dp = %d, want >= %d for %s", gotInset("inset_top_dp"), tc.wantTop, tc.corner)
			}
			if gotInset("inset_right_dp") < tc.wantRight {
				t.Fatalf("inset_right_dp = %d, want >= %d for %s", gotInset("inset_right_dp"), tc.wantRight, tc.corner)
			}
			if gotInset("inset_bottom_dp") < tc.wantBottom {
				t.Fatalf("inset_bottom_dp = %d, want >= %d for %s", gotInset("inset_bottom_dp"), tc.wantBottom, tc.corner)
			}
			if gotInset("inset_left_dp") < tc.wantLeft {
				t.Fatalf("inset_left_dp = %d, want >= %d for %s", gotInset("inset_left_dp"), tc.wantLeft, tc.corner)
			}
		})
	}
}

func TestWithCornerAffordanceConfig_InvisibleSkipsInjection(t *testing.T) {
	root := ui.New("stack", map[string]string{"id": "root"})
	cfg := defaultCornerAffordanceConfig()
	cfg.Visible = false
	got := withCornerAffordanceConfig(root, "device-1", cfg)
	if findNodeByID(&got, "act:device-1/__affordance.corner__") != nil {
		t.Fatalf("invisible config must not inject affordance")
	}
}

// TestWithCornerAffordance_RegistryReachabilityInvariant exercises every
// registered main-layer scenario by name against fixture-generated
// descriptors and asserts that the wrapper produces a tree containing
// exactly one corner-affordance node with the canonical scoped id and
// `corner.open` action handler. This is the typed-wrapper reachability
// invariant pinned in plans/features/terminal-ui/plan.md Phase A.
func TestWithCornerAffordance_RegistryReachabilityInvariant(t *testing.T) {
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	registry := engine.RegistrySnapshot()
	if len(registry) == 0 {
		t.Fatalf("registry empty; cannot run invariant")
	}

	fixtures := []ui.Descriptor{
		ui.New("stack", map[string]string{"id": "fixture_stack"}),
		ui.New("fullscreen", map[string]string{"id": "fixture_fullscreen"},
			ui.New("text", map[string]string{"id": "body"})),
		ui.New("stack", map[string]string{"id": "fixture_with_existing"},
			ui.New("button", map[string]string{"id": "user_button", "action": "noop"})),
	}

	for _, info := range registry {
		ownerID := "scenario:" + info.Name
		for i, fixture := range fixtures {
			fixture := fixture
			i := i
			info := info
			t.Run(info.Name+"/fixture_"+strconv.Itoa(i), func(t *testing.T) {
				wrapped := withCornerAffordance(fixture, ownerID)
				wantID := "act:" + ownerID + "/__affordance.corner__"
				count := countNodesWithIDPrefix(&wrapped, wantID)
				if count != 1 {
					t.Fatalf("scenario %q fixture %d: corner affordance count = %d, want exactly 1",
						info.Name, i, count)
				}
				node := findNodeByID(&wrapped, wantID)
				if node == nil {
					t.Fatalf("scenario %q fixture %d: missing scoped corner node", info.Name, i)
				}
				if strings.TrimSpace(node.Type) != "button" {
					t.Fatalf("scenario %q fixture %d: corner type = %q, want button",
						info.Name, i, node.Type)
				}
				if node.Props["action"] != "corner.open" {
					t.Fatalf("scenario %q fixture %d: corner action = %q, want corner.open",
						info.Name, i, node.Props["action"])
				}
				dp, err := strconv.Atoi(node.Props["min_hit_dp"])
				if err != nil || dp < minCornerHitDp {
					t.Fatalf("scenario %q fixture %d: min_hit_dp = %q, want >= %d",
						info.Name, i, node.Props["min_hit_dp"], minCornerHitDp)
				}
			})
		}
	}
}
