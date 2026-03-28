package parser

import "testing"

func TestLoadPackages(t *testing.T) {
	repoPath := "/Users/ratnesh/Desktop/Waldo/services/argo"

	packages, err := LoadPackages(repoPath)
	if err != nil {
		t.Fatalf("Failed to load packages: %v", err)
	}

	if len(packages) == 0 {
		t.Fatal("No packages found")
	}

	t.Logf("Successfully loaded %d packages", len(packages))
}
