// Package query provides local filtering utilities for eventlog JSONL records.
package query

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Record is one decoded eventlog JSON object.
type Record map[string]any

// Filters defines supported search constraints for eventlog queries.
type Filters struct {
	Equals   map[string][]string
	LevelMin string
	Since    *time.Time
	Until    *time.Time
	FreeText []string
	TraceID  string
}

// ReadAll loads all available eventlog files from dir in chronological order.
func ReadAll(dir string) ([]Record, error) {
	files, err := logFiles(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Record, 0, 1024)
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		s := bufio.NewScanner(f)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line == "" {
				continue
			}
			var rec Record
			if err := json.Unmarshal([]byte(line), &rec); err != nil {
				continue
			}
			out = append(out, rec)
		}
		_ = f.Close()
	}
	sortRecords(out)
	return out, nil
}

// Search loads records then applies parsed filters.
func Search(dir string, args []string, now time.Time) ([]Record, error) {
	all, err := ReadAll(dir)
	if err != nil {
		return nil, err
	}
	filters, err := ParseFilters(args, now)
	if err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(all))
	for _, rec := range all {
		if matches(rec, filters) {
			out = append(out, rec)
		}
	}
	return out, nil
}

// ParseFilters parses CLI-style filter arguments into a Filters struct.
func ParseFilters(args []string, now time.Time) (Filters, error) {
	f := Filters{Equals: map[string][]string{}}
	for _, raw := range args {
		if err := parseFilterToken(&f, raw, now); err != nil {
			return Filters{}, err
		}
	}
	return f, nil
}

func parseFilterToken(f *Filters, raw string, now time.Time) error {
	tok := strings.TrimSpace(raw)
	if tok == "" {
		return nil
	}
	if k, v, ok := strings.Cut(tok, ">="); ok && strings.TrimSpace(strings.ToLower(k)) == "level" {
		f.LevelMin = strings.ToLower(strings.TrimSpace(v))
		return nil
	}
	k, v, ok := strings.Cut(tok, "=")
	if !ok {
		f.FreeText = append(f.FreeText, tok)
		return nil
	}
	return parseFieldFilter(f, strings.TrimSpace(strings.ToLower(k)), strings.TrimSpace(v), now)
}

func parseFieldFilter(f *Filters, key, val string, now time.Time) error {
	switch key {
	case "since":
		ts, err := parseTimeValue(val, now)
		if err != nil {
			return err
		}
		f.Since = &ts
	case "until":
		ts, err := parseTimeValue(val, now)
		if err != nil {
			return err
		}
		f.Until = &ts
	case "trace":
		f.TraceID = val
	default:
		f.Equals[key] = append(f.Equals[key], val)
	}
	return nil
}

// Trace returns records matching the provided trace id.
func Trace(records []Record, traceID string) []Record {
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		return nil
	}
	out := make([]Record, 0)
	for _, rec := range records {
		if fieldString(rec, "trace_id") == traceID {
			out = append(out, rec)
		}
	}
	sortRecords(out)
	return out
}

// Activation returns records matching the provided activation id.
func Activation(records []Record, activationID string) []Record {
	activationID = strings.TrimSpace(activationID)
	if activationID == "" {
		return nil
	}
	out := make([]Record, 0)
	for _, rec := range records {
		if fieldString(rec, "activation_id") == activationID {
			out = append(out, rec)
		}
	}
	sortRecords(out)
	return out
}

// Stats counts records grouped by one record key.
func Stats(records []Record, by string) map[string]int {
	by = strings.TrimSpace(strings.ToLower(by))
	if by == "" {
		by = "event"
	}
	out := map[string]int{}
	for _, rec := range records {
		key := fieldString(rec, by)
		if key == "" {
			key = "(empty)"
		}
		out[key]++
	}
	return out
}

func matches(rec Record, filters Filters) bool {
	return matchesTrace(rec, filters) &&
		matchesTimeWindow(rec, filters) &&
		matchesLevel(rec, filters) &&
		matchesEquals(rec, filters.Equals) &&
		matchesFreeText(rec, filters.FreeText)
}

func matchesTrace(rec Record, filters Filters) bool {
	return filters.TraceID == "" || fieldString(rec, "trace_id") == filters.TraceID
}

func matchesTimeWindow(rec Record, filters Filters) bool {
	if filters.Since == nil && filters.Until == nil {
		return true
	}
	ts := fieldTime(rec)
	if ts.IsZero() {
		return false
	}
	if filters.Since != nil && ts.Before(*filters.Since) {
		return false
	}
	return filters.Until == nil || !ts.After(*filters.Until)
}

func matchesLevel(rec Record, filters Filters) bool {
	return filters.LevelMin == "" || levelRank(fieldString(rec, "level")) >= levelRank(filters.LevelMin)
}

func matchesEquals(rec Record, equals map[string][]string) bool {
	for key, values := range equals {
		if !fieldMatchesAny(rec, key, values) {
			return false
		}
	}
	return true
}

func fieldMatchesAny(rec Record, key string, values []string) bool {
	if len(values) == 0 {
		return true
	}
	actual := fieldString(rec, key)
	if actual == "" {
		return false
	}
	for _, want := range values {
		if actual == want {
			return true
		}
	}
	return false
}

func matchesFreeText(rec Record, tokens []string) bool {
	if len(tokens) == 0 {
		return true
	}
	b, _ := json.Marshal(rec)
	hay := strings.ToLower(string(b))
	for _, token := range tokens {
		if !strings.Contains(hay, strings.ToLower(token)) {
			return false
		}
	}
	return true
}

func logFiles(dir string) ([]string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = "logs"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	base := "terminals.jsonl"
	type archived struct {
		idx  int
		path string
	}
	archives := make([]archived, 0)
	active := ""
	for _, entry := range entries {
		name := entry.Name()
		full := filepath.Join(dir, name)
		if name == base {
			active = full
			continue
		}
		if !strings.HasPrefix(name, base+".") {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimPrefix(name, base+"."))
		if err != nil {
			continue
		}
		archives = append(archives, archived{idx: idx, path: full})
	}
	sort.Slice(archives, func(i, j int) bool {
		return archives[i].idx > archives[j].idx
	})
	out := make([]string, 0, len(archives)+1)
	for _, a := range archives {
		out = append(out, a.path)
	}
	if active != "" {
		out = append(out, active)
	}
	return out, nil
}

func fieldString(rec Record, key string) string {
	if rec == nil {
		return ""
	}
	value, ok := pick(rec, key)
	if !ok {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func pick(rec map[string]any, key string) (any, bool) {
	parts := strings.Split(strings.TrimSpace(key), ".")
	cur := any(rec)
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := m[p]
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

func fieldTime(rec Record) time.Time {
	raw := fieldString(rec, "ts")
	if raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

func sortRecords(records []Record) {
	sort.Slice(records, func(i, j int) bool {
		si, iok := uint64FromAny(records[i]["seq"])
		sj, jok := uint64FromAny(records[j]["seq"])
		if iok && jok && si != sj {
			return si < sj
		}
		ti := fieldTime(records[i])
		tj := fieldTime(records[j])
		return ti.Before(tj)
	})
}

func uint64FromAny(v any) (uint64, bool) {
	switch t := v.(type) {
	case uint64:
		return t, true
	case float64:
		return uint64(t), true
	case int64:
		return uint64(t), true
	case int:
		return uint64(t), true
	case json.Number:
		n, err := strconv.ParseUint(t.String(), 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func parseTimeValue(raw string, now time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty time value")
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return now.Add(-d), nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time %q", raw)
	}
	return t, nil
}

func levelRank(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return 10
	case "info":
		return 20
	case "warn", "warning":
		return 30
	case "error":
		return 40
	default:
		return 0
	}
}
