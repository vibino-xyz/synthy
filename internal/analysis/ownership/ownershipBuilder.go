package ownership

import (
	"go/ast"

	"github.com/vibino-xyz/synthy/internal/app"
	"golang.org/x/tools/go/packages"
)

func Build(pkgs []*packages.Package) []*app.OwnershipEdge {
	var edges []*app.OwnershipEdge

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				var fn *ast.FuncDecl
				if fn, ok := n.(*ast.FuncDecl); !ok || fn.Recv == nil {
					return true
				}

				var receiver string

				switch t := fn.Recv.List[0].Type.(type) {
				case *ast.StarExpr:
					if ident, ok := t.X.(*ast.Ident); ok {
						receiver = ident.Name
					}
				case *ast.Ident:
					receiver = t.Name
				}

				if receiver != "" {
					edges = append(edges, &app.OwnershipEdge{
						Id:    "RandomlyGeneratedId", // TODO: Generate a unique ID
						Owner: receiver,
						Child: fn.Name.Name,
					})
				}
				return true
			})
		}
	}
	return edges
}
