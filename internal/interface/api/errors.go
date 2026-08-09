package api

import (
	"errors"

	"github.com/labstack/echo/v5"
	"github.com/vibino-xyz/commons/vhttp"
	ghapi "github.com/vibino-xyz/synthy/internal/infra/github"
	ghuse "github.com/vibino-xyz/synthy/internal/usecase/github"
)

func githubError(err error) error {
	switch {
	case errors.Is(err, ghuse.ErrNotConfigured):
		return vhttp.ServiceUnavailable("GitHub App is not configured on the server")
	case errors.Is(err, ghuse.ErrNotConnected):
		return vhttp.NotFound("GitHub is not connected for this organization")
	}

	var apiErr *ghapi.APIError
	if errors.As(err, &apiErr) {
		if apiErr.Status >= 400 && apiErr.Status < 500 {
			return echo.NewHTTPError(apiErr.Status, "GitHub: "+apiErr.Message)
		}
		return vhttp.BadGateway("GitHub request failed: " + apiErr.Message)
	}

	return vhttp.Internal("internal server error")
}
