package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/printf"
	"honnef.co/go/tools/staticcheck"

	"github.com/sergeizaitcev/metrics/pkg/exitcheck"
)

var excludeStyleChecks = map[string]struct{}{
	"ST1000": {},
	"ST1020": {},
	"ST1021": {},
	"ST1022": {},
}

func main() {
	analyzers := []*analysis.Analyzer{
		exitcheck.Analyzer,
		assign.Analyzer,
		printf.Analyzer,
	}

	for _, v := range staticcheck.Analyzers {
		if _, ok := excludeStyleChecks[v.Analyzer.Name]; !ok {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	multichecker.Main(analyzers...)
}
