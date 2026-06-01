package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nofuss-io/frontmatter/docs"
)

var skillTargets = map[string]string{
	"claude":  ".claude",
	"codex":   ".codex",
	"copilot": ".copilot",
	"gemini":  ".gemini",
}

func installSkill(target string) error {
	dir, ok := skillTargets[target]
	if !ok {
		supported := make([]string, 0, len(skillTargets))
		for k := range skillTargets {
			supported = append(supported, k)
		}
		sort.Strings(supported)
		return fmt.Errorf("unknown target %q; supported: %s", target, strings.Join(supported, ", "))
	}
	return installSkillTo(dir)
}

func installSkillTo(agentDir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	base := filepath.Join(home, agentDir, "skills", "fm")
	files := []struct {
		path string
		data []byte
	}{
		{filepath.Join(base, "SKILL.md"), docs.SkillMD},
		{filepath.Join(base, "docs", "manual.md"), docs.SkillManualMD},
	}
	for _, f := range files {
		if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(f.path, f.data, 0o644); err != nil {
			return err
		}
		fmt.Println(f.path)
	}
	return nil
}
