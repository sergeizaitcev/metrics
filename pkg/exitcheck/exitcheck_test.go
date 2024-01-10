package exitcheck_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/sergeizaitcev/metrics/pkg/exitcheck"
)

func TestExitCheck(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), exitcheck.Analyzer, "./...")
}
