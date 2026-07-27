package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v89/github"
)

const (
	perPage     = 100
	maxPages    = 20 // safety bound: up to 2000 items
	httpTimeout = 15 * time.Second
)

// Repo is a repository visible to an installation, trimmed to what the
// dashboard needs.
type Repo struct {
	ExternalID    int64   `json:"external_id"`
	FullName      string  `json:"full_name"`
	Name          string  `json:"name"`
	Owner         string  `json:"owner"`
	Description   *string `json:"description"`
	Private       bool    `json:"private"`
	DefaultBranch string  `json:"default_branch"`
	Language      *string `json:"language"`
	HTMLURL       string  `json:"html_url"`
	CloneURL      string  `json:"clone_url"`
	PushedAt      string  `json:"pushed_at"`
}

// InstallationAccount is the org/user the App was installed on.
type InstallationAccount struct {
	Login string
	Type  string
}

// Client wraps the go-github SDK. Authentication is handled by ghinstallation:
// the AppsTransport signs the App JWT, and a per-installation Transport mints
// and refreshes installation tokens — so this type carries no token-handling
// code of its own.
type Client struct {
	cfg           Config
	ok            bool
	appsTransport *ghinstallation.AppsTransport
	appClient     *github.Client // authenticated as the App (JWT)

	mu             sync.Mutex
	installClients map[int64]*github.Client // per-installation, tokens cached inside ghinstallation
}

func NewClient(cfg Config) *Client {
	c := &Client{cfg: cfg, installClients: make(map[int64]*github.Client)}
	if !cfg.Configured {
		return c
	}

	appID, err := strconv.ParseInt(cfg.AppID, 10, 64)
	if err != nil {
		return c // leaves ok=false → Configured() reports not set up
	}

	atr := ghinstallation.NewAppsTransportFromPrivateKey(http.DefaultTransport, appID, cfg.PrivateKey)
	appClient, err := github.NewClient(
		github.WithHTTPClient(&http.Client{Transport: atr, Timeout: httpTimeout}),
	)
	if err != nil {
		return c
	}

	c.appsTransport = atr
	c.appClient = appClient
	c.ok = true
	return c
}

func (c *Client) Configured() bool { return c.ok }

// InstallationToken returns a short-lived installation access token, used to
// authenticate git clones of the installation's repositories.
func (c *Client) InstallationToken(ctx context.Context, installationID int64) (string, error) {
	if !c.ok {
		return "", ErrNotConfigured
	}
	return ghinstallation.NewFromAppsTransport(c.appsTransport, installationID).Token(ctx)
}

// installationClient returns a go-github client authenticated as the given
// installation, reusing it (and its cached token) across calls.
func (c *Client) installationClient(installationID int64) (*github.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cl, ok := c.installClients[installationID]; ok {
		return cl, nil
	}

	itr := ghinstallation.NewFromAppsTransport(c.appsTransport, installationID)
	cl, err := github.NewClient(
		github.WithHTTPClient(&http.Client{Transport: itr, Timeout: httpTimeout}),
	)
	if err != nil {
		return nil, err
	}
	c.installClients[installationID] = cl
	return cl, nil
}

// GetInstallationAccount fetches which account the installation belongs to and
// doubles as a validity check that the installation exists.
func (c *Client) GetInstallationAccount(ctx context.Context, installationID int64) (*InstallationAccount, error) {
	if !c.ok {
		return nil, ErrNotConfigured
	}
	inst, resp, err := c.appClient.Apps.GetInstallation(ctx, installationID)
	if err != nil {
		return nil, toAPIError(resp, err)
	}
	account := inst.GetAccount()
	return &InstallationAccount{Login: account.GetLogin(), Type: account.GetType()}, nil
}

// ListRepositories returns every repository the installation can access.
func (c *Client) ListRepositories(ctx context.Context, installationID int64) ([]Repo, error) {
	if !c.ok {
		return nil, ErrNotConfigured
	}
	cl, err := c.installationClient(installationID)
	if err != nil {
		return nil, err
	}

	opts := &github.ListOptions{PerPage: perPage}
	var repos []Repo
	for page := 0; page < maxPages; page++ {
		result, resp, err := cl.Apps.ListRepos(ctx, opts)
		if err != nil {
			return nil, toAPIError(resp, err)
		}
		for _, r := range result.Repositories {
			repos = append(repos, toRepo(r))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return repos, nil
}

// ListBranches returns the branch names of a repository the installation can
// access. repoFullName is "owner/name".
func (c *Client) ListBranches(ctx context.Context, installationID int64, repoFullName string) ([]string, error) {
	if !c.ok {
		return nil, ErrNotConfigured
	}
	owner, name, ok := splitFullName(repoFullName)
	if !ok {
		return nil, fmt.Errorf("invalid repository name %q", repoFullName)
	}
	cl, err := c.installationClient(installationID)
	if err != nil {
		return nil, err
	}

	opts := &github.BranchListOptions{ListOptions: github.ListOptions{PerPage: perPage}}
	var names []string
	for page := 0; page < maxPages; page++ {
		branches, resp, err := cl.Repositories.ListBranches(ctx, owner, name, opts)
		if err != nil {
			return nil, toAPIError(resp, err)
		}
		for _, b := range branches {
			names = append(names, b.GetName())
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return names, nil
}

// InstallURL is the GitHub page where an admin installs the App. state is echoed
// back to our setup URL so we can tie the installation to the right org.
func (c *Client) InstallURL(state string) (string, error) {
	if !c.ok {
		return "", ErrNotConfigured
	}
	return fmt.Sprintf("https://github.com/apps/%s/installations/new?state=%s",
		c.cfg.Slug, url.QueryEscape(state)), nil
}

// APIError carries a GitHub HTTP status so callers can distinguish, e.g., a
// revoked installation (404) from an auth problem (401).
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("github api error (%d): %s", e.Status, e.Message)
}

// toAPIError normalises a go-github failure into our status-carrying APIError.
func toAPIError(resp *github.Response, err error) error {
	status := 0
	if resp != nil && resp.Response != nil {
		status = resp.StatusCode
	}
	message := err.Error()

	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		if status == 0 && ghErr.Response != nil {
			status = ghErr.Response.StatusCode
		}
		if ghErr.Message != "" {
			message = ghErr.Message
		}
	}
	return &APIError{Status: status, Message: message}
}

// toRepo maps a go-github repository to our trimmed Repo. The getters are
// nil-safe, and Description/Language stay as pointers so "absent" is preserved.
func toRepo(r *github.Repository) Repo {
	return Repo{
		ExternalID:    r.GetID(),
		Name:          r.GetName(),
		FullName:      r.GetFullName(),
		Owner:         r.GetOwner().GetLogin(),
		Description:   r.Description,
		Private:       r.GetPrivate(),
		DefaultBranch: r.GetDefaultBranch(),
		Language:      r.Language,
		HTMLURL:       r.GetHTMLURL(),
		CloneURL:      r.GetCloneURL(),
		PushedAt:      pushedAt(r),
	}
}

func pushedAt(r *github.Repository) string {
	ts := r.GetPushedAt()
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339)
}

func splitFullName(fullName string) (owner, name string, ok bool) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
