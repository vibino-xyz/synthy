package analysis

func ProcessRepository(repoPath string) error {
	// pkgs, err := parser.LoadPackages(repoPath)
	// if err != nil {
	// 	return err
	// }

	// symbols := symbol.Extract(pkgs)
	// for _, sym := range symbols {
	// 	slog.Info("Extracted symbol", "name", sym.Name, "kind", sym.Kind, "startLine", sym.StartLine, "endLine", sym.EndLine)
	// }

	// callEdges := calls.Build(pkgs)
	// for _, edge := range callEdges {
	// 	slog.Info("Extracted call edge", "caller", edge.Caller, "callee", edge.Callee)
	// }

	// importEdges := imports.Build(pkgs)
	// for _, edge := range importEdges {
	// 	slog.Info("Extracted import edge", "importer", edge.Importer, "importee", edge.Importee)
	// }

	// ownershipEdges := ownership.Build(pkgs)
	// for _, edge := range ownershipEdges {
	// 	slog.Info("Extracted ownership edge", "owner", edge.Owner, "child", edge.Child)
	// }

	// chunks, err := chunk.Build(repoPath)
	// if err != nil {
	// 	slog.Error("Failed to build chunks", "error", err)
	// 	return err
	// }
	// for _, c := range chunks {
	// 	slog.Info("Extracted chunk", "name", c.Name, "type", c.Type, "filePath", c.FilePath, "startLine", c.StartLine, "endLine", c.EndLine)
	// }

	return nil
}
