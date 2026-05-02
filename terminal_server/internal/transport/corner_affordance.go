package transport

import (
	"math"
	"strconv"
	"strings"

	capv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

const (
	cornerTopLeft     = "top-left"
	cornerTopRight    = "top-right"
	cornerBottomLeft  = "bottom-left"
	cornerBottomRight = "bottom-right"

	minCornerHitDp = 44
)

// CornerAffordanceConfig captures the merged corner-affordance settings the
// wrapper should apply. All fields except SafeArea are required to be set by
// callers via MergeCornerAffordanceConfig; that function fills defaults.
type CornerAffordanceConfig struct {
	Corner   string
	Visible  bool
	MinHitDp int
	Density  float64
	SafeArea *capv1.Insets
}

func defaultCornerAffordanceConfig() CornerAffordanceConfig {
	return CornerAffordanceConfig{
		Corner:   defaultCornerPlacement,
		Visible:  true,
		MinHitDp: minCornerHitDp,
		Density:  1.0,
	}
}

// MergeCornerAffordanceConfig resolves the effective corner-affordance config
// from an optional user preference and an optional activity override.
//
// Activity override wins while present; user preference applies when no
// override is set; defaults apply when neither is provided. The MinHitDp
// floor is enforced so a misconfigured layer cannot regress accessibility.
func MergeCornerAffordanceConfig(userPref, activityOverride *CornerAffordanceConfig) CornerAffordanceConfig {
	switch {
	case activityOverride != nil:
		return normalizeCornerConfig(*activityOverride)
	case userPref != nil:
		return normalizeCornerConfig(*userPref)
	default:
		return defaultCornerAffordanceConfig()
	}
}

func normalizeCornerConfig(cfg CornerAffordanceConfig) CornerAffordanceConfig {
	switch cfg.Corner {
	case cornerTopLeft, cornerTopRight, cornerBottomLeft, cornerBottomRight:
	default:
		cfg.Corner = defaultCornerPlacement
	}
	if cfg.MinHitDp < minCornerHitDp {
		cfg.MinHitDp = minCornerHitDp
	}
	if cfg.Density <= 0 {
		cfg.Density = 1.0
	}
	return cfg
}

// withCornerAffordance is the legacy entry point retained for callers that do
// not yet have a merged config. It uses defaults plus the supplied owner.
func withCornerAffordance(root ui.Descriptor, ownerID string) ui.Descriptor {
	return withCornerAffordanceConfig(root, ownerID, defaultCornerAffordanceConfig())
}

// withCornerAffordanceConfig injects a corner-affordance subtree into root for
// the activation identified by ownerID. If the affordance is invisible per
// cfg, the root is returned unchanged. If an affordance with the canonical
// scoped id is already present, the call is idempotent.
func withCornerAffordanceConfig(root ui.Descriptor, ownerID string, cfg CornerAffordanceConfig) ui.Descriptor {
	cfg = normalizeCornerConfig(cfg)
	cornerID := scopedAffordanceID(ownerID, cornerAffordanceLogicalID)
	if hasNodeID(root, cornerID) {
		return root
	}
	if !cfg.Visible {
		return root
	}

	hitPx := physicalHitTargetPixels(cfg.MinHitDp, cfg.Density)
	insetTopDp, insetRightDp, insetBottomDp, insetLeftDp := safeAreaInsetsForCorner(cfg.Corner, cfg.SafeArea)

	button := ui.New("button", map[string]string{
		"id":              cornerID,
		"label":           "Menu",
		"action":          "corner.open",
		"corner":          cfg.Corner,
		"visible":         "true",
		"min_hit_dp":      strconv.Itoa(cfg.MinHitDp),
		"min_hit_px":      strconv.Itoa(hitPx),
		"density":         formatDensity(cfg.Density),
		"inset_top_dp":    strconv.FormatInt(int64(insetTopDp), 10),
		"inset_right_dp":  strconv.FormatInt(int64(insetRightDp), 10),
		"inset_bottom_dp": strconv.FormatInt(int64(insetBottomDp), 10),
		"inset_left_dp":   strconv.FormatInt(int64(insetLeftDp), 10),
	})

	if strings.TrimSpace(root.Type) == "stack" {
		root.Children = append(root.Children, button)
		return root
	}
	return ui.New("stack", map[string]string{
		"id": "corner_affordance_root",
	}, root, button)
}

// physicalHitTargetPixels converts a dp hit-target floor to physical pixels at
// the supplied density, rounding up so the rendered target never drops below
// the dp minimum.
func physicalHitTargetPixels(minHitDp int, density float64) int {
	if density <= 0 {
		density = 1.0
	}
	return int(math.Ceil(float64(minHitDp) * density))
}

// safeAreaInsetsForCorner returns dp insets to apply to the wrapper's
// positioning for the requested corner, honoring the supplied safe_area.
// Inset values not relevant to the requested corner are zero.
func safeAreaInsetsForCorner(corner string, safe *capv1.Insets) (top, right, bottom, left int32) {
	if safe == nil {
		return 0, 0, 0, 0
	}
	switch corner {
	case cornerTopLeft:
		return safe.GetTop(), 0, 0, safe.GetLeft()
	case cornerTopRight:
		return safe.GetTop(), safe.GetRight(), 0, 0
	case cornerBottomLeft:
		return 0, 0, safe.GetBottom(), safe.GetLeft()
	default:
		return 0, safe.GetRight(), safe.GetBottom(), 0
	}
}

func formatDensity(d float64) string {
	return strconv.FormatFloat(d, 'f', -1, 64)
}
