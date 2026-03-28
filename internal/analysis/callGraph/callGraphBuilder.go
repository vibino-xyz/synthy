package callgraph

import (
	"go/ast"
	"go/types"

	"github.com/vibino-xyz/synthy/internal/app"
	"golang.org/x/tools/go/packages"
)

func Build(pkgs []*packages.Package) []*app.CallEdge {
	var edges []*app.CallEdge

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			var currentFunction *types.Func

			ast.Inspect(file, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.FuncDecl:
					if obj, ok := pkg.TypesInfo.Defs[node.Name]; ok {
						if fn, ok := obj.(*types.Func); ok {
							currentFunction = fn
						}
					}
				case *ast.CallExpr:
					if currentFunction == nil {
						return true
					}
					var callee *types.Func

					switch fun := node.Fun.(type) {
					case *ast.Ident:
						if obj, ok := pkg.TypesInfo.Uses[fun]; ok {
							if fn, ok := obj.(*types.Func); ok {
								callee = fn
							}
						}
					case *ast.SelectorExpr:
						if obj, ok := pkg.TypesInfo.Selections[fun]; ok {
							if fn, ok := obj.Obj().(*types.Func); ok {
								callee = fn
							}
						}
					}

					if callee != nil {
						edges = append(edges, &app.CallEdge{
							Id:     "RandomlyGeneratedId", // TODO: Generate a unique ID
							Caller: currentFunction.FullName(),
							Callee: callee.FullName(),
						})
					}
				}
				return true
			})
		}
	}

	return edges
}
