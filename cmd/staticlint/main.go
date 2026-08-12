/*
Package main implements staticlint — a customized multichecker tool for static code analysis.

# Overview

staticlint integrates a comprehensive set of static analysis tools to maintain code quality,
detect potential bugs, enforce coding standards, and ensure safety practices across Go codebases.

# Launching multichecker

To run the multichecker on the current project:

	go run ./cmd/staticlint ./...

Or build the executable binary first and execute it:

	go build -o staticlint ./cmd/staticlint
	./staticlint ./...

# Included Analyzers

1. Standard Analyzers (golang.org/x/tools/go/analysis/passes):
  - appends: checks for missing values after append.
  - asmdecl: reports mismatches between assembly files and Go declarations.
  - assign: detects useless assignments.
  - atomic: checks for common mistakes using the sync/atomic package.
  - atomicalign: checks for non-64-bit-aligned atomic accesses.
  - bools: detects common mistakes involving boolean operators.
  - buildtag: checks that build tags are valid.
  - cgocall: detects dangerous join of C and Go pointers in cgo calls.
  - composite: checks for unkeyed composite literals.
  - copylock: detects locks passed by value.
  - directive: checks format of //go:building directives.
  - errorsas: checks that the second argument to errors.As is a pointer to a type implementing error.
  - framepointer: reports assembly code that clobbers frame pointers.
  - httpresponse: detects mistakes using HTTP responses.
  - ifaceassert: detects impossible interface-to-interface type assertions.
  - loopclosure: detects references to loop variables from within nested functions.
  - lostcancel: checks for failure to call a context cancelation function.
  - nilfunc: checks for useless comparisons between functions and nil.
  - nilness: inspects control flow to detect impossible nil comparisons.
  - printf: checks consistency of Printf format strings and arguments.
  - shadow: checks for shadowed variables.
  - shift: checks for shifts that exceed the width of an integer.
  - sigchanyzer: detects misuse of unbuffered signal channels.
  - slog: checks key-value pairs in slog calls.
  - stdmethods: checks for misspellings of standard interface methods.
  - stringintconv: checks for string(int) conversions.
  - structtag: checks struct field tags for well-formedness.
  - testinggoroutine: checks for calls to Fatal from non-test goroutines.
  - tests: checks for common mistaken usages of tests and examples.
  - timeformat: checks for invalid time format strings.
  - unmarshal: checks for passing non-pointer arguments to unmarshal.
  - unreachable: detects unreachable code.
  - unsafeptr: checks for invalid conversions of unsafe.Pointer to uintptr.
  - unusedresult: checks for unused results of calls to standard functions.
  - unusedwrite: checks for unused writes to variables.
  - usesgenerics: checks for usage of generic features.

2. Staticcheck SA Class Analyzers (honnef.co/go/tools/staticcheck):
  - All SA analyzers (SA1000 - SA9009): static code checks for correctness, security, and performance.

3. Staticcheck Non-SA Class Analyzers:
  - ST1000: checks package comments.
  - ST1005: checks error strings formatting.
  - S1000: checks for single-case select statements that can be simplified.
  - QF1001: checks for redundant if statements that can be simplified.

4. External Public Analyzers:
  - ineffassign (github.com/gordonklaus/ineffassign/pkg/ineffassign): detects ineffectual assignments in Go code.
  - bodyclose (github.com/timakin/bodyclose/passes/bodyclose): checks whether res.Body is closed correctly.

5. Custom Analyzer (osexit):
  - osexit: prohibits direct calls to os.Exit in the main function of the main package.
*/
package main

import (
	"github.com/gordonklaus/ineffassign/pkg/ineffassign"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"

	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	var mychecks []*analysis.Analyzer

	// 1. Standard passes from golang.org/x/tools/go/analysis/passes
	stdPasses := []*analysis.Analyzer{
		appends.Analyzer,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		atomicalign.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		framepointer.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		printf.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		unusedwrite.Analyzer,
		usesgenerics.Analyzer,
	}
	mychecks = append(mychecks, stdPasses...)

	// 2. All SA analyzers from staticcheck.io
	for _, v := range staticcheck.Analyzers {
		mychecks = append(mychecks, v.Analyzer)
	}

	// 3. One analyzer from other staticcheck classes (ST, S, QF)
	for _, v := range stylecheck.Analyzers {
		if v.Analyzer.Name == "ST1005" {
			mychecks = append(mychecks, v.Analyzer)
		}
	}
	for _, v := range simple.Analyzers {
		if v.Analyzer.Name == "S1000" {
			mychecks = append(mychecks, v.Analyzer)
		}
	}
	for _, v := range quickfix.Analyzers {
		if v.Analyzer.Name == "QF1001" {
			mychecks = append(mychecks, v.Analyzer)
		}
	}

	// 4. External public analyzers
	//may find forgotten  err = someFunc() without checking err
	mychecks = append(mychecks, ineffassign.Analyzer)
	//may find forgotten defer on body.CLose
	mychecks = append(mychecks, bodyclose.Analyzer)

	// 5. Custom analyzer
	mychecks = append(mychecks, OSExitAnalyzer)

	multichecker.Main(mychecks...)
}
