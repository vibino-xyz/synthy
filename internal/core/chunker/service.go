package chunker

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
)

func Build(filePath string) ([]*CodeChunk, error) {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var chunks []*CodeChunk

	ast.Inspect(node, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			chunks = append(chunks, buildFunctionChunk(fset, filePath, decl))
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					switch t := s.Type.(type) {

					case *ast.StructType:
						chunks = append(chunks, buildStructChunk(fset, filePath, s, t))

					case *ast.InterfaceType:
						chunks = append(chunks, buildInterfaceChunk(fset, filePath, s, t))
					}
				}
			}
		}
		return true
	})

	if len(chunks) == 0 {
		chunks = append(chunks, buildFileChunk(fset, filePath, node))
	}

	return chunks, nil
}

func buildFunctionChunk(fset *token.FileSet, filePath string, fn *ast.FuncDecl) *CodeChunk {
	_ = fset.Position(fn.Pos()).Line
	_ = fset.Position(fn.End()).Line

	return &CodeChunk{
		// TODO
	}
}

func buildStructChunk(fset *token.FileSet, filePath string, ts *ast.TypeSpec, st *ast.StructType) *CodeChunk {
	_ = fset.Position(ts.Pos())
	_ = fset.Position(ts.End())

	return &CodeChunk{
		// TODO
	}
}

func buildInterfaceChunk(fset *token.FileSet, filePath string, ts *ast.TypeSpec, it *ast.InterfaceType) *CodeChunk {
	_ = fset.Position(ts.Pos())
	_ = fset.Position(ts.End())

	return &CodeChunk{
		// TODO
	}
}

func buildFileChunk(fset *token.FileSet, filePath string, node ast.Node) *CodeChunk {
	_ = fset.Position(node.Pos())
	_ = fset.Position(node.End())

	return &CodeChunk{
		// TODO
	}
}

func extractDependencies(node ast.Node) []string {
	dependencies := make(map[string]struct{})

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			dependencies[x.Name] = struct{}{}
		case *ast.SelectorExpr:
			if ident, ok := x.X.(*ast.Ident); ok {
				dependencies[fmt.Sprintf("%s.%s", ident.Name, x.Sel.Name)] = struct{}{}
			}
		}
		return true
	})

	var result []string
	for dependency := range dependencies {
		result = append(result, dependency)
	}

	return result
}

func extractSource(fset *token.FileSet, node ast.Node) string {
	if node == nil {
		return ""
	}

	start := fset.Position(node.Pos())
	end := fset.Position(node.End())

	content, err := os.ReadFile(start.Filename)
	if err != nil {
		slog.Error("failed to read file content", "error", err)
		return ""
	}

	startOffset := offsetFromPosition(content, start)
	endOffset := offsetFromPosition(content, end)

	if startOffset == -1 || endOffset == -1 || startOffset > endOffset {
		slog.Error("invalid offsets for source extraction", "startOffset", startOffset, "endOffset", endOffset)
		return ""
	}

	return string(content[startOffset:endOffset])
}

func offsetFromPosition(content []byte, pos token.Position) int {
	line := 1
	col := 1

	for i, b := range content {
		if line == pos.Line && col == pos.Column {
			return i
		}
		if b == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return -1
}
