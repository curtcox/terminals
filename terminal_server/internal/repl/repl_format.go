package repl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

func formatUnixMillis(raw any) string {
	switch typed := raw.(type) {
	case float64:
		if typed <= 0 {
			return ""
		}
		return time.UnixMilli(int64(typed)).UTC().Format(time.RFC3339)
	case int64:
		if typed <= 0 {
			return ""
		}
		return time.UnixMilli(typed).UTC().Format(time.RFC3339)
	case json.Number:
		n, err := typed.Int64()
		if err != nil || n <= 0 {
			return ""
		}
		return time.UnixMilli(n).UTC().Format(time.RFC3339)
	case string:
		if strings.TrimSpace(typed) == "" {
			return ""
		}
		if parsed, err := time.Parse(time.RFC3339Nano, typed); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
		if parsed, err := time.Parse(time.RFC3339, typed); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
		return typed
	default:
		return ""
	}
}

func printTable(out io.Writer, headers []string, rows [][]string) error {
	if len(headers) == 0 {
		return nil
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i := range headers {
			if i >= len(row) {
				continue
			}
			if len(row[i]) > widths[i] {
				widths[i] = len(row[i])
			}
		}
	}
	var line bytes.Buffer
	for i, h := range headers {
		if i > 0 {
			line.WriteString("  ")
		}
		line.WriteString(padRight(h, widths[i]))
	}
	if _, err := fmt.Fprintln(out, line.String()); err != nil {
		return err
	}
	for _, row := range rows {
		line.Reset()
		for i := range headers {
			if i > 0 {
				line.WriteString("  ")
			}
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			line.WriteString(padRight(cell, widths[i]))
		}
		if _, err := fmt.Fprintln(out, line.String()); err != nil {
			return err
		}
	}
	return nil
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func emptyAsNone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "none"
	}
	return value
}

func toString(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func writeJSON(out io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(b))
	return err
}

func lookupMapAny(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value
		}
	}
	return nil
}

func toAnySlice(v any) []any {
	switch typed := v.(type) {
	case []any:
		return typed
	case []string:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func joinAnyStrings(value any, sep string) string {
	items, ok := value.([]any)
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(toString(item))
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, sep)
}

func defaultIfBlank(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func decodeEscapes(in string) string {
	quoted := strconv.Quote(in)
	decoded, err := strconv.Unquote(quoted)
	if err != nil {
		return in
	}
	decoded, err = strconv.Unquote("\"" + strings.ReplaceAll(decoded, "\"", "\\\"") + "\"")
	if err != nil {
		return decoded
	}
	return decoded
}

func migrationPendingRecordSummary(migration map[string]any) string {
	if migration == nil {
		return "none"
	}
	records := toAnySlice(migration["pending_records"])
	if len(records) == 0 {
		return "none"
	}
	ids := make([]string, 0, len(records))
	for _, raw := range records {
		record, _ := raw.(map[string]any)
		recordID := strings.TrimSpace(toString(record["record_id"]))
		if recordID == "" {
			continue
		}
		resolution := strings.TrimSpace(toString(record["recommended_resolution"]))
		if resolution != "" {
			recordID += ":" + resolution
		}
		ids = append(ids, recordID)
	}
	if len(ids) == 0 {
		return "none"
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		exists := false
		for _, existing := range out {
			if strings.EqualFold(existing, trimmed) {
				exists = true
				break
			}
		}
		if !exists {
			out = append(out, trimmed)
		}
	}
	return out
}
