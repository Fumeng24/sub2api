package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenAIProductionCallsContextAwareSSEToJSON(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob service files: %v", err)
	}

	delegateCalls := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "handleSSEToJSON" {
				return true
			}
			position := fset.Position(call.Pos())
			if filepath.Base(path) == "openai_sse_json_custom.go" {
				delegateCalls++
				return true
			}
			t.Errorf("%s:%d must call handleSSEToJSONWithContext so request context and account policy are preserved", path, position.Line)
			return true
		})
	}
	if delegateCalls != 1 {
		t.Fatalf("openai_sse_json_custom.go must delegate exactly once to the upstream handleSSEToJSON implementation; got %d calls", delegateCalls)
	}
}
