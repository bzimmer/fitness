package fitness

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"

	"github.com/bzimmer/gravl/pkg/providers/activity/strava"
)

const sessionName = "fitness"

// LoginHandler redirects to the oauth provider's credential acceptance page
func LoginHandler(cfg *oauth2.Config, state string) echo.HandlerFunc {
	return func(c echo.Context) error {
		u := cfg.AuthCodeURL(state)
		return c.Redirect(http.StatusFound, u)
	}
}

// LogoutHandler removes the token from the session
func LogoutHandler(cfg *oauth2.Config, state, path string) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, err := session.Get(sessionName, c)
		if err != nil {
			return err
		}
		session.Values = map[interface{}]interface{}{}
		if err := session.Save(c.Request(), c.Response()); err != nil {
			return err
		}
		return c.Redirect(http.StatusFound, path)
	}
}

// AuthCallbackHandler receives the callback from the oauth provider with the credentials
func AuthCallbackHandler(cfg *oauth2.Config, state, path string) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.QueryParam("state") != state {
			log.Error().Msg("state does not match")
			return fmt.Errorf("invalid state")
		}

		code := c.QueryParam("code")
		if code == "" {
			return fmt.Errorf("code not present")
		}

		token, err := cfg.Exchange(c.Request().Context(), code)
		if err != nil {
			log.Error().Err(err).Msg("failed to exchange code for token")
			return err
		}

		session, err := session.Get(sessionName, c)
		if err != nil {
			// log the error but do nothing; a new session has been created
			log.Error().Err(err).Msg("failed to find session")
			if session == nil {
				return err
			}
		}
		session.Values["token"] = token.RefreshToken
		if err := session.Save(c.Request(), c.Response()); err != nil {
			log.Error().Err(err).Msg("failed to save session")
			return err
		}
		return c.Redirect(http.StatusTemporaryRedirect, path)
	}
}

// ScoreboardHandler generates the scoreboard for the user
func ScoreboardHandler(clientID, clientSecret string, config *Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, err := session.Get(sessionName, c)
		if err != nil {
			return err
		}
		token, ok := session.Values["token"]
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}
		client, err := strava.NewClient(
			strava.WithTokenCredentials(token.(string), token.(string), time.Now().Add(-1*time.Minute)),
			strava.WithClientCredentials(clientID, clientSecret),
			strava.WithAutoRefresh(c.Request().Context()))
		if err != nil {
			return err
		}
		sb := NewScoreboard(config)
		board, err := sb.Scoreboard(c.Request().Context(), client)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, board)
	}
}
