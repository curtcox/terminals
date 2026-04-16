package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAppNewCreatesScaffold(t *testing.T) {
	cwd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cwd, "apps"), 0o755); err != nil {
		t.Fatalf("MkdirAll(apps) error = %v", err)
	}
	withCWD(t, cwd, func() {
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := run([]string{"app", "new", "sound_watch"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("run() code = %d, want 0 stderr=%s", code, errOut.String())
		}
		if _, err := os.Stat(filepath.Join(cwd, "apps", "sound_watch", "manifest.toml")); err != nil {
			t.Fatalf("manifest missing: %v", err)
		}
		if _, err := os.Stat(filepath.Join(cwd, "apps", "sound_watch", "main.tal")); err != nil {
			t.Fatalf("main.tal missing: %v", err)
		}
	})
}

func TestRunAppTestReportsTalTests(t *testing.T) {
	cwd := t.TempDir()
	createApp(t, cwd, "sound_watch", "1.0.0")
	if err := os.MkdirAll(filepath.Join(cwd, "apps", "sound_watch", "tests"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "apps", "sound_watch", "tests", "sound_watch_test.tal"), []byte("test('ok')\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(test) error = %v", err)
	}

	withCWD(t, cwd, func() {
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := run([]string{"app", "test", "sound_watch"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("run() code = %d, want 0 stderr=%s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "tests: 1 file(s)") {
			t.Fatalf("stdout = %q, want tests count", out.String())
		}
	})
}

func TestRunAppLogsUsesAdminAPI(t *testing.T) {
	originalClient := adminHTTPClient
	t.Cleanup(func() { adminHTTPClient = originalClient })
	adminHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/admin/api/apps" {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`{"error":"not found"}`)),
				Header:     make(http.Header),
			}, nil
		}
		payload, _ := json.Marshal(map[string]any{
			"apps": []map[string]any{{
				"name":     "sound_watch",
				"version":  "1.2.3",
				"revision": 7,
			}},
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(payload))),
			Header:     make(http.Header),
		}, nil
	})}

	t.Setenv("TERM_ADMIN_URL", "http://example.test")
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"app", "logs", "sound_watch"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("run() code = %d, want 0 stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"version": "1.2.3"`) {
		t.Fatalf("stdout = %q, want app payload", out.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestRunSimRunValidatesFixtureAndApp(t *testing.T) {
	cwd := t.TempDir()
	createApp(t, cwd, "sound_watch", "1.0.0")
	fixturePath := filepath.Join(cwd, "kitchen_house.yaml")
	if err := os.WriteFile(fixturePath, []byte("devices: []\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(fixture) error = %v", err)
	}

	withCWD(t, cwd, func() {
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := run([]string{"sim", "run", "sound_watch", "--fixture", fixturePath}, &out, &errOut)
		if code != 0 {
			t.Fatalf("run() code = %d, want 0 stderr=%s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "sim run ok") {
			t.Fatalf("stdout = %q, want sim run success", out.String())
		}
	})
}

func TestRunLogsSearchFiltersByEvent(t *testing.T) {
	cwd := t.TempDir()
	logsDir := filepath.Join(cwd, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(logs) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(logsDir, "terminals.jsonl"), []byte(
		`{"ts":"2026-04-16T10:00:00Z","seq":1,"event":"scenario.activation.started"}`+"\n"+
			`{"ts":"2026-04-16T10:00:01Z","seq":2,"event":"device.registered"}`+"\n",
	), 0o644); err != nil {
		t.Fatalf("WriteFile(terminals.jsonl) error = %v", err)
	}

	t.Setenv("TERMINALS_LOG_DIR", logsDir)
	withCWD(t, cwd, func() {
		var out bytes.Buffer
		var errOut bytes.Buffer
		code := run([]string{"logs", "search", "event=scenario.activation.started"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("run() code = %d, want 0 stderr=%s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "scenario.activation.started") {
			t.Fatalf("stdout = %q", out.String())
		}
		if strings.Contains(out.String(), "device.registered") {
			t.Fatalf("stdout should not include other event: %q", out.String())
		}
	})
}

func createApp(t *testing.T, cwd, name, version string) {
	t.Helper()
	root := filepath.Join(cwd, "apps", name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll(app root) error = %v", err)
	}
	manifest := "name = \"" + name + "\"\nversion = \"" + version + "\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(root, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.tal"), []byte("def on_start():\n  pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.tal) error = %v", err)
	}
}

func withCWD(t *testing.T, dir string, fn func()) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}
	defer func() {
		if chdirErr := os.Chdir(original); chdirErr != nil {
			t.Fatalf("restore Chdir(%q) error = %v", original, chdirErr)
		}
	}()
	fn()
}
