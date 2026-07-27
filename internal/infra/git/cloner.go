// Package git shallow-clones repositories for indexing, authenticating with a
// short-lived GitHub installation token.
package git

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Cloner struct{}

func NewCloner() *Cloner { return &Cloner{} }

// Clone shallow-clones cloneURL (an https git URL) at the given branch into a
// fresh temporary directory, authenticating with token. It returns the path to
// the clone and a cleanup func the caller must invoke when done. leafName (e.g.
// the repo name) becomes the clone's directory name so downstream tooling sees
// a sensible path.
func (c *Cloner) Clone(ctx context.Context, cloneURL, branch, leafName, token string) (string, func(), error) {
	parent, err := os.MkdirTemp("", "synthy-clone-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(parent) }

	dest := filepath.Join(parent, sanitizeLeaf(leafName))

	authURL, err := withToken(cloneURL, token)
	if err != nil {
		cleanup()
		return "", nil, err
	}

	args := []string{"clone", "--depth", "1", "--single-branch"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, authURL, dest)

	cmd := exec.CommandContext(ctx, "git", args...)
	// Never prompt for credentials; fail fast instead of hanging.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("git clone failed: %w (%s)", err, redact(stderr.String(), token))
	}

	return dest, cleanup, nil
}

// withToken injects the installation token as the GitHub App auth user.
func withToken(cloneURL, token string) (string, error) {
	u, err := url.Parse(cloneURL)
	if err != nil {
		return "", fmt.Errorf("invalid clone url: %w", err)
	}
	if token != "" {
		u.User = url.UserPassword("x-access-token", token)
	}
	return u.String(), nil
}

func redact(s, token string) string {
	if token == "" {
		return s
	}
	return strings.ReplaceAll(s, token, "***")
}

func sanitizeLeaf(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == "/" {
		return "repo"
	}
	return name
}
