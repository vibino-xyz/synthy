package api

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/vibino-xyz/commons/jwtauth"
	"github.com/vibino-xyz/commons/whttp"
	ghapi "github.com/vibino-xyz/synthy/internal/infra/github"
	ghuse "github.com/vibino-xyz/synthy/internal/usecase/github"
)

type GitHubController struct {
	service  *ghuse.Service
	verifier *jwtauth.Verifier
}

func NewGitHubController(service *ghuse.Service, verifier *jwtauth.Verifier) whttp.Controller {
	return &GitHubController{service: service, verifier: verifier}
}

func (c *GitHubController) Register(e *echo.Echo) {
	g := e.Group("/synthy/v1/github", jwtauth.Middleware(c.verifier))
	g.GET("/connection", c.handleConnection)
	g.GET("/install-url", c.handleInstallURL)
	g.POST("/connect", c.handleConnect)
	g.DELETE("/connection", c.handleDisconnect)
	g.GET("/repositories", c.handleRepositories)
	g.GET("/repositories/indexed", c.handleIndexedRepositories)
	g.GET("/branches", c.handleBranches)
	g.POST("/index", c.handleIndex)
}

func (c *GitHubController) handleConnection(ctx *echo.Context) error {
	claims, err := jwtauth.RequireMember(ctx)
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
	claims, err := jwtauth.RequireManager(ctx)
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
	claims, err := jwtauth.RequireManager(ctx)
	if err != nil {
		return err
	}

	var req struct {
		InstallationID int64  `json:"installation_id"`
		State          string `json:"state"`
	}
	if err := ctx.Bind(&req); err != nil {
		return whttp.BadRequest("invalid request body")
	}
	if req.InstallationID == 0 {
		return whttp.BadRequest("installation_id is required")
	}
	// If GitHub echoed our state (the org id), it must match the caller's org.
	if req.State != "" && req.State != claims.OrganizationId {
		return whttp.Forbidden("installation belongs to a different organization")
	}

	view, err := c.service.Connect(ctx.Request().Context(), claims.OrganizationId, req.InstallationID)
	if err != nil {
		return githubError(err)
	}
	return ctx.JSON(http.StatusOK, view)
}

func (c *GitHubController) handleDisconnect(ctx *echo.Context) error {
	claims, err := jwtauth.RequireManager(ctx)
	if err != nil {
		return err
	}
	if err := c.service.Disconnect(ctx.Request().Context(), claims.OrganizationId); err != nil {
		return githubError(err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (c *GitHubController) handleRepositories(ctx *echo.Context) error {
	claims, err := jwtauth.RequireMember(ctx)
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

func (c *GitHubController) handleIndexedRepositories(ctx *echo.Context) error {
	claims, err := jwtauth.RequireMember(ctx)
	if err != nil {
		return err
	}
	repos, err := c.service.ListIndexedRepositories(ctx.Request().Context(), claims.OrganizationId)
	if err != nil {
		return githubError(err)
	}
	return ctx.JSON(http.StatusOK, map[string]any{"repositories": repos})
}

func (c *GitHubController) handleBranches(ctx *echo.Context) error {
	claims, err := jwtauth.RequireMember(ctx)
	if err != nil {
		return err
	}
	repo := strings.TrimSpace(ctx.QueryParam("repo"))
	if repo == "" {
		return whttp.BadRequest("repo query parameter is required")
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

func (c *GitHubController) handleIndex(ctx *echo.Context) error {
	claims, err := jwtauth.RequireManager(ctx)
	if err != nil {
		return err
	}
	queued, err := c.service.IndexConnectedRepositories(ctx.Request().Context(), claims.OrganizationId)
	if err != nil {
		return githubError(err)
	}
	return ctx.JSON(http.StatusAccepted, map[string]any{"queued": queued})
}
