package fitness

import (
	"net/http"
	"time"

	"github.com/bzimmer/gravl/pkg/providers/activity/strava"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type TokenCallback func(*gin.Context, *oauth2.Token)

func tokenCallback(c *gin.Context, t *oauth2.Token) {
	c.IndentedJSON(http.StatusOK, t)
}

// AuthHandler redirects to the oauth provider's credential acceptance page
func AuthHandler(cfg *oauth2.Config, state string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := cfg.AuthCodeURL(state)
		c.Redirect(http.StatusFound, u)
	}
}

// AuthCallbackHandler receives the callback from the oauth provider with the credentials
func AuthCallbackHandler(c *oauth2.Config, state string) gin.HandlerFunc {
	return AuthCallbackHandlerF(c, state, tokenCallback)
}

// AuthCallbackHandler receives the callback from the oauth provider with the credentials
func AuthCallbackHandlerF(cfg *oauth2.Config, state string, f TokenCallback) gin.HandlerFunc {
	type form struct {
		State string `form:"state" binding:"required"`
		Code  string `form:"code" binding:"required"`
	}
	return func(c *gin.Context) {
		var cb form
		if err := c.ShouldBind(&cb); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if cb.State != state {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}

		code := cb.Code
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is nil"})
			return
		}

		token, err := cfg.Exchange(c.Request.Context(), code)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		f(c, token)
	}
}

func ScoreboardHandler(creds *Credentials) gin.HandlerFunc {
	return func(c *gin.Context) {
		client, err := strava.NewClient(
			strava.WithTokenCredentials(creds.AccessToken, creds.RefreshToken, time.Now().Add(-1*time.Minute)),
			strava.WithClientCredentials(creds.ClientID, creds.ClientSecret),
			strava.WithAutoRefresh(c.Request.Context()))
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		board, err := Scoreboard(c.Request.Context(), client)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.IndentedJSON(http.StatusOK, board)
	}
}
