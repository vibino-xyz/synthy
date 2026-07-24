// Package github is synthy's GitHub App integration: it authenticates as the
// Vibino GitHub App, exchanges installation ids for short-lived installation
// access tokens, and reads repositories/branches the installation can see.
package github

import (
	"crypto/rsa"
	"errors"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ErrNotConfigured is returned by the client when the GitHub App env vars are
// missing, so the API can respond with a clear "set this up first" message
// instead of failing obscurely.
var ErrNotConfigured = errors.New("github app is not configured")

// Config holds the resolved GitHub App credentials. Configured is false when
// the required env vars are absent — the rest of the app keeps working, the
// GitHub endpoints just report that setup is needed.
type Config struct {
	AppID      string
	Slug       string
	PrivateKey *rsa.PrivateKey
	Configured bool
}

// NewConfigFromEnv reads the GitHub App configuration:
//
//	GITHUB_APP_ID               - the numeric App ID
//	GITHUB_APP_SLUG             - the App's URL slug (for the install link)
//	GITHUB_APP_PRIVATE_KEY_PATH - path to the downloaded .pem, OR
//	GITHUB_APP_PRIVATE_KEY      - the PEM contents inline (\n escaped is fine)
//
// A missing/invalid key yields an unconfigured (not fatal) Config so the
// service can boot without GitHub set up yet.
func NewConfigFromEnv() (Config, error) {
	appID := strings.TrimSpace(os.Getenv("GITHUB_APP_ID"))
	slug := strings.TrimSpace(os.Getenv("GITHUB_APP_SLUG"))

	pem, err := readPrivateKeyPEM()
	if err != nil || appID == "" || slug == "" || len(pem) == 0 {
		return Config{Configured: false}, nil
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(pem)
	if err != nil {
		// Present but unparseable — treat as unconfigured rather than crash.
		return Config{Configured: false}, nil
	}

	return Config{AppID: appID, Slug: slug, PrivateKey: key, Configured: true}, nil
}

func readPrivateKeyPEM() ([]byte, error) {
	if path := strings.TrimSpace(os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH")); path != "" {
		return os.ReadFile(path)
	}
	if inline := os.Getenv("GITHUB_APP_PRIVATE_KEY"); strings.TrimSpace(inline) != "" {
		// Allow keys pasted into .env with literal \n rather than real newlines.
		return []byte(strings.ReplaceAll(inline, `\n`, "\n")), nil
	}
	return nil, nil
}
