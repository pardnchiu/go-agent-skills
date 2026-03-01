package skill

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------- Scanner ----------

func TestNewScanner(t *testing.T) {
	s := NewScanner()
	if s == nil {
		t.Fatal("NewScanner() returned nil")
	}
	if s.Skills == nil {
		t.Fatal("Skills should not be nil after NewScanner")
	}
	// List() must not panic regardless of what paths exist on this machine
	_ = s.List()
}

func TestScanner_Scan(t *testing.T) {
	dir := t.TempDir()

	// valid skill: sub-dir with SKILL.md and frontmatter
	skillDir := filepath.Join(dir, "my-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: my-skill\ndescription: A test skill\n---\n# Usage\nDo stuff."),
		0644)

	// hidden dir must be ignored
	hiddenDir := filepath.Join(dir, ".hidden")
	os.MkdirAll(hiddenDir, 0755)
	os.WriteFile(filepath.Join(hiddenDir, "SKILL.md"), []byte("---\nname: hidden\n---\nbody"), 0644)

	// file (not dir) must be ignored
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: top-level\n---\nbody"), 0644)

	// skill dir without SKILL.md must be ignored
	emptyDir := filepath.Join(dir, "no-skill")
	os.MkdirAll(emptyDir, 0755)

	s := &Scanner{paths: []string{dir}}
	s.Scan()

	names := s.List()
	found := false
	for _, n := range names {
		if n == "my-skill" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'my-skill' in List(), got: %v", names)
	}

	// hidden and top-level file should NOT appear
	for _, n := range names {
		if n == "hidden" {
			t.Error("hidden dir should not be scanned")
		}
	}
}

func TestScanner_DuplicateSkillName(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	for _, d := range []string{dir1, dir2} {
		sd := filepath.Join(d, "dup-skill")
		os.MkdirAll(sd, 0755)
		os.WriteFile(filepath.Join(sd, "SKILL.md"),
			[]byte("---\nname: dup-skill\n---\nbody"),
			0644)
	}

	s := &Scanner{paths: []string{dir1, dir2}}
	s.Scan()

	count := 0
	for _, n := range s.List() {
		if n == "dup-skill" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("duplicate skill should appear only once, got %d", count)
	}
}

func TestScanner_NonexistentPath(t *testing.T) {
	s := &Scanner{paths: []string{"/nonexistent/path/that/does/not/exist"}}
	s.Scan() // should not panic or error
	if s.Skills == nil {
		t.Fatal("Skills should not be nil even with nonexistent paths")
	}
}

// ---------- extractHeader ----------

func TestExtractHeader(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantErr  bool
		wantName string // something we expect in the frontmatter
		wantBody string
	}{
		{
			name: "valid header and body",
			content: `---
name: my-skill
description: does things
---
# Body content
Some text here`,
			wantErr:  false,
			wantName: "name: my-skill",
			wantBody: "# Body content\nSome text here",
		},
		{
			name:    "no header",
			content: "# Just a body\nNo frontmatter",
			wantErr: true,
		},
		{
			name: "header only no body",
			content: `---
name: empty-body
---
`,
			wantErr:  false,
			wantName: "name: empty-body",
			wantBody: "",
		},
		{
			name:    "empty content",
			content: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, err := extractHeader([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Fatalf("extractHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if tt.wantName != "" && !containsStr(string(fm), tt.wantName) {
				t.Errorf("frontmatter = %q, want to contain %q", fm, tt.wantName)
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

// ---------- scan: ReadDir error + parser error ----------

func TestScanner_ReadDirError(t *testing.T) {
	// Passing a regular file as scan root causes os.ReadDir to fail.
	// The error is sent to errChan; Scan should not panic.
	tmpFile, err := os.CreateTemp("", "not-a-dir-*")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	s := &Scanner{paths: []string{tmpFile.Name()}}
	s.Scan() // must not panic

	if s.Skills == nil {
		t.Fatal("Skills must not be nil even when ReadDir fails")
	}
}

func TestScanner_ParseError_UnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; chmod 000 does not restrict reads")
	}

	dir := t.TempDir()
	skillDir := filepath.Join(dir, "broken-skill")
	os.MkdirAll(skillDir, 0755)
	path := filepath.Join(skillDir, "SKILL.md")
	os.WriteFile(path, []byte("content"), 0600)
	os.Chmod(path, 0000) // make unreadable → parser returns error
	t.Cleanup(func() { os.Chmod(path, 0600) })

	s := &Scanner{paths: []string{dir}}
	s.Scan() // should warn and skip, not panic

	for _, n := range s.List() {
		if n == "broken-skill" {
			t.Error("unreadable skill should be skipped, not registered")
		}
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

// ---------- parser ----------

func TestParser(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		_, err := parser("/nonexistent/path/SKILL.md")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("file with valid header", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, "my-skill")
		os.MkdirAll(skillDir, 0755)
		path := filepath.Join(skillDir, "SKILL.md")

		content := "---\nname: my-skill\ndescription: A test skill\n---\n# Usage\nDo stuff."
		os.WriteFile(path, []byte(content), 0644)

		skill, err := parser(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if skill.Name != "my-skill" {
			t.Errorf("Name = %q, want %q", skill.Name, "my-skill")
		}
		if skill.Description != "A test skill" {
			t.Errorf("Description = %q, want %q", skill.Description, "A test skill")
		}
		if skill.Body != "# Usage\nDo stuff." {
			t.Errorf("Body = %q, want %q", skill.Body, "# Usage\nDo stuff.")
		}
		if skill.Hash == "" {
			t.Error("Hash should not be empty")
		}
		if skill.AbsPath == "" {
			t.Error("AbsPath should not be empty")
		}
	})

	t.Run("file without header uses dir name", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, "fallback-skill")
		os.MkdirAll(skillDir, 0755)
		path := filepath.Join(skillDir, "SKILL.md")

		os.WriteFile(path, []byte("# Just a skill without frontmatter"), 0644)

		skill, err := parser(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Name falls back to directory name
		if skill.Name != "fallback-skill" {
			t.Errorf("Name = %q, want %q", skill.Name, "fallback-skill")
		}
		if skill.Description != "" {
			t.Errorf("Description should be empty, got %q", skill.Description)
		}
	})

	t.Run("file with name override only", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, "dir-name")
		os.MkdirAll(skillDir, 0755)
		path := filepath.Join(skillDir, "SKILL.md")

		os.WriteFile(path, []byte("---\nname: override-name\n---\nbody"), 0644)

		skill, err := parser(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if skill.Name != "override-name" {
			t.Errorf("Name = %q, want %q", skill.Name, "override-name")
		}
	})
}
