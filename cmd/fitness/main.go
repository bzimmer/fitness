package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/oauth2"

	"github.com/bzimmer/fitness"
	"github.com/bzimmer/gravl/pkg/providers/activity/strava"
	"github.com/bzimmer/gravl/pkg/web"
)

type Credentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func credentials(c *cli.Context) (*Credentials, error) {
	fp, err := os.Open(c.String("config"))
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	val, _ := io.ReadAll(fp)

	var creds Credentials
	err = json.Unmarshal(val, &creds)
	if err != nil {
		return nil, err
	}
	return &creds, nil
}

// token produces a random token of length `n`
func token(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func newRouter(c *cli.Context) (*http.ServeMux, error) {
	creds, err := credentials(c)
	if err != nil {
		return nil, err
	}
	state, err := token(16)
	if err != nil {
		return nil, err
	}
	filesystem, err := fs.Sub(fitness.Content, "html")
	if err != nil {
		return nil, err
	}
	address := fmt.Sprintf("%s:%d", c.String("origin"), c.Int("port"))
	config := &oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Scopes:       []string{"read_all,profile:read_all,activity:read_all"},
		RedirectURL:  fmt.Sprintf("%s/callback", address),
		Endpoint:     strava.Endpoint}
	handle := web.NewLogHandler(log.Logger)
	mux := http.NewServeMux()
	mux.Handle("/", handle(http.FileServer(http.FS(filesystem))))
	mux.Handle("/login", handle(fitness.AuthHandler(config, state)))
	mux.Handle("/callback", handle(fitness.AuthCallbackHandler(config, state)))
	mux.HandleFunc("/scoreboard", func(w http.ResponseWriter, r *http.Request) {
		client, err := strava.NewClient(
			strava.WithTokenCredentials(creds.AccessToken, creds.RefreshToken, time.Now().Add(-1*time.Minute)),
			strava.WithClientCredentials(creds.ClientID, creds.ClientSecret),
			strava.WithAutoRefresh(c.Context))
		if err != nil {
			http.Error(w, "State invalid", http.StatusBadRequest)
			return
		}
		board, err := fitness.Scoreboard(c.Context, client)
		if err != nil {
			http.Error(w, "Failed scoreboard", http.StatusBadRequest)
			return
		}
		b, err := json.MarshalIndent(board, "", " ")
		if err != nil {
			http.Error(w, "Failed to marshal", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, "%s\n", b)
	})
	return mux, nil
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

var listCommand = &cli.Command{
	Name:  "list",
	Usage: "List",
	Action: func(c *cli.Context) error {
		creds, err := credentials(c)
		if err != nil {
			return err
		}
		client, err := strava.NewClient(
			strava.WithTokenCredentials(creds.AccessToken, creds.RefreshToken, time.Now().Add(-1*time.Minute)),
			strava.WithClientCredentials(creds.ClientID, creds.ClientSecret),
			strava.WithAutoRefresh(c.Context))
		if err != nil {
			return err
		}
		board, err := fitness.Scoreboard(c.Context, client)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(board, "", " ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(c.App.Writer, "%s\n", b)
		return err
	},
}

func main() {
	app := &cli.App{
		Name:     "fitness",
		HelpName: "fitness",
		Usage:    "Fitness Challenge",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config",
				Aliases:  []string{"c"},
				Required: true,
				Usage:    "file with strava credentials",
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
			listCommand,
			serveCommand,
		},
	}
	if err := app.RunContext(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
