package fitness

import (
	"encoding/json"
	"net/http"

	"golang.org/x/oauth2"
)

type TokenCallback func(w http.ResponseWriter, r *http.Request, t *oauth2.Token)

func tokenCallback(w http.ResponseWriter, r *http.Request, t *oauth2.Token) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// AuthHandler redirects to the oauth provider's credential acceptance page
func AuthHandler(c *oauth2.Config, state string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := c.AuthCodeURL(state)
		http.Redirect(w, r, u, http.StatusFound)
	}
}

// AuthCallbackHandler receives the callback from the oauth provider with the credentials
func AuthCallbackHandler(c *oauth2.Config, state string) http.HandlerFunc {
	return AuthCallbackHandlerF(c, state, tokenCallback)
}

// AuthCallbackHandler receives the callback from the oauth provider with the credentials
func AuthCallbackHandlerF(c *oauth2.Config, state string, f TokenCallback) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s := r.Form.Get("state")
		if s != state {
			http.Error(w, "State invalid", http.StatusBadRequest)
			return
		}

		code := r.Form.Get("code")
		if code == "" {
			http.Error(w, "Code not found", http.StatusBadRequest)
			return
		}

		token, err := c.Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		f(w, r, token)
	}
}
