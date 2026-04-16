package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/api/apps" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"apps": []map[string]any{{
				"name":     "sound_watch",
				"version":  "1.2.3",
				"revision": 7,
			}},
		})
	}))
	defer server.Close()

	t.Setenv("TERM_ADMIN_URL", server.URL)
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
