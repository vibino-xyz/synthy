package repository

import (
	"slices"
	"testing"

	contracts "github.com/vibino-xyz/protos/contracts/build"
)

func TestFromProtoReadsNestedRepository(t *testing.T) {
	var m IngestionMessage
	err := m.FromProto(&contracts.RepositoryEventMessage{
		Provider:  contracts.Provider_GITHUB,
		EventType: contracts.EventType_INCREMENTAL_INDEX,
		Repository: &contracts.Repository{
			Id:            919577984,
			FullName:      "Ratnesh2003/to-do-go",
			CloneUrl:      "https://github.com/Ratnesh2003/to-do-go.git",
			DefaultBranch: "main",
		},
		InstallationId: 150810960,
	})
	if err != nil {
		t.Fatalf("FromProto: %v", err)
	}

	if m.RepositoryID != 919577984 {
		t.Errorf("RepositoryID = %d", m.RepositoryID)
	}
	if m.RepoFullName != "Ratnesh2003/to-do-go" {
		t.Errorf("RepoFullName = %q", m.RepoFullName)
	}
	if m.CloneURL != "https://github.com/Ratnesh2003/to-do-go.git" {
		t.Errorf("CloneURL = %q", m.CloneURL)
	}
	if m.InstallationID != 150810960 {
		t.Errorf("InstallationID = %d", m.InstallationID)
	}
	if m.EventType != IngestionEventTypeIncrementalIndex {
		t.Errorf("EventType = %q", m.EventType)
	}
}

// A message without a repository must be rejected rather than silently indexing
// an empty clone url.
func TestFromProtoRejectsMissingRepository(t *testing.T) {
	var m IngestionMessage
	if err := m.FromProto(&contracts.RepositoryEventMessage{}); err == nil {
		t.Fatal("expected an error for a message with no repository")
	}
}

func TestNetFileChanges(t *testing.T) {
	tests := []struct {
		name        string
		commits     []*contracts.Commit
		head        *contracts.Commit
		wantChanged []string
		wantRemoved []string
	}{
		{
			name:        "single commit",
			commits:     []*contracts.Commit{{Added: []string{"a.go"}, Modified: []string{"b.go"}, Removed: []string{"c.go"}}},
			wantChanged: []string{"a.go", "b.go"},
			wantRemoved: []string{"c.go"},
		},
		{
			name: "later delete wins over earlier edit",
			commits: []*contracts.Commit{
				{Added: []string{"tmp.go"}},
				{Modified: []string{"tmp.go"}},
				{Removed: []string{"tmp.go"}},
			},
			wantRemoved: []string{"tmp.go"},
		},
		{
			name: "re-added after delete counts as changed",
			commits: []*contracts.Commit{
				{Removed: []string{"x.go"}},
				{Added: []string{"x.go"}},
			},
			wantChanged: []string{"x.go"},
		},
		{
			name:        "falls back to head commit",
			head:        &contracts.Commit{Modified: []string{"only.go"}},
			wantChanged: []string{"only.go"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			changed, removed := netFileChanges(&contracts.RepositoryEventMessage{
				Commits:    tc.commits,
				HeadCommit: tc.head,
			})
			if !slices.Equal(changed, tc.wantChanged) {
				t.Errorf("changed = %v, want %v", changed, tc.wantChanged)
			}
			if !slices.Equal(removed, tc.wantRemoved) {
				t.Errorf("removed = %v, want %v", removed, tc.wantRemoved)
			}
		})
	}
}
