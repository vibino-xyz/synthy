package imports

import (
	"golang.org/x/tools/go/packages"
)

func Build(pkgs []*packages.Package) []*ImportEdge {
	var edges []*ImportEdge

	for _, pkg := range pkgs {
		for _ = range pkg.Imports {
			edges = append(edges, &ImportEdge{
				// TODO
			})
		}
	}

	return edges
}