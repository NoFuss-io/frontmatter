// Package integration runs table-driven, end-to-end tests against the built
// `fm` binary. Each subdirectory under `cases/` is one test case. See
// README.md for the layout.
package integration

import (
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite expected/ goldens from actual output")

// fmBin is the path to the compiled `fm` binary, populated by TestMain.
var fmBin string

func TestMain(m *testing.M) {
	flag.Parse()

	tmp, err := os.MkdirTemp("", "fm-integration-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	name := "fm"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	fmBin = filepath.Join(tmp, name)

	// Locate repo root: this file lives at test/integration/, so two levels up.
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	build := exec.Command("go", "build", "-o", fmBin, "./cmd/fm")
	build.Dir = repoRoot
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("build fm: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestCases(t *testing.T) {
	casesDir := "cases"
	entries, err := os.ReadDir(casesDir)
	if err != nil {
		t.Fatalf("read cases dir: %v", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		caseDir := filepath.Join(casesDir, e.Name())
		t.Run(e.Name(), func(t *testing.T) {
			runCase(t, caseDir)
		})
	}
}

func runCase(t *testing.T, caseDir string) {
	t.Helper()

	cmdBytes, err := os.ReadFile(filepath.Join(caseDir, "cmd"))
	if err != nil {
		t.Fatalf("read cmd: %v", err)
	}
	script := string(cmdBytes)

	work := t.TempDir()
	inputDir := filepath.Join(caseDir, "input")
	if _, err := os.Stat(inputDir); err == nil {
		if err := copyTree(inputDir, work); err != nil {
			t.Fatalf("copy input: %v", err)
		}
	}

	cmd := exec.Command("sh", "-ec", script)
	cmd.Dir = work
	cmd.Env = append(os.Environ(), "PATH="+filepath.Dir(fmBin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()

	expectedPath := filepath.Join(caseDir, "expected")
	checkGolden(t, expectedPath, stdout.String(), "stdout")

	// Optional stderr golden.
	if stderrPath := filepath.Join(caseDir, "expected_stderr"); fileExists(stderrPath) || *update && stderr.Len() > 0 {
		checkGolden(t, stderrPath, stderr.String(), "stderr")
	} else if stderr.Len() > 0 {
		t.Logf("stderr (unchecked):\n%s", stderr.String())
	}

	wantExit := 0
	if exitPath := filepath.Join(caseDir, "expected_exit"); fileExists(exitPath) {
		raw, err := os.ReadFile(exitPath)
		if err != nil {
			t.Fatalf("read expected_exit: %v", err)
		}
		wantExit, err = strconv.Atoi(strings.TrimSpace(string(raw)))
		if err != nil {
			t.Fatalf("parse expected_exit %q: %v", raw, err)
		}
	}
	gotExit := cmd.ProcessState.ExitCode()
	if gotExit != wantExit {
		t.Fatalf("exit code = %d, want %d\nstderr:\n%s", gotExit, wantExit, stderr.String())
	}

	// Optional post-run file goldens.
	expectedFiles := filepath.Join(caseDir, "expected_files")
	if _, err := os.Stat(expectedFiles); err == nil {
		checkFiles(t, expectedFiles, work)
	}
}

func checkGolden(t *testing.T, path, actual, label string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, []byte(actual), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s golden %s: %v (run with -update to create)", label, path, err)
	}
	if string(want) != actual {
		t.Fatalf("%s mismatch (%s)\n--- want ---\n%s\n--- got ---\n%s", label, path, want, actual)
	}
}

func checkFiles(t *testing.T, wantRoot, gotRoot string) {
	t.Helper()
	err := filepath.Walk(wantRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(wantRoot, path)
		gotPath := filepath.Join(gotRoot, rel)
		gotBytes, err := os.ReadFile(gotPath)
		if err != nil {
			t.Errorf("missing post-run file %s: %v", rel, err)
			return nil
		}
		if *update {
			return os.WriteFile(path, gotBytes, 0o644)
		}
		wantBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(wantBytes) != string(gotBytes) {
			t.Errorf("file %s differs\n--- want ---\n%s\n--- got ---\n%s", rel, wantBytes, gotBytes)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk expected_files: %v", err)
	}
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
