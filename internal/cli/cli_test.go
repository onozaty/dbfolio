package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onozaty/dbfolio/internal/output"
)

var sampleSchema = filepath.Join("..", "..", "testdata", "basic", "schema.json")

func execute(t *testing.T, version string, args ...string) (string, error) {
	t.Helper()
	return executeWithInput(t, version, "", args...)
}

// executeWithInput runs the command with stdin as its standard input.
func executeWithInput(t *testing.T, version, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := NewRootCmd(version)
	cmd.SetIn(strings.NewReader(stdin))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestBuild_GeneratesIndexAndMarker(t *testing.T) {
	// Arrange
	outDir := filepath.Join(t.TempDir(), "docs")

	// Act
	stdout, err := execute(t, "", "build", sampleSchema, "-o", outDir)

	// Assert
	if err != nil {
		t.Fatalf("build error = %v\n%s", err, stdout)
	}
	index, err := os.ReadFile(filepath.Join(outDir, "index.html"))
	if err != nil {
		t.Fatalf("index.html not generated: %v", err)
	}
	if !strings.Contains(string(index), "public.users") || !strings.Contains(string(index), "ユーザー情報") {
		t.Error("index.html does not list the sample tables")
	}
	if _, err := os.Stat(filepath.Join(outDir, output.MarkerFile)); err != nil {
		t.Errorf("marker file not generated: %v", err)
	}
}

func TestBuild_GeneratesTablePagesAndAssets(t *testing.T) {
	// Arrange
	outDir := filepath.Join(t.TempDir(), "docs")

	// Act
	_, err := execute(t, "", "build", sampleSchema, "-o", outDir)

	// Assert
	if err != nil {
		t.Fatalf("build error = %v", err)
	}
	for _, name := range []string{"relationships.html", "functions.html", "enums.html", "tables/public.users.html", "tables/public.active_users.html", "assets/app.css", "assets/search-index.js", "assets/search.js"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Errorf("%s not generated: %v", name, err)
		}
	}
}

func TestBuild_KeepsPreviousOutputWhenFileNamesCollide(t *testing.T) {
	// Arrange
	outDir := t.TempDir()
	if _, err := execute(t, "", "build", sampleSchema, "-o", outDir); err != nil {
		t.Fatal(err)
	}
	colliding := filepath.Join(t.TempDir(), "colliding.json")
	content := `{"tables": [{"name": "Users"}, {"name": "` + output.FileName("Users") + `"}]}`
	if err := os.WriteFile(colliding, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Act
	_, err := execute(t, "", "build", colliding, "-o", outDir)

	// Assert
	if err == nil {
		t.Fatal("build error = nil, want file name collision error")
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "tables", "public.users.html")); statErr != nil {
		t.Errorf("previous output was removed: %v", statErr)
	}
}

func TestBuild_AcceptsOutputFlagBeforeSchemaPath(t *testing.T) {
	// Arrange
	outDir := filepath.Join(t.TempDir(), "docs")

	// Act
	_, err := execute(t, "", "build", "-o", outDir, sampleSchema)

	// Assert
	if err != nil {
		t.Fatalf("build error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "index.html")); err != nil {
		t.Errorf("index.html not generated: %v", err)
	}
}

func TestBuild_UsesDefaultOutputDir(t *testing.T) {
	// Arrange
	abs, err := filepath.Abs(sampleSchema)
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())

	// Act
	_, err = execute(t, "", "build", abs)

	// Assert
	if err != nil {
		t.Fatalf("build error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(DefaultOutputDir, "index.html")); err != nil {
		t.Errorf("index.html not generated in %s: %v", DefaultOutputDir, err)
	}
}

func TestBuild_KeepsPreviousOutputWhenSchemaIsInvalid(t *testing.T) {
	// Arrange
	outDir := t.TempDir()
	if _, err := execute(t, "", "build", sampleSchema, "-o", outDir); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Act
	_, err := execute(t, "", "build", bad, "-o", outDir)

	// Assert
	if err == nil {
		t.Fatal("build error = nil, want parse error")
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "index.html")); statErr != nil {
		t.Errorf("previous output was removed: %v", statErr)
	}
}

func TestBuild_RejectsNonEmptyDirectoryWithoutForce(t *testing.T) {
	// Arrange
	outDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outDir, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Act
	_, err := execute(t, "", "build", sampleSchema, "-o", outDir)

	// Assert
	if !errors.Is(err, output.ErrNotEmpty) {
		t.Errorf("build error = %v, want ErrNotEmpty", err)
	}
}

func TestBuild_WritesIntoNonEmptyDirectoryWithForceAndKeepsFiles(t *testing.T) {
	// Arrange
	outDir := t.TempDir()
	userFile := filepath.Join(outDir, "README.md")
	if err := os.WriteFile(userFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Act
	_, err := execute(t, "", "build", sampleSchema, "-o", outDir, "--force")

	// Assert
	if err != nil {
		t.Fatalf("build error = %v", err)
	}
	_, idxErr := os.Stat(filepath.Join(outDir, "index.html"))
	_, userErr := os.Stat(userFile)
	if idxErr != nil || userErr != nil {
		t.Errorf("index.html err = %v, user file err = %v; want both to exist", idxErr, userErr)
	}
}

func TestBuild_DoesNotCleanUserFilesOnRebuildAfterForce(t *testing.T) {
	// Arrange
	outDir := t.TempDir()
	userFile := filepath.Join(outDir, "README.md")
	if err := os.WriteFile(userFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := execute(t, "", "build", sampleSchema, "-o", outDir, "--force"); err != nil {
		t.Fatal(err)
	}

	// Act
	_, err := execute(t, "", "build", sampleSchema, "-o", outDir)

	// Assert
	_, userErr := os.Stat(userFile)
	if !errors.Is(err, output.ErrNotEmpty) || userErr != nil {
		t.Errorf("build error = %v, user file err = %v; want ErrNotEmpty and file kept", err, userErr)
	}
}

func TestBuild_ReadsSchemaFromStdin(t *testing.T) {
	// Arrange
	content, err := os.ReadFile(sampleSchema)
	if err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(t.TempDir(), "docs")

	// Act
	stdout, err := executeWithInput(t, "", string(content), "build", "-", "-o", outDir)

	// Assert
	if err != nil {
		t.Fatalf("build error = %v\n%s", err, stdout)
	}
	index, err := os.ReadFile(filepath.Join(outDir, "index.html"))
	if err != nil {
		t.Fatalf("index.html not generated: %v", err)
	}
	if !strings.Contains(string(index), "public.users") {
		t.Error("index.html does not list the sample tables")
	}
}

func TestBuild_NamesStdinInParseError(t *testing.T) {
	// Arrange
	outDir := filepath.Join(t.TempDir(), "docs")

	// Act
	_, err := executeWithInput(t, "", "", "build", "-", "-o", outDir)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "standard input") {
		t.Errorf("build error = %v, want a parse error naming standard input", err)
	}
	if _, statErr := os.Stat(outDir); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("output directory created for invalid input: %v", statErr)
	}
}

func TestBuild_AddsCustomCSS(t *testing.T) {
	// Arrange
	outDir := filepath.Join(t.TempDir(), "docs")
	css := filepath.Join(t.TempDir(), "theme.css")
	if err := os.WriteFile(css, []byte(":root { --link: #8250df; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Act
	_, err := execute(t, "", "build", sampleSchema, "-o", outDir, "--css", css)

	// Assert
	if err != nil {
		t.Fatalf("build error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(outDir, "assets", "custom.css"))
	if err != nil || string(got) != ":root { --link: #8250df; }\n" {
		t.Errorf("assets/custom.css = %q, %v; want the given CSS", got, err)
	}
	// Loaded after app.css so that it overrides it, with paths relative to each page.
	for name, link := range map[string]string{
		"index.html":               `<link rel="stylesheet" href="assets/app.css">` + "\n" + `<link rel="stylesheet" href="assets/custom.css">`,
		"tables/public.users.html": `<link rel="stylesheet" href="../assets/app.css">` + "\n" + `<link rel="stylesheet" href="../assets/custom.css">`,
	} {
		page, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(page), link) {
			t.Errorf("%s does not load custom.css after app.css", name)
		}
	}
}

func TestBuild_OmitsCustomCSSByDefault(t *testing.T) {
	// Arrange
	outDir := filepath.Join(t.TempDir(), "docs")

	// Act
	_, err := execute(t, "", "build", sampleSchema, "-o", outDir)

	// Assert
	if err != nil {
		t.Fatalf("build error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "assets", "custom.css")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("assets/custom.css exists without --css: %v", err)
	}
	index, err := os.ReadFile(filepath.Join(outDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(index), "custom.css") {
		t.Error("index.html loads custom.css without --css")
	}
}

func TestBuild_KeepsPreviousOutputWhenCSSIsMissing(t *testing.T) {
	// Arrange
	outDir := t.TempDir()
	if _, err := execute(t, "", "build", sampleSchema, "-o", outDir); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "missing.css")

	// Act
	_, err := execute(t, "", "build", sampleSchema, "-o", outDir, "--css", missing)

	// Assert
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("build error = %v, want os.ErrNotExist", err)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "index.html")); statErr != nil {
		t.Errorf("previous output was removed: %v", statErr)
	}
}

func TestBuild_RequiresSchemaArgument(t *testing.T) {
	// Act
	_, err := execute(t, "", "build")

	// Assert
	if err == nil {
		t.Error("build error = nil, want argument error")
	}
}

func TestVersion_PrintsInjectedVersion(t *testing.T) {
	// Act
	stdout, err := execute(t, "1.2.3", "version")

	// Assert
	if err != nil {
		t.Fatalf("version error = %v", err)
	}
	if stdout != "dbfolio 1.2.3\n" {
		t.Errorf("stdout = %q, want %q", stdout, "dbfolio 1.2.3\n")
	}
}

func TestVersion_FallsBackWhenNotInjected(t *testing.T) {
	// Act
	stdout, err := execute(t, "", "version")

	// Assert
	if err != nil {
		t.Fatalf("version error = %v", err)
	}
	if stdout != "dbfolio dev\n" {
		t.Errorf("stdout = %q, want %q", stdout, "dbfolio dev\n")
	}
}
