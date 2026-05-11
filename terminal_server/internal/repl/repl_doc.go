package repl

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

var (
	docsRootOnce sync.Once
	docsRootPath string
)

func resolveDocsRoot() string {
	docsRootOnce.Do(func() {
		docsRootPath = discoverDocsRoot()
	})
	return docsRootPath
}

func discoverDocsRoot() string {
	envRoot := strings.TrimSpace(os.Getenv("TERMINALS_REPL_DOCS_ROOT"))
	if envRoot != "" {
		if dirExists(filepath.Join(envRoot, "docs", "repl")) {
			return filepath.Join(envRoot, "docs", "repl")
		}
		if dirExists(envRoot) && strings.HasSuffix(filepath.ToSlash(envRoot), "/docs/repl") {
			return envRoot
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		if found := findDocsRootFrom(cwd); found != "" {
			return found
		}
	}
	if _, sourceFile, _, ok := runtime.Caller(0); ok {
		if found := findDocsRootFrom(filepath.Dir(sourceFile)); found != "" {
			return found
		}
	}
	return filepath.Join("docs", "repl")
}

func findDocsRootFrom(start string) string {
	dir := filepath.Clean(strings.TrimSpace(start))
	if dir == "" {
		return ""
	}
	for {
		candidate := filepath.Join(dir, "docs", "repl")
		if dirExists(candidate) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func listDocTopics(root string) ([]string, error) {
	out := make([]string, 0, 32)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = strings.TrimSuffix(filepath.ToSlash(rel), ".md")
		if rel == "index" || rel == "." {
			out = append(out, "repl/index")
			return nil
		}
		out = append(out, "repl/"+rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func searchDocTopics(root, query string) ([]string, error) {
	if query == "" {
		return listDocTopics(root)
	}
	out := make([]string, 0, 16)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		topic := "repl/" + strings.TrimSuffix(filepath.ToSlash(rel), ".md")
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(strings.ToLower(topic), query) || strings.Contains(strings.ToLower(string(content)), query) {
			out = append(out, topic)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func resolveDocTopicPath(root, topic string) string {
	topic = strings.TrimSpace(topic)
	topic = strings.TrimPrefix(topic, "repl/")
	topic = strings.TrimSuffix(topic, ".md")
	if topic == "" || topic == "repl" {
		topic = "index"
	}
	return filepath.Join(root, filepath.FromSlash(topic)+".md")
}
