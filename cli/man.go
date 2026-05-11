package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func genManCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:    "gen-man [dir]",
		Short:  "Generate man page files",
		Hidden: true,
		Args:   cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			header := &doc.GenManHeader{
				Title:   "FM",
				Section: "1",
				Source:  "fm " + VERSION.String(),
			}
			return doc.GenManTree(root, header, dir)
		},
	}
}

func installManCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:    "install-man",
		Short:  "Install man page to system man path",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tmp, err := os.MkdirTemp("", "fm-man-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tmp)

			header := &doc.GenManHeader{
				Title:   "FM",
				Section: "1",
				Source:  "fm " + VERSION.String(),
			}
			if err := doc.GenManTree(root, header, tmp); err != nil {
				return err
			}

			manDir := findMan1Dir()
			if err := os.MkdirAll(manDir, 0755); err != nil {
				return fmt.Errorf("cannot create %s: %w", manDir, err)
			}

			data, err := os.ReadFile(filepath.Join(tmp, "fm.1"))
			if err != nil {
				return err
			}
			dst := filepath.Join(manDir, "fm.1")
			if err := os.WriteFile(dst, data, 0644); err != nil {
				return fmt.Errorf("cannot install to %s: %w", dst, err)
			}
			fmt.Println(dst)
			return nil
		},
	}
}

func findMan1Dir() string {
	if out, err := exec.Command("man", "--manpath").Output(); err == nil {
		for _, d := range strings.Split(strings.TrimSpace(string(out)), ":") {
			man1 := filepath.Join(d, "man1")
			if isWritableDir(man1) {
				return man1
			}
		}
	}
	return filepath.Join(os.Getenv("HOME"), ".local/share/man/man1")
}

func isWritableDir(dir string) bool {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}
	f, err := os.CreateTemp(dir, ".write-test-*")
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(f.Name())
	return true
}
