package main

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// OSExitAnalyzer prohibits direct calls to os.Exit in main function of main package.
var OSExitAnalyzer = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "prohibits direct call to os.Exit in main function of package main",
	Run:  runOSExit,
}

func runOSExit(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		// Only inspect main package files
		if file.Name.Name != "main" {
			continue
		}

		filename := pass.Fset.Position(file.Pos()).Filename
		if strings.Contains(filename, "go-build") || strings.Contains(filename, "go_build") {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}

			// Only inspect main function
			if fn.Name.Name != "main" {
				return true
			}

			if fn.Body == nil {
				return true
			}

			// Traversal inside main() body
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}

				if ident.Name == "os" && sel.Sel.Name == "Exit" {
					pass.Reportf(call.Pos(), "direct call to os.Exit in main function of main package is prohibited")
				}

				return true
			})

			return true
		})
	}

	return nil, nil
}
