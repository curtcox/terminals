package repl

import (
	"fmt"
	"io"
	"sort"
)

func printClaimsByDeviceTree(out io.Writer, claimsByDevice map[string]any) error {
	if len(claimsByDevice) == 0 {
		_, err := fmt.Fprintln(out, "(no claims)")
		return err
	}
	deviceIDs := make([]string, 0, len(claimsByDevice))
	for deviceID := range claimsByDevice {
		deviceIDs = append(deviceIDs, deviceID)
	}
	sort.Strings(deviceIDs)
	for _, deviceID := range deviceIDs {
		if err := printDeviceClaims(out, deviceID, claimsByDevice[deviceID]); err != nil {
			return err
		}
	}
	return nil
}

func printDeviceClaims(out io.Writer, deviceID string, claimsAny any) error {
	if _, err := fmt.Fprintf(out, "%s\n", deviceID); err != nil {
		return err
	}
	claims, _ := claimsAny.([]any)
	if len(claims) == 0 {
		_, err := fmt.Fprintln(out, "  (none)")
		return err
	}
	for _, claimAny := range claims {
		claim, _ := claimAny.(map[string]any)
		if claim == nil {
			continue
		}
		if _, err := fmt.Fprintf(out, "  - %s by %s\n", toString(claim["resource"]), toString(claim["activation_id"])); err != nil {
			return err
		}
	}
	return nil
}
