package skill

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

type Scanner struct {
	paths  []string
	Skills *SkillList
	mu     sync.RWMutex
}

type Skill struct {
	Name        string
	Description string
	AbsPath     string
	Path        string
	Content     string
	Body        string
	Hash        string
}

type SkillList struct {
	ByName map[string]*Skill
	ByPath map[string]*Skill
	Paths  []string
}

func NewScanner() *Scanner {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	cwd, _ := os.Getwd()
	paths := []string{
		filepath.Join(cwd, ".claude", "skills"),
		filepath.Join(cwd, ".skills"),
		// Claude
		filepath.Join(home, ".claude", "skills"),
		// OpenCode
		filepath.Join(home, ".opencode", "skills"),
		// OpenAI / Codex
		filepath.Join(home, ".openai", "skills"),
		filepath.Join(home, ".codex", "skills"),
		"/mnt/skills/public",
		"/mnt/skills/user",
		"/mnt/skills/examples",
	}

	scanner := &Scanner{
		paths: paths,
	}
	scanner.Scan()

	return scanner
}

func (s *Scanner) Scan() {
	list := &SkillList{
		ByName: make(map[string]*Skill),
		ByPath: make(map[string]*Skill),
		Paths:  s.paths,
	}

	// * concurrent scan path list
	var wg sync.WaitGroup
	skillChan := make(chan *Skill, 100)
	errChan := make(chan error, len(s.paths))
	for _, path := range s.paths {
		wg.Add(1)

		go func(dir string) {
			defer wg.Done()
			if err := s.scan(dir, skillChan); err != nil {
				errChan <- fmt.Errorf("s.scan %s: %w", dir, err)
			}
		}(path)
	}

	go func() {
		wg.Wait()
		close(skillChan)
		close(errChan)
	}()

	for skill := range skillChan {
		if _, ok := list.ByName[skill.Name]; ok {
			continue
		}
		list.ByName[skill.Name] = skill
		list.ByPath[skill.AbsPath] = skill
	}

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
		slog.Warn("scan error",
			slog.String("error", err.Error()))
	}

	s.mu.Lock()
	s.Skills = list
	s.mu.Unlock()
}

func (s *Scanner) scan(root string, skillChan chan<- *Skill) error {
	// * path not exists
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.IsDir() || e.Name()[0] == '.' {
			continue
		}

		// ~/.claude/skills/
		// └── {skill_name}/
		//     └── SKILL.md
		path := filepath.Join(root, e.Name(), "SKILL.md")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		skill, err := parser(path)
		if err != nil {
			slog.Warn("failed to parse skill",
				slog.String("path", path),
				slog.String("error", err.Error()))
			continue
		}
		skillChan <- skill
	}

	return nil
}

func (s *Scanner) List() []string {
	names := make([]string, 0, len(s.Skills.ByName))
	for name := range s.Skills.ByName {
		names = append(names, name)
	}
	return names
}
