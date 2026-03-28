package symbol

import (
	"go/ast"

	"github.com/vibino-xyz/synthy/internal/app"
	"golang.org/x/tools/go/packages"
)

func Extract(pkgs []*packages.Package) []*app.Symbol {
	var result []*app.Symbol

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {

			ast.Inspect(file, func(n ast.Node) bool {
				switch fn := n.(type) {
				case *ast.FuncDecl:
					symbol := &app.Symbol{
						Id:        "RandomlyGeneratedId", // TODO: Generate a unique ID
						Name:      fn.Name.Name,
						StartLine: pkg.Fset.Position(fn.Pos()).Line,
						EndLine:   pkg.Fset.Position(fn.End()).Line,
					}
					if fn.Recv != nil {
						symbol.Kind = app.SymbolKindMethod

						if len(fn.Recv.List) > 0 {
							switch recvType := fn.Recv.List[0].Type.(type) {
							case *ast.Ident:
								symbol.ReceiverType = recvType.Name
							case *ast.StarExpr:
								if ident, ok := recvType.X.(*ast.Ident); ok {
									symbol.ReceiverType = ident.Name
								}
							}
						} else {
							symbol.Kind = app.SymbolKindFunction
						}

						result = append(result, symbol)
					}
				case *ast.TypeSpec:
					symbol := &app.Symbol{
						Id:        "RandomlyGeneratedId", // TODO: Generate a unique ID
						Name:      fn.Name.Name,
						StartLine: pkg.Fset.Position(fn.Pos()).Line,
						EndLine:   pkg.Fset.Position(fn.End()).Line,
						Kind:      app.SymbolKindType,
					}
					result = append(result, symbol)
				}

				return true
			})
		}
	}

	return result
}
