package cli

import (
	"github.com/xpwu/go-mongodb/gen"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── NewBuildConfig ────────────────────────────────────────

func TestNewBuildConfig_Defaults(t *testing.T) {
	b := NewBuildConfig()
	if b == nil {
		t.Fatal("NewBuildConfig() returned nil")
	}
	if b.config == nil {
		t.Fatal("b.config is nil")
	}
	if b.outDirRaw != "." {
		t.Errorf("default outDirRaw = %q, want \".\"", b.outDirRaw)
	}
}

func TestNewBuildConfig_WithXoptOptions(t *testing.T) {
	b := NewBuildConfig()
	if b == nil {
		t.Fatal("NewBuildConfig() with opts returned nil")
	}
}

// ─── 链式调用 ────────────────────────────────────────────────

func TestBuildConfig_ChainOutDir(t *testing.T) {
	b := NewBuildConfig()
	ret := b.OutDir("./zgen")
	if ret != b {
		t.Error("OutDir should return receiver for chaining")
	}
	if b.outDirRaw != "./zgen" {
		t.Errorf("outDirRaw = %q, want ./zgen", b.outDirRaw)
	}
}

func TestBuildConfig_ChainTargetPkg(t *testing.T) {
	b := NewBuildConfig()
	ret := b.TargetPkg("myproject/models")
	if ret != b {
		t.Error("TargetPkg should return receiver")
	}
	if b.config.Pkg != "myproject/models" {
		t.Errorf("Pkg = %q, want myproject/models", b.config.Pkg)
	}
}

func TestBuildConfig_FullChain(t *testing.T) {
	b := NewBuildConfig().
		OutDir("$GOMOD/zgen").
		TargetPkg("myproject/models")

	if b.outDirRaw != "$GOMOD/zgen" {
		t.Errorf("outDirRaw = %q", b.outDirRaw)
	}
	if b.config.Pkg != "myproject/models" {
		t.Errorf("Pkg = %q", b.config.Pkg)
	}
}

// ─── AddMap 泛型校验 ────────────────────────────────────────

func TestBuildConfig_AddMap_Valid(t *testing.T) {
	b := NewBuildConfig()
	b.AddMap("int", "github.com/xpwu/go-mongodb/field.IntField",
		"github.com/xpwu/go-mongodb/field.NewIntField", true)
	if b.err != nil {
		t.Errorf("AddMap with valid args set err: %v", b.err)
	}
}

func TestBuildConfig_AddMap_GenericTypeIdent(t *testing.T) {
	b := NewBuildConfig()
	b.AddMap("MapType[int]", "github.com/xpwu/go-mongodb/field.IntField",
		"github.com/xpwu/go-mongodb/field.NewIntField", true)
	if b.err == nil {
		t.Error("expected err for generic TypeIdent, got nil")
	}
	if !strings.Contains(b.err.Error(), "generic") {
		t.Errorf("err = %q, should mention generic", b.err.Error())
	}
}

func TestBuildConfig_AddMap_GenericFieldType(t *testing.T) {
	b := NewBuildConfig()
	b.AddMap("int", "field.MapField[int]",
		"github.com/xpwu/go-mongodb/field.NewIntField", true)
	if b.err == nil {
		t.Error("expected err for generic FieldType, got nil")
	}
	if !strings.Contains(b.err.Error(), "generic") {
		t.Errorf("err = %q, should mention generic", b.err.Error())
	}
}

func TestBuildConfig_AddMap_GenericNewFunc(t *testing.T) {
	b := NewBuildConfig()
	b.AddMap("int", "field.IntField",
		"field.NewField[T]", true)
	if b.err == nil {
		t.Error("expected err for generic NewFunc, got nil")
	}
	if !strings.Contains(b.err.Error(), "generic") {
		t.Errorf("err = %q, should mention generic", b.err.Error())
	}
}

// ─── findGoModDir（委托 gen.FindGoModDir）──────────────────

func TestFindGoModDir_Found(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module myproject\n"), 0644); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(projectDir, "model")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	got := gen.FindGoModDir(subDir)
	if got != projectDir {
		t.Errorf("findGoModDir(%q) = %q, want %q", subDir, got, projectDir)
	}
}

func TestFindGoModDir_NotFound(t *testing.T) {
	tmp := t.TempDir()
	got := gen.FindGoModDir(tmp)
	if got != "" {
		t.Errorf("findGoModDir(%q) = %q, want empty", tmp, got)
	}
}

func TestFindGoModDir_AtRoot(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := gen.FindGoModDir(tmp)
	if got != tmp {
		t.Errorf("findGoModDir(%q) = %q, want %q", tmp, got, tmp)
	}
}

// ─── resolveOutDir ──────────────────────────────────────────

func TestResolveOutDir_RelativePath(t *testing.T) {
	anchor := "/home/user/project/model"
	got, err := resolveOutDir("./zgen", anchor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Clean(filepath.Join(anchor, "./zgen"))
	if got != want {
		t.Errorf("resolveOutDir(\"./zgen\", %q) = %q, want %q", anchor, got, want)
	}
}

func TestResolveOutDir_GOMODPlaceholder(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module myproject\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveOutDir("$GOMOD/zgen", projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(projectDir, "zgen")
	if got != want {
		t.Errorf("resolveOutDir(\"$GOMOD/zgen\") = %q, want %q", got, want)
	}
}

func TestResolveOutDir_GOMODNotFound(t *testing.T) {
	tmp := t.TempDir()
	_, err := resolveOutDir("$GOMOD/zgen", tmp)
	if err == nil {
		t.Error("expected error when go.mod not found, got nil")
	}
	if !strings.Contains(err.Error(), "go.mod not found") {
		t.Errorf("error = %q, should mention go.mod not found", err.Error())
	}
}

func TestResolveOutDir_AbsolutePathRejected(t *testing.T) {
	_, err := resolveOutDir("/home/user/zgen", "/some/anchor")
	if err == nil {
		t.Error("expected error for absolute disk path, got nil")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("error = %q, should mention not allowed", err.Error())
	}
}

func TestResolveOutDir_WindowsAbsolutePathRejected(t *testing.T) {
	_, err := resolveOutDir("C:/Users/foo/zgen", "/some/anchor")
	if err == nil {
		t.Error("expected error for Windows absolute path, got nil")
	}
}

func TestResolveOutDir_GOMODSubPath(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module myproject\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveOutDir("$GOMOD/model/fields", projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(projectDir, "model", "fields")
	if got != want {
		t.Errorf("resolveOutDir(\"$GOMOD/model/fields\") = %q, want %q", got, want)
	}
}

// ─── determineSrcDir ────────────────────────────────────────

func TestDetermineSrcDir_WithGOFILE(t *testing.T) {
	t.Setenv("GOFILE", "/tmp/some/path/model.go")
	dir := determineSrcDir()
	if dir != "/tmp/some/path" {
		t.Errorf("got %q, want /tmp/some/path", dir)
	}
}

func TestDetermineSrcDir_WithRelativeGOFILE(t *testing.T) {
	t.Setenv("GOFILE", "model.go")
	dir := determineSrcDir()
	if dir == "" {
		t.Fatal("returned empty")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("got %q, want absolute path", dir)
	}
}

func TestDetermineSrcDir_WithoutGOFILE(t *testing.T) {
	t.Setenv("GOFILE", "")
	dir := determineSrcDir()
	if dir == "" {
		t.Fatal("returned empty")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("got %q, want absolute path", dir)
	}
}

func TestDetermineSrcDir_ViaRuntimeCaller(t *testing.T) {
	dir := determineSrcDir()
	if dir == "" {
		t.Fatal("returned empty")
	}
	// determineSrcDir 会跳过 github.com/xpwu/go-mongodb/ 内部的调用帧，
	// 因此从 cli 包内调用时，不会通过 runtime.Caller 找到本包路径，
	// 而是走到 os.Getwd() 兜底。这里只验证返回了合法的绝对路径即可。
	if !filepath.IsAbs(dir) {
		t.Errorf("got %q, want absolute path", dir)
	}
}

// ─── stringSlice ────────────────────────────────────────────

func TestStringSlice_SetAndString(t *testing.T) {
	var s stringSlice
	if got := s.String(); got != "[]" {
		t.Errorf("empty String() = %s, want []", got)
	}
	for _, v := range []string{"a", "b", "c"} {
		if err := s.Set(v); err != nil {
			t.Fatalf("Set(%q) error: %v", v, err)
		}
	}
	if got := s.String(); got != "[a b c]" {
		t.Errorf("String() = %q, want %q", got, "[a b c]")
	}
	if len(s) != 3 || s[0] != "a" || s[1] != "b" || s[2] != "c" {
		t.Errorf("slice content = %v, want [a b c]", []string(s))
	}
}

func TestStringSlice_SetEmpty(t *testing.T) {
	var s stringSlice
	if err := s.Set(""); err != nil {
		t.Fatalf("Set(\"\") error: %v", err)
	}
	if len(s) != 1 || s[0] != "" {
		t.Errorf("Set(\"\") = %v, want [\"\"]", []string(s))
	}
}

// ─── Run 集成测试（使用临时目录）────────────────────────────

func TestRun_GenerateToRelativeOutDir(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module myproject\n"), 0644); err != nil {
		t.Fatal(err)
	}

	modelDir := filepath.Join(projectDir, "model")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}

	userGo := `package model

//go:generate gomongodbgen -out-dir ./zgen

type User struct {
	ID   string ` + "`bson:\"_id\"`" + `
	Name string ` + "`bson:\"name\"`" + `
	Age  int    ` + "`bson:\"age\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(modelDir, "user.go"), []byte(userGo), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	os.Chdir(modelDir)
	t.Setenv("GOFILE", "user.go")
	defer os.Chdir(oldWd)

	// 通过 API 显式设置 OutDir，测试 Run() 的路径解析
	b := NewBuildConfig().OutDir("./zgen")
	b.Run()

	generated := filepath.Join(modelDir, "zgen", "zUserField.go")
	if _, err := os.Stat(generated); err != nil {
		t.Errorf("expected %s to exist, got %v", generated, err)
		entries, _ := os.ReadDir(filepath.Join(modelDir, "zgen"))
		t.Logf("zgen/ contents: %+v", entries)
	}
}

func TestRun_GenerateWithGOMODOutDir(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module myproject\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmdDir := filepath.Join(projectDir, "cmd")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	modelDir := filepath.Join(projectDir, "model")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}

	userGo := `package model

//go:generate gomongodbgen

type User struct {
	ID   string ` + "`bson:\"_id\"`" + `
	Name string ` + "`bson:\"name\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(modelDir, "user.go"), []byte(userGo), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	os.Chdir(cmdDir)
	t.Setenv("GOFILE", "../model/user.go")
	defer os.Chdir(oldWd)

	b := NewBuildConfig().OutDir("$GOMOD/zgen")
	b.Run()

	generated := filepath.Join(projectDir, "zgen", "zUserField.go")
	if _, err := os.Stat(generated); err != nil {
		t.Errorf("expected %s to exist, got %v", generated, err)
		wrong := filepath.Join(cmdDir, "zgen", "zUserField.go")
		if _, err := os.Stat(wrong); err == nil {
			t.Errorf("file was incorrectly generated at %s", wrong)
		}
	}
}

func TestRun_NoStructFound(t *testing.T) {
	tmp := t.TempDir()
	emptyGo := `package empty
`
	if err := os.WriteFile(filepath.Join(tmp, "empty.go"), []byte(emptyGo), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	t.Setenv("GOFILE", "empty.go")
	defer os.Chdir(oldWd)

	b := NewBuildConfig()
	b.Run()
}

func TestRunFromArgs_ParseOutDir(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module myproject\n"), 0644); err != nil {
		t.Fatal(err)
	}

	modelDir := filepath.Join(projectDir, "model")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}

	userGo := `package model

//go:generate gomongodbgen -out-dir ./zgen

type User struct {
	ID   string ` + "`bson:\"_id\"`" + `
	Name string ` + "`bson:\"name\"`" + `
	Age  int    ` + "`bson:\"age\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(modelDir, "user.go"), []byte(userGo), 0644); err != nil {
		t.Fatal(err)
	}

	// 模拟命令行调用：go run mongodb_gen.go -out-dir ./zgen
	oldArgs := os.Args
	os.Args = []string{"gomongodbgen", "-out-dir", "./zgen"}
	defer func() { os.Args = oldArgs }()

	oldWd, _ := os.Getwd()
	os.Chdir(modelDir)
	t.Setenv("GOFILE", "user.go")
	defer os.Chdir(oldWd)

	b := NewBuildConfig()
	RunFromArgs(b)

	generated := filepath.Join(modelDir, "zgen", "zUserField.go")
	if _, err := os.Stat(generated); err != nil {
		t.Errorf("expected %s to exist, got %v", generated, err)
	}
}

func TestResolveOutDir_RelativePathWithParent(t *testing.T) {
	anchor := "/home/user/project/model"
	got, err := resolveOutDir("../zgen", anchor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Clean("/home/user/project/zgen")
	if got != want {
		t.Errorf("resolveOutDir(\"../zgen\", %q) = %q, want %q", anchor, got, want)
	}
}

func TestResolveOutDir_GOMODNotAtStart(t *testing.T) {
	_, err := resolveOutDir("./output/$GOMOD/zgen", "/some/anchor")
	if err == nil {
		t.Error("expected error when $GOMOD is not at start of path, got nil")
	}
	if !strings.Contains(err.Error(), "$GOMOD must appear at the beginning") {
		t.Errorf("error = %q, should mention $GOMOD must appear at the beginning", err.Error())
	}
}

func TestResolveOutDir_GOMODOnly(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module myproject\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveOutDir("$GOMOD", projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != projectDir {
		t.Errorf("resolveOutDir(\"$GOMOD\") = %q, want %q", got, projectDir)
	}
}

func TestResolveOutDir_GOMODLowerCase(t *testing.T) {
	_, err := resolveOutDir("$gomod/zgen", "/some/anchor")
	if err == nil {
		t.Error("expected error for $gomod, got nil")
	}
	if !strings.Contains(err.Error(), "did you mean $GOMOD") {
		t.Errorf("error = %q, should suggest $GOMOD", err.Error())
	}
}
