package app

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

var canonicalWriterAllowlist = map[string]bool{
	// Initial Project construction is not post-creation maintenance.
	"setup.go:saveClarification":           true,
	"liner.go:writeOperatingLayerLinerCmd": true,
	"liner.go:writeProjectSkillFile":       true,
	"liner.go:writeOperatingLayerMetadata": true,
	// Composition lineage and reusable skill authoring are separate artifact
	// managers. Their maintenance entry points are not exposed through the
	// Project/Source maintenance contract in this release.
	"composition_artifacts.go:writeCompositionNesting": true,
	"skills.go:acceptSkillDraft":                       true,
	"skills.go:acceptSkillGroundingDraft":              true,
	"skills.go:acceptSkillDeprecationDraft":            true,
	"skills.go:acceptSkillStateDraft":                  true,
	// This writer targets an os.MkdirTemp sandbox, never the user Project.
	"source_recovery.go:runDroppedCustomSourceRecovery": true,
}

// TestCanonicalWriterAudit is the repository check for the Go adapter boundary.
// The small allowlist contains first-run construction and an isolated temporary
// compile sandbox. Supported post-creation maintenance must go through Core.
func TestCanonicalWriterAudit(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(".", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, violation := range canonicalWriterViolations(name, body) {
			t.Error(violation)
		}
	}
}

func TestCanonicalWriterAuditCatchesIndirectAndAliasedWrites(t *testing.T) {
	illegal := []byte(`package sample
import filesystem "os"
func mutate(root string) error {
    linerPath := filepath.Join(root, "LINER.md")
    return filesystem.WriteFile(linerPath, []byte("unsafe"), 0o644)
}`)
	violations := canonicalWriterViolations("sample.go", illegal)
	if len(violations) != 1 || !strings.Contains(violations[0], "LINER.md") {
		t.Fatalf("expected variable-based aliased write violation, got %#v", violations)
	}
	safe := []byte(`package sample
import "os"
func draft(root string) error {
    draftPath := filepath.Join(root, "working", "review.md")
    return os.WriteFile(draftPath, []byte("safe"), 0o644)
}`)
	if violations := canonicalWriterViolations("sample.go", safe); len(violations) != 0 {
		t.Fatalf("working artifact should remain allowed, got %#v", violations)
	}
}

func canonicalWriterViolations(filename string, body []byte) []string {
	fileset := token.NewFileSet()
	file, err := parser.ParseFile(fileset, filename, body, 0)
	if err != nil {
		return []string{fmt.Sprintf("parse %s: %v", filename, err)}
	}
	osNames := importNames(file, "os")
	tapeNames := importNames(file, "github.com/cmdux/liner/packages/go-tui/internal/tape")
	violations := []string{}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		key := filename + ":" + function.Name.Name
		if canonicalWriterAllowlist[key] {
			continue
		}
		tainted := map[string]string{}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.AssignStmt:
				for index, left := range typed.Lhs {
					identifier, ok := left.(*ast.Ident)
					if !ok || len(typed.Rhs) == 0 {
						continue
					}
					right := typed.Rhs[min(index, len(typed.Rhs)-1)]
					if canonical := canonicalReference(right, tainted); canonical != "" {
						tainted[identifier.Name] = canonical
					}
				}
			case *ast.CallExpr:
				selector, ok := typed.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				owner, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}
				if tapeNames[owner.Name] && selector.Sel.Name == "WriteProject" {
					violations = append(violations, fmt.Sprintf("%s:%d %s directly calls tape.WriteProject", filename, fileset.Position(typed.Pos()).Line, function.Name.Name))
					return true
				}
				if !osNames[owner.Name] || len(typed.Args) == 0 || !filesystemMutation(selector.Sel.Name) {
					return true
				}
				if canonical := canonicalReference(typed.Args[0], tainted); canonical != "" {
					violations = append(violations, fmt.Sprintf("%s:%d %s directly mutates canonical %s via %s.%s", filename, fileset.Position(typed.Pos()).Line, function.Name.Name, canonical, owner.Name, selector.Sel.Name))
				}
			}
			return true
		})
	}
	return violations
}

func importNames(file *ast.File, importPath string) map[string]bool {
	names := map[string]bool{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != importPath {
			continue
		}
		name := filepath.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		names[name] = true
	}
	return names
}

func canonicalReference(expression ast.Expr, tainted map[string]string) string {
	canonical := ""
	ast.Inspect(expression, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.BasicLit:
			value, err := strconv.Unquote(typed.Value)
			if err == nil {
				for _, name := range []string{"LINER.md", "liner.yaml", "tape.yaml", "SKILL.md", "lineage.yaml"} {
					if value == name || strings.HasSuffix(value, "/"+name) {
						canonical = name
					}
				}
			}
		case *ast.Ident:
			if value := tainted[typed.Name]; value != "" {
				canonical = value
			} else if typed.Name == "skillPath" || typed.Name == "deprecatedPath" {
				canonical = "skills/*.md"
			} else if typed.Name == "lineagePath" {
				canonical = "lineage.yaml"
			}
		}
		return canonical == ""
	})
	return canonical
}

func filesystemMutation(name string) bool {
	switch name {
	case "WriteFile", "Remove", "RemoveAll", "Rename", "OpenFile", "Create":
		return true
	default:
		return false
	}
}

func TestLegacyCanonicalWriterRefusalNamesExactRemediation(t *testing.T) {
	for _, action := range []string{
		"apply a composition draft",
		"apply contradiction cleanup",
		"merge child Project state",
	} {
		message := legacyCoreWriterError(action).Error()
		if !strings.Contains(message, "Liner Core is the sole Project write authority") || !strings.Contains(message, coreWriterRemediation) {
			t.Fatalf("refusal for %q lacks exact remediation: %s", action, message)
		}
	}
}
