package transport

import (
	"context"
	"errors"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func (h *StreamHandler) suspendClaimsForLostResources(
	ctx context.Context,
	routeIO interface {
		Claims() *iorouter.ClaimManager
	},
	deviceID string,
	lostResources map[string]struct{},
) {
	claims := routeIO.Claims()
	if claims == nil || len(lostResources) == 0 {
		return
	}
	suspendedClaims := make([]iorouter.Claim, 0)
	for _, claim := range claims.Snapshot(deviceID) {
		if _, exists := lostResources[claim.Resource]; !exists {
			continue
		}
		suspendedClaims = append(suspendedClaims, claim)
	}
	h.rememberSuspendedClaims(deviceID, suspendedClaims)
	if len(suspendedClaims) > 0 {
		_ = claims.ReleaseClaims(ctx, suspendedClaims)
	}
}

func (h *StreamHandler) disconnectRoutesForLostResources(
	routeIO interface {
		RoutesForDevice(deviceID string) []iorouter.Route
		Disconnect(sourceID, targetID, streamKind string) error
	},
	deviceID string,
	lostResources map[string]struct{},
) []ServerMessage {
	if len(lostResources) == 0 {
		return nil
	}
	routes := routeIO.RoutesForDevice(deviceID)
	out := make([]ServerMessage, 0, len(routes))
	for _, route := range routes {
		if !shouldDisconnectRouteForLostResources(route, deviceID, lostResources) {
			continue
		}
		if err := routeIO.Disconnect(route.SourceID, route.TargetID, route.StreamKind); err != nil && !errors.Is(err, iorouter.ErrRouteNotFound) {
			continue
		}
		out = append(out, ServerMessage{
			StopStream: &StopStreamResponse{StreamID: routeStreamID(route)},
		})
	}
	return out
}
