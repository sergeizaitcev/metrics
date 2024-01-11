package exitcheck

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Analyzer обнаруживает использование os.Exit в функции main.
var Analyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "detects the use os.Exit in the main func",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	expr := func(x *ast.SelectorExpr) bool {
		pkg, ok := x.X.(*ast.Ident)
		if ok && pkg.Name == "os" && x.Sel.Name == "Exit" {
			pass.Reportf(x.Pos(), "calling os.Exit in main")
			return false
		}
		return true
	}

	for _, file := range pass.Files {
		if file.Name.Name != "main" {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if x, ok := node.(*ast.SelectorExpr); ok {
				return expr(x)
			}
			return true
		})
	}

	return nil, nil
}
