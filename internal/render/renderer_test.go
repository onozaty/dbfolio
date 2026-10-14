package render

import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onozaty/dbfolio/internal/input/tbls"
	"github.com/onozaty/dbfolio/internal/output"
	"github.com/onozaty/dbfolio/internal/schema"
)

var update = flag.Bool("update", false, "update golden files")

func TestPlan_RejectsTablesMappedToSameFileName(t *testing.T) {
	// Arrange: "Users" gets a hash suffix, and another table is literally named like the result.
	s := &schema.Schema{Tables: []*schema.Table{{Name: "Users"}, {Name: output.FileName("Users")}}}

	// Act
	_, err := Plan(s)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "same file name") {
		t.Errorf("Plan() error = %v, want same file name error", err)
	}
}

func TestPlan_AcceptsNamesDifferingOnlyByCase(t *testing.T) {
	// Arrange
	s := &schema.Schema{Tables: []*schema.Table{{Name: "users"}, {Name: "Users"}}}

	// Act
	site, err := Plan(s)

	// Assert
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if site.files[s.Tables[0]] == site.files[s.Tables[1]] {
		t.Errorf("both tables map to %s", site.files[s.Tables[0]])
	}
}

// TestWrite_MatchesGolden compares the whole generated site with testdata/basic/expected.
// Run `go test ./internal/render -update` to regenerate the expected files.
func TestWrite_MatchesGolden(t *testing.T) {
	// Arrange
	base := filepath.Join("..", "..", "testdata", "basic")
	expected := filepath.Join(base, "expected")
	s, err := tbls.Load(filepath.Join(base, "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	site, err := Plan(s)
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	// Act
	err = site.Write(outDir)

	// Assert
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	got := readTree(t, outDir)
	if *update {
		if err := os.RemoveAll(expected); err != nil {
			t.Fatal(err)
		}
		for name, content := range got {
			path := filepath.Join(expected, name)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	want := readTree(t, expected)
	for name, content := range want {
		if got[name] != content {
			t.Errorf("%s differs from golden file (run with -update to regenerate)", name)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("%s is generated but has no golden file", name)
		}
	}
}

func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
