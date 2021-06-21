package fitness

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"

	"github.com/bzimmer/gravl/pkg/providers/activity/strava"
)

// LoginHandler redirects to the oauth provider's credential acceptance page
func LoginHandler(cfg *oauth2.Config, state string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := cfg.AuthCodeURL(state)
		log.Info().Int("code", http.StatusFound).Str("path", u).Str("action", "login").Msg("redirect")
		c.Redirect(http.StatusFound, u)
	}
}

// LogoutHandler removes the token from the session
func LogoutHandler(cfg *oauth2.Config, state, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		session.Clear()
		if err := session.Save(); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		log.Info().Int("code", http.StatusTemporaryRedirect).Str("path", path).Str("action", "logout").Msg("redirect")
		c.Redirect(http.StatusTemporaryRedirect, path)
	}
}

// AuthCallbackHandler receives the callback from the oauth provider with the credentials
func AuthCallbackHandler(cfg *oauth2.Config, state, path string) gin.HandlerFunc {
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

		session := sessions.Default(c)
		session.Set("token", token.RefreshToken)
		if err := session.Save(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to save session value"})
			return
		}

		log.Info().Int("code", http.StatusTemporaryRedirect).Str("path", path).Str("action", "auth callback").Msg("redirect")
		c.Redirect(http.StatusTemporaryRedirect, path)
	}
}

func ScoreboardHandler(clientID, clientSecret string, config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		token := session.Get("token").(string)
		client, err := strava.NewClient(
			strava.WithTokenCredentials(token, token, time.Now().Add(-1*time.Minute)),
			strava.WithClientCredentials(clientID, clientSecret),
			strava.WithAutoRefresh(c.Request.Context()))
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		sb := NewScoreboard(config)
		board, err := sb.Scoreboard(c.Request.Context(), client)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.IndentedJSON(http.StatusOK, board)
	}
}

func LambdaHandler(gl *ginadapter.GinLambda) interface{} {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		// If no name is provided in the HTTP request body, throw an error
		return gl.ProxyWithContext(ctx, req)
	}
}
