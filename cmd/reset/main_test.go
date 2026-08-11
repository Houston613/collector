package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResetGenerator(t *testing.T) {
	tempDir := t.TempDir()

	goMod := "module testpkg\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	code := `package testpkg

// generate:reset
type ParentStruct struct {
	IntVal    int
	StrVal    string
	BoolVal   bool
	FloatVal  float64
	IntPtr    *int
	StrPtr    *string
	SliceVal  []string
	MapVal    map[string]int
	ChildVal  ChildStruct
	ChildPtr  *ChildStruct
}

// generate:reset
type ChildStruct struct {
	Val int
}
`

	srcPath := filepath.Join(tempDir, "sample.go")
	if err := os.WriteFile(srcPath, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write sample file: %v", err)
	}

	if err := generateResetForDir(tempDir); err != nil {
		t.Fatalf("generateResetForDir failed: %v", err)
	}

	genPath := filepath.Join(tempDir, "reset.gen.go")
	content, err := os.ReadFile(genPath)
	if err != nil {
		t.Fatalf("failed to read generated reset.gen.go: %v", err)
	}

	genStr := string(content)

	expectedSnippets := []string{
		"package testpkg",
		"func (cs *ChildStruct) Reset()",
		"func (ps *ParentStruct) Reset()",
		"ps.IntVal = 0",
		"ps.StrVal = \"\"",
		"ps.BoolVal = false",
		"ps.FloatVal = 0",
		"if ps.IntPtr != nil {",
		"*ps.IntPtr = 0",
		"if ps.StrPtr != nil {",
		"*ps.StrPtr = \"\"",
		"ps.SliceVal = ps.SliceVal[:0]",
		"clear(ps.MapVal)",
		"if resetter, ok := any(&ps.ChildVal).(interface{ Reset() }); ok {",
		"resetter.Reset()",
		"if resetter, ok := any(ps.ChildPtr).(interface{ Reset() }); ok && ps.ChildPtr != nil {",
		"resetter.Reset()",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(genStr, snippet) {
			t.Errorf("expected generated code to contain %q, but got:\n%s", snippet, genStr)
		}
	}
}

func TestResetGeneratorCleanup(t *testing.T) {
	tempDir := t.TempDir()

	goMod := "module testpkg\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	genPath := filepath.Join(tempDir, "reset.gen.go")
	if err := os.WriteFile(genPath, []byte("// old code"), 0644); err != nil {
		t.Fatalf("failed to write dummy reset.gen.go: %v", err)
	}

	codeWithoutAnnotation := `package testpkg

type UnannotatedStruct struct {
	Val int
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "sample.go"), []byte(codeWithoutAnnotation), 0644); err != nil {
		t.Fatalf("failed to write sample file: %v", err)
	}

	if err := generateResetForDir(tempDir); err != nil {
		t.Fatalf("generateResetForDir failed: %v", err)
	}

	if _, err := os.Stat(genPath); !os.IsNotExist(err) {
		t.Errorf("expected reset.gen.go to be deleted when no structs have generate:reset, but it exists")
	}
}
