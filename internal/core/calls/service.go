package calls

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

func Build(pkgs []*packages.Package) []*CallEdge {
	var edges []*CallEdge

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
						edges = append(edges, &CallEdge{
							// TODO
						})
					}
				}
				return true
			})
		}
	}

	return edges
}
