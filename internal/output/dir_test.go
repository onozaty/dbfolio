package output

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeMarker(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, MarkerFile), []byte(markerContent), 0o644); err != nil {
		t.Fatal(err)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
}

func exists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return err == nil
}

func TestPrepare_CreatesMissingDirectoryWithMarker(t *testing.T) {
	// Arrange
	dir := filepath.Join(t.TempDir(), "nested", "out")

	// Act
	err := Prepare(dir, false)

	// Assert
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if !exists(t, filepath.Join(dir, MarkerFile)) {
		t.Error("marker file was not written")
	}
}

func TestPrepare_UsesEmptyDirectory(t *testing.T) {
	// Arrange
	dir := t.TempDir()

	// Act
	err := Prepare(dir, false)

	// Assert
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if !exists(t, filepath.Join(dir, MarkerFile)) {
		t.Error("marker file was not written")
	}
}

func TestPrepare_CleansPreviousOutputWhenMarkerExists(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeMarker(t, dir)
	touch(t, filepath.Join(dir, "tables", "dropped_table.html"))
	touch(t, filepath.Join(dir, "index.html"))

	// Act
	err := Prepare(dir, false)

	// Assert
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != MarkerFile {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("directory entries = %v, want only %s", names, MarkerFile)
	}
}

func TestPrepare_RejectsNonEmptyDirectoryWithoutMarker(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "README.md"))

	// Act
	err := Prepare(dir, false)

	// Assert
	if !errors.Is(err, ErrNotEmpty) {
		t.Errorf("Prepare() error = %v, want ErrNotEmpty", err)
	}
}

func TestPrepare_DoesNotWriteMarkerWhenRejected(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "README.md"))

	// Act
	_ = Prepare(dir, false)

	// Assert
	if exists(t, filepath.Join(dir, MarkerFile)) {
		t.Error("marker file was written into a rejected directory")
	}
}

func TestPrepare_KeepsExistingFilesWithForce(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	userFile := filepath.Join(dir, "README.md")
	touch(t, userFile)

	// Act
	err := Prepare(dir, true)

	// Assert
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if !exists(t, userFile) {
		t.Error("user's file was removed")
	}
}

func TestPrepare_DoesNotMarkUserDirectoryWithForce(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "README.md"))

	// Act
	err := Prepare(dir, true)

	// Assert
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if exists(t, filepath.Join(dir, MarkerFile)) {
		t.Error("marker was written into a user directory; a later build would clean it")
	}
}

func TestPrepare_RejectsRebuildWithoutForceAfterForcedWrite(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	userFile := filepath.Join(dir, "README.md")
	touch(t, userFile)
	if err := Prepare(dir, true); err != nil {
		t.Fatal(err)
	}
	touch(t, filepath.Join(dir, "index.html"))

	// Act
	err := Prepare(dir, false)

	// Assert
	if !errors.Is(err, ErrNotEmpty) || !exists(t, userFile) {
		t.Errorf("Prepare() error = %v, user file exists = %v; want ErrNotEmpty and file kept", err, exists(t, userFile))
	}
}

func TestPrepare_IgnoresInvalidMarkers(t *testing.T) {
	tests := []struct {
		name  string
		plant func(t *testing.T, dir string)
	}{
		{"directory", func(t *testing.T, dir string) {
			if err := os.Mkdir(filepath.Join(dir, MarkerFile), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"symlink to a valid marker", func(t *testing.T, dir string) {
			other := t.TempDir()
			writeMarker(t, other)
			symlink(t, filepath.Join(other, MarkerFile), filepath.Join(dir, MarkerFile))
		}},
		{"file with other content", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, MarkerFile), []byte("some other tool's config"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			userFile := filepath.Join(dir, "keep")
			touch(t, userFile)
			tt.plant(t, dir)

			// Act
			err := Prepare(dir, false)

			// Assert
			if !errors.Is(err, ErrNotEmpty) || !exists(t, userFile) {
				t.Errorf("Prepare() error = %v, user file exists = %v; want ErrNotEmpty and file kept", err, exists(t, userFile))
			}
		})
	}
}

func TestPrepare_RemovesSymlinksWithoutTouchingTargets(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeMarker(t, dir)
	outside := filepath.Join(t.TempDir(), "outside.txt")
	touch(t, outside)
	symlink(t, outside, filepath.Join(dir, "link.txt"))
	outsideDir := t.TempDir()
	touch(t, filepath.Join(outsideDir, "inner.txt"))
	symlink(t, outsideDir, filepath.Join(dir, "linkdir"))

	// Act
	err := Prepare(dir, false)

	// Assert
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if !exists(t, outside) || !exists(t, filepath.Join(outsideDir, "inner.txt")) {
		t.Error("symlink targets outside the output directory were removed")
	}
}

func TestPrepare_RefusesToCleanDirectoryContainingGit(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeMarker(t, dir)
	touch(t, filepath.Join(dir, ".git", "HEAD"))

	// Act
	err := Prepare(dir, false)

	// Assert
	if err == nil || !exists(t, filepath.Join(dir, ".git", "HEAD")) {
		t.Errorf("Prepare() error = %v; want refusal and .git kept", err)
	}
}

func TestPrepare_RefusesToCleanHomeDirectoryReachedThroughSymlink(t *testing.T) {
	// Arrange
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeMarker(t, home)
	userFile := filepath.Join(home, "x")
	touch(t, userFile)
	link := filepath.Join(t.TempDir(), "home-link")
	symlink(t, home, link)

	// Act
	err := Prepare(link, false)

	// Assert
	if err == nil || !exists(t, userFile) {
		t.Errorf("Prepare() error = %v; want refusal and home files kept", err)
	}
}

func TestPrepare_CanRetryAfterInterruptedClean(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeMarker(t, dir)
	locked := filepath.Join(dir, "locked")
	touch(t, filepath.Join(locked, "file.html"))
	// A read-only directory makes RemoveAll fail partway through.
	if err := os.Chmod(locked, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	if err := Prepare(dir, false); err == nil {
		t.Fatal("expected the first clean to fail")
	}
	if err := os.Chmod(locked, 0o755); err != nil {
		t.Fatal(err)
	}

	// Act
	err := Prepare(dir, false)

	// Assert
	if err != nil {
		t.Errorf("Prepare() retry error = %v, want success because the marker was kept", err)
	}
}

func TestCreateFile_ReplacesSymlinkWithoutWritingThroughIt(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "target.txt")
	if err := os.WriteFile(target, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "index.html")
	symlink(t, target, path)

	// Act
	f, err := CreateFile(path)
	if err == nil {
		_, err = f.WriteString("generated")
		_ = f.Close()
	}

	// Assert
	if err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}
	b, _ := os.ReadFile(target)
	if string(b) != "original" {
		t.Errorf("symlink target content = %q, want unchanged", b)
	}
}

func TestCreateFile_OverwritesExistingFile(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "index.html")
	if err := os.WriteFile(path, []byte("old content that is longer"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Act
	f, err := CreateFile(path)
	if err == nil {
		_, err = f.WriteString("new")
		_ = f.Close()
	}

	// Assert
	if err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "new" {
		t.Errorf("content = %q, want %q", b, "new")
	}
}

func TestPrepare_RejectsPathThatIsAFile(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "file")
	touch(t, path)

	// Act
	err := Prepare(path, true)

	// Assert
	if err == nil {
		t.Error("Prepare() error = nil, want error for a file path")
	}
}

func TestCreateDir_CreatesMissingDirectory(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "tables")

	// Act
	err := CreateDir(path)

	// Assert
	if err != nil {
		t.Fatalf("CreateDir() error = %v", err)
	}
	if info, statErr := os.Stat(path); statErr != nil || !info.IsDir() {
		t.Errorf("directory not created: %v", statErr)
	}
}

func TestCreateDir_AcceptsExistingDirectory(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "tables")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}

	// Act
	err := CreateDir(path)

	// Assert
	if err != nil {
		t.Errorf("CreateDir() error = %v", err)
	}
}

func TestCreateDir_RejectsSymlinkToDirectory(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "tables")
	symlink(t, t.TempDir(), path)

	// Act
	err := CreateDir(path)

	// Assert
	if err == nil {
		t.Error("CreateDir() error = nil, want error for symlink")
	}
}

func TestCreateDir_RejectsFile(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "tables")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Act
	err := CreateDir(path)

	// Assert
	if err == nil {
		t.Error("CreateDir() error = nil, want error for file")
	}
}
