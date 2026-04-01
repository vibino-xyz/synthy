package chunk

import (
	"log/slog"
	"testing"
)

func TestBuild(t *testing.T) {
	filePath := "/Users/ratnesh/Desktop/Waldo/services/argo/controller/httpd/inboundPlanService.go"
	chunks, err := Build(filePath)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatalf("No chunks were built")
	}
	for _, chunk := range chunks {
		slog.Info("Chunks", "name", chunk.Name, "type", chunk.Type, "filePath", chunk.FilePath, "startLine", chunk.StartLine, "endLine", chunk.EndLine, "content", chunk.Content)
		slog.Info("--------------------------------------------------------")
	}

	t.Log("Successfully built chunks")
}
