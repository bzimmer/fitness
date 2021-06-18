package fitness

import (
	"net/http"

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

type Callback struct {
	State string `form:"state" binding:"required"`
	Code  string `form:"code" binding:"required"`
}

// AuthCallbackHandler receives the callback from the oauth provider with the credentials
func AuthCallbackHandlerF(cfg *oauth2.Config, state string, f TokenCallback) gin.HandlerFunc {
	return func(c *gin.Context) {
		var res Callback
		if err := c.ShouldBind(&res); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if res.State != state {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}

		code := res.Code
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
