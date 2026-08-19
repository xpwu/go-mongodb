package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInferPackagePath_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	path, err := InferPackagePath(dir)

	if err == nil && path != "" {
		t.Fatalf("empty dir: expected error or empty path, got path=%q", path)
	}
	if err != nil && path != "" {
		t.Fatalf("empty dir: got error=%v but path=%q (should be empty)", err, path)
	}
}

func TestInferPackagePath_WithGoModAndGoFile(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/myproj\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	subDir := filepath.Join(dir, "subpkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "subpkg.go"),
		[]byte("package subpkg\n"), 0644); err != nil {
		t.Fatalf("write subpkg.go: %v", err)
	}

	path, err := InferPackagePath(subDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "example.com/myproj/subpkg" {
		t.Fatalf("got %q, want %q", path, "example.com/myproj/subpkg")
	}
}

func TestInferPackagePath_RootPackage(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/root\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "root.go"),
		[]byte("package root\n"), 0644); err != nil {
		t.Fatalf("write root.go: %v", err)
	}

	path, err := InferPackagePath(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "example.com/root" {
		t.Fatalf("got %q, want %q", path, "example.com/root")
	}
}

func TestInferPackagePath_DeepNested(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/deep\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	nested := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "c.go"),
		[]byte("package c\n"), 0644); err != nil {
		t.Fatalf("write c.go: %v", err)
	}

	path, err := InferPackagePath(nested)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "example.com/deep/a/b/c" {
		t.Fatalf("got %q, want %q", path, "example.com/deep/a/b/c")
	}
}

func TestInferPackagePath_NonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")

	path, err := InferPackagePath(dir)

	if err == nil && path != "" {
		t.Fatalf("non-existent dir: expected error or empty path, got path=%q", path)
	}
}

func TestInferPackagePath_MultipleGoFiles(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/multi\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, name := range []string{"a.go", "b.go", "c.go"} {
		if err := os.WriteFile(filepath.Join(dir, name),
			[]byte("package multi\n"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	path, err := InferPackagePath(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "example.com/multi" {
		t.Fatalf("got %q, want %q", path, "example.com/multi")
	}
}

func TestInferPackagePath_PackageNameMismatch(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/mismatch\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	subDir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "f.go"),
		[]byte("package differentname\n"), 0644); err != nil {
		t.Fatalf("write f.go: %v", err)
	}

	path, err := InferPackagePath(subDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(path, "example.com/mismatch") {
		t.Fatalf("path=%q should contain module name", path)
	}
	if !strings.Contains(path, "subdir") {
		t.Fatalf("path=%q should contain directory name subdir", path)
	}
}

func TestInferPackagePath_Idempotent(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/idem\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f.go"),
		[]byte("package idem\n"), 0644); err != nil {
		t.Fatalf("write f.go: %v", err)
	}

	path1, err1 := InferPackagePath(dir)
	if err1 != nil {
		t.Fatalf("first call error: %v", err1)
	}

	path2, err2 := InferPackagePath(dir)
	if err2 != nil {
		t.Fatalf("second call error: %v", err2)
	}
	if path1 != path2 {
		t.Fatalf("not idempotent: path1=%q path2=%q", path1, path2)
	}
}
