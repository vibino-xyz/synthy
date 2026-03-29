package importgraph

import (
	"github.com/vibino-xyz/synthy/internal/app"
	"golang.org/x/tools/go/packages"
)

func Build(pkgs []*packages.Package) []*app.ImportEdge {
	var edges []*app.ImportEdge

	for _, pkg := range pkgs {
		for _, imp := range pkg.Imports {
			edges = append(edges, &app.ImportEdge{
				Id:       "RandomlyGeneratedId", // TODO: Generate a unique ID
				Importer: pkg.PkgPath,
				Importee: imp.PkgPath,
			})
		}
	}

	return edges
}
