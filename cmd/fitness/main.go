package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/oauth2"

	"github.com/bzimmer/fitness"
	"github.com/bzimmer/gravl/pkg/providers/activity/strava"
)

func config(c *cli.Context) (*fitness.Config, error) {
	fp, err := os.Open(c.String("config"))
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	val, _ := io.ReadAll(fp)

	var cfg fitness.Config
	err = json.Unmarshal(val, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// token produces a random token of length `n`
func token(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func newRouter(c *cli.Context) (*gin.Engine, error) {
	cfg, err := config(c)
	if err != nil {
		return nil, err
	}
	state, err := token(16)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.ParseFS(fitness.Content, "templates/index.html")
	if err != nil {
		return nil, err
	}

	address := fmt.Sprintf("%s:%d", c.String("origin"), c.Int("port"))
	config := &oauth2.Config{
		ClientID:     c.String("client-id"),
		ClientSecret: c.String("client-secret"),
		Scopes:       []string{"read_all,profile:read_all,activity:read_all"},
		RedirectURL:  fmt.Sprintf("%s/auth/callback", address),
		Endpoint:     strava.Endpoint}

	r := gin.Default()
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("default", store))
	r.SetHTMLTemplate(tmpl)
	r.GET("/", func(c *gin.Context) {
		session := sessions.Default(c)
		if session.Get("token") == nil {
			c.Redirect(http.StatusTemporaryRedirect, "/auth/login")
			return
		}
		c.HTML(http.StatusOK, "index.html", nil)
	})
	r.GET("/auth/login", fitness.LoginHandler(config, state))
	r.GET("/auth/logout", fitness.LogoutHandler(config, state))
	r.GET("/auth/callback", fitness.AuthCallbackHandler(config, state))
	r.GET("/scoreboard", fitness.ScoreboardHandler(
		config.ClientID, config.ClientSecret, cfg))
	return r, nil
}

var serveCommand = &cli.Command{
	Name:  "serve",
	Usage: "Serve",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "origin",
			Value: "http://localhost",
			Usage: "Callback origin",
		},
		&cli.IntFlag{
			Name:  "port",
			Value: 9001,
			Usage: "Port on which to listen",
		},
	},
	Action: func(c *cli.Context) error {
		mux, err := newRouter(c)
		if err != nil {
			return err
		}
		address := fmt.Sprintf("0.0.0.0:%d", c.Int("port"))
		log.Info().Str("address", address).Msg("serving")
		return http.ListenAndServe(address, mux)
	},
}

func main() {
	app := &cli.App{
		Name:     "fitness",
		HelpName: "fitness",
		Usage:    "Fitness Challenge",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "client-id",
				Required: true,
				Usage:    "client id",
				EnvVars:  []string{"STRAVA_CLIENT_ID"},
			},
			&cli.StringFlag{
				Name:     "client-secret",
				Required: true,
				Usage:    "client secret",
				EnvVars:  []string{"STRAVA_CLIENT_SECRET"},
			},
			&cli.StringFlag{
				Name:     "config",
				Aliases:  []string{"c"},
				Required: true,
				Usage:    "file with configuration",
			},
		},
		ExitErrHandler: func(c *cli.Context, err error) {
			if err == nil {
				return
			}
			log.Error().Err(err).Msg(c.App.Name)
		},
		Before: func(c *cli.Context) error {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			zerolog.DurationFieldUnit = time.Millisecond
			zerolog.DurationFieldInteger = false
			log.Logger = log.Output(
				zerolog.ConsoleWriter{
					Out:        c.App.ErrWriter,
					NoColor:    false,
					TimeFormat: time.RFC3339,
				},
			)
			return nil
		},
		Commands: []*cli.Command{
			serveCommand,
		},
	}
	if err := app.RunContext(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
