package pipeline

import "testing"

func TestProcessRepository(t *testing.T) {
	repoPath := "/Users/ratnesh/Desktop/Waldo/services/argo"

	err := ProcessRepository(repoPath)
	if err != nil {
		t.Fatalf("Failed to process repository: %v", err)
	}

	t.Log("Successfully processed repository")
}
