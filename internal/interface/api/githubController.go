package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/vibino-xyz/commons/jwtauth"
	"github.com/vibino-xyz/commons/whttp"
	ghapi "github.com/vibino-xyz/synthy/internal/infra/github"
	ghuse "github.com/vibino-xyz/synthy/internal/usecase/github"
)

// Roles that may manage sources. Authorization is derived entirely from the
// role claim nexy embeds in the access token — synthy does not own the
// membership tables, it trusts the (verified) token.
const (
	roleOwner = "OWNER"
	roleAdmin = "ADMIN"
)

// GitHubController serves the GitHub App connection surface: install link,
// connect/disconnect, connection status, and repo/branch listing.
type GitHubController struct {
	service  *ghuse.Service
	verifier *jwtauth.Verifier
}

func NewGitHubController(service *ghuse.Service, verifier *jwtauth.Verifier) whttp.Controller {
	return &GitHubController{service: service, verifier: verifier}
}

func (c *GitHubController) Register(e *echo.Echo) {
	g := e.Group("/github", jwtauth.Middleware(c.verifier))
	g.GET("/connection", c.handleConnection)
	g.GET("/install-url", c.handleInstallURL)
	g.POST("/connect", c.handleConnect)
	g.DELETE("/connection", c.handleDisconnect)
	g.GET("/repositories", c.handleRepositories)
	g.GET("/branches", c.handleBranches)
}

func (c *GitHubController) handleConnection(ctx *echo.Context) error {
	claims, err := requireMember(ctx)
	if err != nil {
		return err
	}
	view, err := c.service.Connection(ctx.Request().Context(), claims.OrganizationId)
	if err != nil {
		return githubError(err)
	}
	return ctx.JSON(http.StatusOK, view)
}

func (c *GitHubController) handleInstallURL(ctx *echo.Context) error {
	claims, err := requireManager(ctx)
	if err != nil {
		return err
	}
	link, err := c.service.InstallURL(claims.OrganizationId)
	if err != nil {
		return githubError(err)
	}
	return ctx.JSON(http.StatusOK, map[string]string{"url": link})
}

func (c *GitHubController) handleConnect(ctx *echo.Context) error {
	claims, err := requireManager(ctx)
	if err != nil {
		return err
	}

	var req struct {
		InstallationID int64  `json:"installation_id"`
		State          string `json:"state"`
	}
	if err := ctx.Bind(&req); err != nil {
		return badRequest("invalid request body")
	}
	if req.InstallationID == 0 {
		return badRequest("installation_id is required")
	}
	// If GitHub echoed our state (the org id), it must match the caller's org.
	if req.State != "" && req.State != claims.OrganizationId {
		return echo.NewHTTPError(http.StatusForbidden, "installation belongs to a different organization")
	}

	view, err := c.service.Connect(ctx.Request().Context(), claims.OrganizationId, req.InstallationID)
	if err != nil {
		return githubError(err)
	}
	return ctx.JSON(http.StatusOK, view)
}

func (c *GitHubController) handleDisconnect(ctx *echo.Context) error {
	claims, err := requireManager(ctx)
	if err != nil {
		return err
	}
	if err := c.service.Disconnect(ctx.Request().Context(), claims.OrganizationId); err != nil {
		return githubError(err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (c *GitHubController) handleRepositories(ctx *echo.Context) error {
	claims, err := requireMember(ctx)
	if err != nil {
		return err
	}
	repos, err := c.service.ListRepositories(ctx.Request().Context(), claims.OrganizationId)
	if err != nil {
		return githubError(err)
	}
	if repos == nil {
		repos = []ghapi.Repo{}
	}
	return ctx.JSON(http.StatusOK, map[string]any{"repositories": repos})
}

func (c *GitHubController) handleBranches(ctx *echo.Context) error {
	claims, err := requireMember(ctx)
	if err != nil {
		return err
	}
	repo := strings.TrimSpace(ctx.QueryParam("repo"))
	if repo == "" {
		return badRequest("repo query parameter is required")
	}
	branches, err := c.service.ListBranches(ctx.Request().Context(), claims.OrganizationId, repo)
	if err != nil {
		return githubError(err)
	}
	if branches == nil {
		branches = []string{}
	}
	return ctx.JSON(http.StatusOK, map[string]any{"branches": branches})
}

// --- authorization (from the JWT claims) ---

// requireMember requires a valid token scoped to an organization.
func requireMember(ctx *echo.Context) (*jwtauth.Claims, error) {
	claims := jwtauth.ClaimsFromContext(ctx)
	if claims == nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if claims.OrganizationId == "" {
		return nil, badRequest("no active organization")
	}
	return claims, nil
}

// requireManager additionally requires an OWNER/ADMIN role.
func requireManager(ctx *echo.Context) (*jwtauth.Claims, error) {
	claims, err := requireMember(ctx)
	if err != nil {
		return nil, err
	}
	if claims.Role != roleOwner && claims.Role != roleAdmin {
		return nil, echo.NewHTTPError(http.StatusForbidden, "requires an admin or owner role")
	}
	return claims, nil
}

func badRequest(message string) error {
	return echo.NewHTTPError(http.StatusBadRequest, message)
}

// githubError maps GitHub-specific errors to HTTP statuses.
func githubError(err error) error {
	switch {
	case errors.Is(err, ghuse.ErrNotConfigured):
		return echo.NewHTTPError(http.StatusServiceUnavailable, "GitHub App is not configured on the server")
	case errors.Is(err, ghuse.ErrNotConnected):
		return echo.NewHTTPError(http.StatusNotFound, "GitHub is not connected for this organization")
	}

	var apiErr *ghapi.APIError
	if errors.As(err, &apiErr) {
		if apiErr.Status >= 400 && apiErr.Status < 500 {
			return echo.NewHTTPError(apiErr.Status, "GitHub: "+apiErr.Message)
		}
		return echo.NewHTTPError(http.StatusBadGateway, "GitHub request failed: "+apiErr.Message)
	}

	return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
}
