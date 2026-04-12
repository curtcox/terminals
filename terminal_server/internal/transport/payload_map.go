package transport

import "sort"

// DataEntry is a stable key-value item for adapter payload encoding.
type DataEntry struct {
	Key   string
	Value string
}

// EncodeDataMap converts a map to deterministic key-sorted entries.
func EncodeDataMap(data map[string]string) []DataEntry {
	if len(data) == 0 {
		return nil
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]DataEntry, 0, len(keys))
	for _, k := range keys {
		out = append(out, DataEntry{Key: k, Value: data[k]})
	}
	return out
}

// DecodeDataEntries converts entries back to a map. Later duplicate keys win.
func DecodeDataEntries(entries []DataEntry) map[string]string {
	if len(entries) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		out[e.Key] = e.Value
	}
	return out
}
