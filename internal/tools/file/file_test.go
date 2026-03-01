package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

// newExec returns an Executor rooted in a fresh temp dir.
func newExec(t *testing.T) *toolTypes.Executor {
	t.Helper()
	return &toolTypes.Executor{WorkPath: t.TempDir()}
}

// writeTemp writes content to a relative path inside e.WorkPath.
func writeTemp(t *testing.T, e *toolTypes.Executor, rel, content string) {
	t.Helper()
	full := filepath.Join(e.WorkPath, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// ---------- checkLine ----------

func TestCheckLine(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantOk bool
		wantEF toolTypes.Exclude
	}{
		{"empty string", "", false, toolTypes.Exclude{}},
		{"whitespace only", "   ", false, toolTypes.Exclude{}},
		{"comment", "# node_modules", false, toolTypes.Exclude{}},
		{"normal", "node_modules", true, toolTypes.Exclude{File: "node_modules", Negate: false}},
		{"negate", "!dist", true, toolTypes.Exclude{File: "dist", Negate: true}},
		{"leading slash", "/vendor", true, toolTypes.Exclude{File: "vendor", Negate: false}},
		{"trailing slash", "tmp/", true, toolTypes.Exclude{File: "tmp", Negate: false}},
		{"slash only", "/", false, toolTypes.Exclude{}},
		{"negate slash only", "!/", false, toolTypes.Exclude{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := checkLine(tt.input)
			if ok != tt.wantOk {
				t.Fatalf("checkLine(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if ok && got != tt.wantEF {
				t.Errorf("checkLine(%q) = %+v, want %+v", tt.input, got, tt.wantEF)
			}
		})
	}
}

// ---------- getFullPath ----------

func TestGetFullPath(t *testing.T) {
	e := &toolTypes.Executor{WorkPath: "/work"}
	tests := []struct {
		path string
		want string
	}{
		{"/abs/path", "/abs/path"},
		{"relative/path", "/work/relative/path"},
		{"file.go", "/work/file.go"},
		{"", "/work"},
	}
	for _, tt := range tests {
		got := getFullPath(e, tt.path)
		if got != tt.want {
			t.Errorf("getFullPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

// ---------- isExclude ----------

func TestIsExclude(t *testing.T) {
	e := &toolTypes.Executor{
		WorkPath: "/work",
		Exclude: []toolTypes.Exclude{
			{File: "node_modules", Negate: false},
			{File: ".git", Negate: false},
		},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"/work/node_modules/index.js", true},
		{"/work/src/main.go", false},
		{"/work/.git/config", true},
		{"/work/dist/app.js", false},
	}
	for _, tt := range tests {
		got := isExclude(e, tt.path)
		if got != tt.want {
			t.Errorf("isExclude(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsExclude_Negate(t *testing.T) {
	// negate: last matching rule wins
	e := &toolTypes.Executor{
		WorkPath: "/work",
		Exclude: []toolTypes.Exclude{
			{File: "vendor", Negate: false},
			{File: "vendor", Negate: true}, // un-exclude vendor
		},
	}
	if isExclude(e, "/work/vendor/lib.go") {
		t.Error("expected vendor to NOT be excluded after negate rule")
	}
}

// ---------- matchFiles ----------

func TestMatchFiles(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		parts    []string
		want     bool
	}{
		{"both empty", nil, nil, true},
		{"empty patterns non-empty parts", nil, []string{"a"}, false},
		{"simple match", []string{"*.go"}, []string{"main.go"}, true},
		{"simple no match", []string{"*.go"}, []string{"main.ts"}, false},
		{"multi-level match", []string{"internal", "*.go"}, []string{"internal", "tool.go"}, true},
		{"multi-level no match", []string{"internal", "*.go"}, []string{"cmd", "tool.go"}, false},
		{"glob star zero parts", []string{"**", "*.go"}, []string{"main.go"}, true},
		{"glob star one level", []string{"**", "*.go"}, []string{"internal", "main.go"}, true},
		{"glob star deep", []string{"**", "*.go"}, []string{"a", "b", "c", "main.go"}, true},
		{"glob star no match", []string{"**", "*.go"}, []string{"a", "b", "main.ts"}, false},
		{"patterns remaining no parts", []string{"*.go"}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchFiles(tt.patterns, tt.parts)
			if got != tt.want {
				t.Errorf("matchFiles(%v, %v) = %v, want %v", tt.patterns, tt.parts, got, tt.want)
			}
		})
	}
}

// ---------- write ----------

func TestWrite(t *testing.T) {
	e := newExec(t)

	t.Run("empty content", func(t *testing.T) {
		_, err := write(e, "test.txt", "")
		if err == nil {
			t.Fatal("expected error for empty content")
		}
	})

	t.Run("write new file", func(t *testing.T) {
		msg, err := write(e, "hello.txt", "hello world")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(msg, "hello.txt") {
			t.Errorf("message should contain filename, got: %s", msg)
		}
	})

	t.Run("create nested dirs", func(t *testing.T) {
		_, err := write(e, "a/b/c.txt", "nested content")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(e.WorkPath, "a/b/c.txt"))
		if string(data) != "nested content" {
			t.Errorf("content mismatch: %q", data)
		}
	})

	t.Run("overwrite existing", func(t *testing.T) {
		write(e, "over.txt", "original")
		write(e, "over.txt", "updated")
		got, _ := read(e, "over.txt")
		if got != "updated" {
			t.Errorf("expected 'updated', got %q", got)
		}
	})

	t.Run("MkdirAll fails when parent is a file", func(t *testing.T) {
		e2 := newExec(t)
		// create a regular file at "blocked"
		writeTemp(t, e2, "blocked", "i am a file")
		// now try to write to "blocked/child.txt" — MkdirAll("blocked") should fail
		_, err := write(e2, "blocked/child.txt", "content")
		if err == nil {
			t.Fatal("expected error when parent path is a file, not a dir")
		}
	})
}

// ---------- read ----------

func TestRead(t *testing.T) {
	e := newExec(t)

	t.Run("file not found", func(t *testing.T) {
		_, err := read(e, "nonexistent.txt")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("read existing file", func(t *testing.T) {
		writeTemp(t, e, "hello.txt", "hello world")
		got, err := read(e, "hello.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "hello world" {
			t.Errorf("read() = %q, want %q", got, "hello world")
		}
	})

	t.Run("absolute path", func(t *testing.T) {
		abs := filepath.Join(e.WorkPath, "abs.txt")
		os.WriteFile(abs, []byte("abs content"), 0644)
		got, err := read(e, abs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "abs content" {
			t.Errorf("read() = %q, want %q", got, "abs content")
		}
	})

	t.Run("excluded file", func(t *testing.T) {
		e2 := &toolTypes.Executor{
			WorkPath: e.WorkPath,
			Exclude:  []toolTypes.Exclude{{File: "secret.txt", Negate: false}},
		}
		writeTemp(t, e, "secret.txt", "secret")
		_, err := read(e2, "secret.txt")
		if err == nil {
			t.Fatal("expected error for excluded file")
		}
	})
}

// ---------- patch ----------

func TestPatch(t *testing.T) {
	e := newExec(t)

	t.Run("old_string not found", func(t *testing.T) {
		writeTemp(t, e, "file.txt", "hello world")
		_, err := patch(e, "file.txt", "missing", "replacement")
		if err == nil {
			t.Fatal("expected error when old_string not found")
		}
	})

	t.Run("successful patch", func(t *testing.T) {
		writeTemp(t, e, "patch.txt", "foo bar baz")
		_, err := patch(e, "patch.txt", "bar", "qux")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, _ := read(e, "patch.txt")
		if got != "foo qux baz" {
			t.Errorf("patch result = %q, want %q", got, "foo qux baz")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := patch(e, "missing.txt", "old", "new")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("excluded file", func(t *testing.T) {
		e2 := &toolTypes.Executor{
			WorkPath: e.WorkPath,
			Exclude:  []toolTypes.Exclude{{File: "locked.txt", Negate: false}},
		}
		writeTemp(t, e, "locked.txt", "data")
		_, err := patch(e2, "locked.txt", "data", "new")
		if err == nil {
			t.Fatal("expected error for excluded file")
		}
	})
}

// ---------- list ----------

func TestList(t *testing.T) {
	e := newExec(t)
	writeTemp(t, e, "a.txt", "a")
	writeTemp(t, e, "b.txt", "b")
	writeTemp(t, e, "sub/c.txt", "c")

	t.Run("non-recursive", func(t *testing.T) {
		got, err := list(e, "", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "a.txt") || !strings.Contains(got, "b.txt") {
			t.Errorf("expected a.txt and b.txt in listing, got:\n%s", got)
		}
		// sub should appear as directory
		if !strings.Contains(got, "sub/") {
			t.Errorf("expected sub/ directory in listing, got:\n%s", got)
		}
	})

	t.Run("recursive", func(t *testing.T) {
		got, err := list(e, "", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "sub/c.txt") {
			t.Errorf("expected sub/c.txt in recursive listing, got:\n%s", got)
		}
	})

	t.Run("invalid path", func(t *testing.T) {
		_, err := list(e, "nonexistent_dir", false)
		if err == nil {
			t.Fatal("expected error for nonexistent directory")
		}
	})

	t.Run("hidden dir skipped in recursive", func(t *testing.T) {
		e2 := newExec(t)
		writeTemp(t, e2, "visible.txt", "v")
		// create hidden dir with a file inside
		hiddenDir := filepath.Join(e2.WorkPath, ".hidden")
		os.MkdirAll(hiddenDir, 0755)
		os.WriteFile(filepath.Join(hiddenDir, "secret.txt"), []byte("s"), 0644)

		got, err := list(e2, "", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(got, "secret.txt") {
			t.Error("files inside hidden dirs should be skipped")
		}
	})

	t.Run("excluded dir skipped in recursive", func(t *testing.T) {
		e2 := &toolTypes.Executor{
			WorkPath: t.TempDir(),
			Exclude:  []toolTypes.Exclude{{File: "node_modules", Negate: false}},
		}
		writeTemp(t, e2, "main.go", "code")
		nmDir := filepath.Join(e2.WorkPath, "node_modules")
		os.MkdirAll(nmDir, 0755)
		os.WriteFile(filepath.Join(nmDir, "index.js"), []byte("pkg"), 0644)

		got, err := list(e2, "", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(got, "index.js") {
			t.Error("files inside excluded dirs should be skipped")
		}
	})
}

// ---------- glob ----------

func TestGlob(t *testing.T) {
	e := newExec(t)
	writeTemp(t, e, "main.go", "")
	writeTemp(t, e, "utils.go", "")
	writeTemp(t, e, "main_test.go", "")
	writeTemp(t, e, "sub/helper.go", "")

	t.Run("match all go files", func(t *testing.T) {
		got, err := glob(e, "*.go")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "main.go") {
			t.Errorf("expected main.go in result, got:\n%s", got)
		}
	})

	t.Run("match nested via glob star", func(t *testing.T) {
		got, err := glob(e, "**/*.go")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "sub/helper.go") {
			t.Errorf("expected sub/helper.go, got:\n%s", got)
		}
	})

	t.Run("no match", func(t *testing.T) {
		got, err := glob(e, "*.ts")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "No files found") {
			t.Errorf("expected no-match message, got: %s", got)
		}
	})
}

// ---------- search ----------

func TestSearch(t *testing.T) {
	e := newExec(t)
	writeTemp(t, e, "main.go", "package main\n\nfunc main() {}\n")
	writeTemp(t, e, "utils.go", "package main\n\nfunc helper() {}\n")

	t.Run("pattern found", func(t *testing.T) {
		got, err := search(e, "func", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "func") {
			t.Errorf("expected 'func' in results, got:\n%s", got)
		}
	})

	t.Run("pattern not found", func(t *testing.T) {
		got, err := search(e, "xyz_not_found_anywhere", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "No fils found") {
			t.Errorf("expected no-match message, got: %s", got)
		}
	})

	t.Run("with file pattern filter", func(t *testing.T) {
		got, err := search(e, "func", "*.go")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "func") {
			t.Errorf("expected 'func' in filtered results, got:\n%s", got)
		}
	})

	t.Run("invalid regex", func(t *testing.T) {
		_, err := search(e, "[invalid", "")
		if err == nil {
			t.Fatal("expected error for invalid regex")
		}
	})

	t.Run("binary file skipped", func(t *testing.T) {
		full := filepath.Join(e.WorkPath, "bin.exe")
		os.WriteFile(full, []byte("func main()"), 0644)
		got, err := search(e, "func", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(got, "bin.exe") {
			t.Error("binary .exe file should be skipped")
		}
	})

	t.Run("dot file skipped", func(t *testing.T) {
		full := filepath.Join(e.WorkPath, ".envrc")
		os.WriteFile(full, []byte("func secret()"), 0644)
		got, err := search(e, "secret", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(got, ".envrc") {
			t.Error("dot files should be skipped")
		}
	})
}

// ---------- extractSec ----------

func TestExtractSec(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantTs  int64
		wantStr string
	}{
		{
			name:    "no ts prefix",
			content: "hello world",
			wantTs:  0,
			wantStr: "hello world",
		},
		{
			name:    "valid ts prefix",
			content: "ts:1700000000\nbody text",
			wantTs:  1700000000,
			wantStr: "body text",
		},
		{
			name:    "ts prefix but no newline",
			content: "ts:1700000000",
			wantTs:  0,
			wantStr: "ts:1700000000",
		},
		{
			name:    "ts prefix with non-numeric value",
			content: "ts:not-a-number\nbody",
			wantTs:  0,
			wantStr: "ts:not-a-number\nbody",
		},
		{
			name:    "ts prefix with empty body",
			content: "ts:1700000000\n",
			wantTs:  1700000000,
			wantStr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTs, gotStr := extractSec(tt.content)
			if gotTs != tt.wantTs {
				t.Errorf("extractSec() ts = %d, want %d", gotTs, tt.wantTs)
			}
			if gotStr != tt.wantStr {
				t.Errorf("extractSec() str = %q, want %q", gotStr, tt.wantStr)
			}
		})
	}
}

// ---------- searchHistory early exits ----------

func TestSearchHistory_EarlyExit(t *testing.T) {
	t.Run("empty keyword", func(t *testing.T) {
		_, err := searchHistory("session-id", "", "")
		if err == nil {
			t.Fatal("expected error for empty keyword")
		}
		if !strings.Contains(err.Error(), "keyword is required") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty sessionID", func(t *testing.T) {
		_, err := searchHistory("", "keyword", "")
		if err == nil {
			t.Fatal("expected error for empty sessionID")
		}
		if !strings.Contains(err.Error(), "sessionID is required") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// ---------- ListExcludes / parseIgnore ----------

func TestListExcludes(t *testing.T) {
	dir := t.TempDir()

	// Write a .gitignore with a mix of rules
	ignoreContent := "# comment\nnode_modules\n!dist\n\n/vendor/\n"
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(ignoreContent), 0644)

	result := ListExcludes(dir)

	findFile := func(name string, negate bool) bool {
		for _, e := range result {
			if e.File == name && e.Negate == negate {
				return true
			}
		}
		return false
	}

	if !findFile("node_modules", false) {
		t.Error("expected node_modules in excludes")
	}
	if !findFile("dist", true) {
		t.Error("expected !dist (negate) in excludes")
	}
	if !findFile("vendor", false) {
		t.Error("expected vendor in excludes")
	}
}

func TestListExcludes_NoIgnoreFile(t *testing.T) {
	dir := t.TempDir()
	// No .gitignore — should still return defaults from embedded JSON without panic
	result := ListExcludes(dir)
	_ = result
}

func TestListExcludes_NonexistentDir(t *testing.T) {
	result := ListExcludes("/nonexistent/path/to/dir")
	_ = result // should not panic
}

// ---------- Routes ----------

func TestRoutes(t *testing.T) {
	e := newExec(t)
	writeTemp(t, e, "hello.txt", "hello world")

	t.Run("read_file", func(t *testing.T) {
		got, err := Routes(e, "read_file", []byte(`{"path":"hello.txt"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "hello world" {
			t.Errorf("got %q, want %q", got, "hello world")
		}
	})

	t.Run("write_file", func(t *testing.T) {
		_, err := Routes(e, "write_file", []byte(`{"path":"out.txt","content":"written"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(e.WorkPath, "out.txt"))
		if string(data) != "written" {
			t.Errorf("content = %q, want %q", data, "written")
		}
	})

	t.Run("list_files", func(t *testing.T) {
		got, err := Routes(e, "list_files", []byte(`{"path":"","recursive":false}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "hello.txt") {
			t.Errorf("expected hello.txt in listing, got: %s", got)
		}
	})

	t.Run("glob_files", func(t *testing.T) {
		got, err := Routes(e, "glob_files", []byte(`{"pattern":"*.txt"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "hello.txt") {
			t.Errorf("expected hello.txt in glob, got: %s", got)
		}
	})

	t.Run("search_content", func(t *testing.T) {
		got, err := Routes(e, "search_content", []byte(`{"pattern":"hello","file_pattern":""}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "hello") {
			t.Errorf("expected match, got: %s", got)
		}
	})

	t.Run("patch_edit", func(t *testing.T) {
		_, err := Routes(e, "patch_edit", []byte(`{"path":"hello.txt","old_string":"hello","new_string":"hi"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, _ := read(e, "hello.txt")
		if !strings.Contains(got, "hi") {
			t.Errorf("expected patched content, got: %s", got)
		}
	})

	t.Run("search_history empty keyword", func(t *testing.T) {
		_, err := Routes(e, "search_history", []byte(`{"keyword":"","time_range":""}`))
		if err == nil {
			t.Fatal("expected error for empty keyword in search_history")
		}
	})

	t.Run("unknown tool", func(t *testing.T) {
		_, err := Routes(e, "unknown_tool", []byte(`{}`))
		if err == nil {
			t.Fatal("expected error for unknown tool")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := Routes(e, "read_file", []byte(`{bad json`))
		if err == nil {
			t.Fatal("expected error for invalid json")
		}
	})
}

// ---------- walkFiles: excluded non-dir file ----------

func TestWalkFiles_ExcludedFile(t *testing.T) {
	e := &toolTypes.Executor{
		WorkPath: t.TempDir(),
		Exclude:  []toolTypes.Exclude{{File: "secret.txt", Negate: false}},
	}
	writeTemp(t, e, "public.txt", "public")
	writeTemp(t, e, "secret.txt", "secret")

	got, err := list(e, "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "secret.txt") {
		t.Error("excluded file should not appear in walkFiles output")
	}
	if !strings.Contains(got, "public.txt") {
		t.Error("public.txt should appear in walkFiles output")
	}
}

// ---------- search: invalid filePattern ----------

func TestSearch_InvalidFilePattern(t *testing.T) {
	e := newExec(t)
	writeTemp(t, e, "main.go", "func main() {}")

	// filepath.Match returns error for patterns like "[invalid"
	// search should warn and treat it as no-match (skips the file)
	got, err := search(e, "func", "[invalid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// file is skipped due to match error → no results
	if !strings.Contains(got, "No fils found") {
		t.Errorf("expected no-match message for invalid filePattern, got: %s", got)
	}
}

// ---------- searchHistory ----------

func setupHistoryFile(t *testing.T, sessionID string, entries []historyEntry) string {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	sessDir := filepath.Join(tmpHome, ".config", "agenvoy", "sessions", sessionID)
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessDir, "history.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	return tmpHome
}

func TestSearchHistory_FileNotFound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	got, err := searchHistory("nonexistent-session", "keyword", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "No history found") {
		t.Errorf("expected 'No history found', got: %s", got)
	}
}

func TestSearchHistory_BadJSON(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	sessDir := filepath.Join(tmpHome, ".config", "agenvoy", "sessions", "sess-bad")
	os.MkdirAll(sessDir, 0755)
	os.WriteFile(filepath.Join(sessDir, "history.json"), []byte("not valid json"), 0644)

	_, err := searchHistory("sess-bad", "keyword", "")
	if err == nil {
		t.Fatal("expected error for bad JSON history file")
	}
}

func TestSearchHistory_WithMatches(t *testing.T) {
	// searchHistory iterates from len(entries)-5 down to 0
	// Need ≥6 entries so index 0 and 1 are checked
	entries := []historyEntry{
		{Role: "user", Content: "first entry with target keyword"},
		{Role: "assistant", Content: "second entry also has target"},
		{Role: "user", Content: "third entry"},
		{Role: "user", Content: "fourth entry"},
		{Role: "user", Content: "fifth entry"},
		{Role: "user", Content: "sixth entry (last 4 skipped)"},
	}
	setupHistoryFile(t, "sess-match", entries)

	got, err := searchHistory("sess-match", "target", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "target") {
		t.Errorf("expected 'target' in matches, got: %s", got)
	}
}

func TestSearchHistory_NoMatches(t *testing.T) {
	entries := []historyEntry{
		{Role: "user", Content: "apple"},
		{Role: "assistant", Content: "banana"},
		{Role: "user", Content: "cherry"},
		{Role: "user", Content: "date"},
		{Role: "user", Content: "elderberry"},
		{Role: "user", Content: "fig (last 4 boundary)"},
	}
	setupHistoryFile(t, "sess-nomatch", entries)

	got, err := searchHistory("sess-nomatch", "xyzzy_not_here", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "No matches found") {
		t.Errorf("expected 'No matches found', got: %s", got)
	}
}

func TestSearchHistory_TimeRange(t *testing.T) {
	// ts:1000 is year 1970, well outside any "1d" window
	// Entry at index 0 and 1 contain keyword but are too old → filtered
	entries := []historyEntry{
		{Role: "user", Content: "ts:1000\ntarget keyword here"},
		{Role: "assistant", Content: "ts:1000\nalso target but old"},
		{Role: "user", Content: "third"},
		{Role: "user", Content: "fourth"},
		{Role: "user", Content: "fifth"},
		{Role: "user", Content: "sixth"},
	}
	setupHistoryFile(t, "sess-timerange", entries)

	// With "1d" range, entries with ts=1000 (year 1970) are filtered out
	got, err := searchHistory("sess-timerange", "target", "1d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "No matches found") {
		t.Errorf("expected no matches after time filtering old entries, got: %s", got)
	}
}

// ---------- Routes: list_files recursive + search_history with session ----------

func TestRoutes_ListFiles_Recursive(t *testing.T) {
	e := newExec(t)
	writeTemp(t, e, "sub/deep.txt", "deep content")

	got, err := Routes(e, "list_files", []byte(`{"path":"","recursive":true}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "sub/deep.txt") {
		t.Errorf("expected sub/deep.txt in recursive listing, got: %s", got)
	}
}

// ---------- Routes: invalid JSON for each remaining case ----------

func TestRoutes_InvalidJSON_AllCases(t *testing.T) {
	e := newExec(t)
	cases := []string{
		"write_file",
		"list_files",
		"glob_files",
		"search_content",
		"patch_edit",
		"search_history",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Routes(e, name, []byte(`{bad json`))
			if err == nil {
				t.Fatalf("expected json.Unmarshal error for %s with invalid JSON", name)
			}
		})
	}
}

// ---------- write: WriteFile fails (read-only parent dir) ----------

func TestWrite_WriteFileFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; permission restrictions do not apply")
	}
	e := newExec(t)
	subdir := filepath.Join(e.WorkPath, "ro")
	os.MkdirAll(subdir, 0755)
	os.Chmod(subdir, 0555)
	t.Cleanup(func() { os.Chmod(subdir, 0755) })

	_, err := write(e, "ro/out.txt", "content")
	if err == nil {
		t.Fatal("expected error writing to read-only directory")
	}
}

// ---------- ListExcludes: unreadable ignore file (parseIgnore error) ----------

func TestListExcludes_UnreadableIgnoreFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; chmod 000 does not restrict reads")
	}
	dir := t.TempDir()
	ignorePath := filepath.Join(dir, ".gitignore")
	os.WriteFile(ignorePath, []byte("node_modules\n"), 0644)
	os.Chmod(ignorePath, 0000)
	t.Cleanup(func() { os.Chmod(ignorePath, 0644) })

	// parseIgnore fails to open → silently returns nil; ListExcludes should not panic
	result := ListExcludes(dir)
	_ = result
}

func TestRoutes_SearchHistory_WithSession(t *testing.T) {
	entries := []historyEntry{
		{Role: "user", Content: "route target entry"},
		{Role: "assistant", Content: "route response"},
		{Role: "user", Content: "third"},
		{Role: "user", Content: "fourth"},
		{Role: "user", Content: "fifth"},
		{Role: "user", Content: "sixth"},
	}
	setupHistoryFile(t, "route-sess", entries)

	e := &toolTypes.Executor{
		WorkPath:  t.TempDir(),
		SessionID: "route-sess",
	}

	got, err := Routes(e, "search_history", []byte(`{"keyword":"route target","time_range":""}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "route target") {
		t.Errorf("expected 'route target' in results, got: %s", got)
	}
}
