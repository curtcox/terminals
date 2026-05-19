package scenario

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func endpointResourceIDsForDevice(env *Environment, deviceID string, family string) []string {
	caps := capabilitiesForDevice(env, deviceID)
	if len(caps) == 0 {
		return nil
	}
	family = strings.TrimSpace(strings.ToLower(family))
	if family == "" {
		return nil
	}
	endpoints := parseEndpointIDsFromCaps(caps, family)
	if len(endpoints) == 0 {
		return nil
	}
	return sortedEndpointResourceIDs(endpoints)
}

func parseEndpointIDsFromCaps(caps map[string]string, family string) map[string]string {
	endpointPrefix := family + ".endpoint."
	endpoints := map[string]string{}
	for key, value := range caps {
		key = strings.TrimSpace(strings.ToLower(key))
		if !strings.HasPrefix(key, endpointPrefix) {
			continue
		}
		indexToken, ok := endpointIndexFromCapKey(key, endpointPrefix)
		if !ok {
			continue
		}
		if endpointID := sanitizeResourceToken(value); endpointID != "" {
			endpoints[indexToken] = endpointID
		}
	}
	return endpoints
}

func endpointIndexFromCapKey(key, endpointPrefix string) (string, bool) {
	remainder := strings.TrimPrefix(key, endpointPrefix)
	parts := strings.Split(remainder, ".")
	if len(parts) < 2 {
		return "", false
	}
	indexToken := strings.TrimSpace(parts[0])
	fieldToken := strings.TrimSpace(parts[1])
	if indexToken == "" || fieldToken != "id" {
		return "", false
	}
	index, err := strconv.Atoi(indexToken)
	if err != nil || index < 0 {
		return "", false
	}
	return indexToken, true
}

func sortedEndpointResourceIDs(endpoints map[string]string) []string {
	indexes := make([]int, 0, len(endpoints))
	for raw := range endpoints {
		idx, err := strconv.Atoi(raw)
		if err != nil {
			continue
		}
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	resolved := make([]string, 0, len(indexes))
	for _, index := range indexes {
		endpointID := endpoints[strconv.Itoa(index)]
		if endpointID == "" {
			endpointID = fmt.Sprintf("endpoint-%d", index)
		}
		resolved = append(resolved, endpointID)
	}
	return resolved
}
