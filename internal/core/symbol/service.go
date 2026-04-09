package symbol

import (
	"go/ast"

	"golang.org/x/tools/go/packages"
)

func Extract(pkgs []*packages.Package) []*Symbol {
	var result []*Symbol

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {

			ast.Inspect(file, func(n ast.Node) bool {
				switch fn := n.(type) {
				case *ast.FuncDecl:
					symbol := &Symbol{
						// TODO
					}
					if fn.Recv != nil {
						// symbol.Kind = SymbolKindMethod TODO

						if len(fn.Recv.List) > 0 {
							switch recvType := fn.Recv.List[0].Type.(type) {
							case *ast.Ident:
								// symbol.ReceiverType = recvType.Name TODO
							case *ast.StarExpr:
								if _, ok := recvType.X.(*ast.Ident); ok {
									// symbol.ReceiverType = ident.Name TODO
								}
							}
						} else {
							// symbol.Kind = app.SymbolKindFunction TODO
						}

						result = append(result, symbol)
					}
				case *ast.TypeSpec:
					symbol := &Symbol{
						// TODO
					}
					result = append(result, symbol)
				}

				return true
			})
		}
	}

	return result
}
