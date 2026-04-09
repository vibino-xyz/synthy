package file

import "golang.org/x/tools/go/packages"

func LoadPackages(repoPath string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedExportFile,
		Dir: repoPath,
	}

	return packages.Load(cfg, "./...")
}
