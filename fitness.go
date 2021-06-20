package fitness

import (
	"context"
	"math"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bzimmer/gravl/pkg/providers/activity"
	"github.com/bzimmer/gravl/pkg/providers/activity/strava"
)

type Scoreboard struct {
	config *Config
}

func NewScoreboard(config *Config) *Scoreboard {
	return &Scoreboard{config: config}
}

func (b *Scoreboard) score(act *strava.Activity) int {
	var val float64
	movingTime := time.Minute * time.Duration(math.Ceil(act.MovingTime.Minutes()))
	switch act.Type {
	case "Hike", "Ride":
		val = 1.75
	case "Run":
		val = 1.50
	default:
		val = 1.0
	}
	for _, epic := range b.config.Epic {
		if epic.Type == act.Type {
			if movingTime > time.Minute*time.Duration(epic.Minutes) {
				val = epic.Multiplier
			}
		}
	}
	return int(movingTime.Minutes() * val)
}

func (b *Scoreboard) week(act *strava.Activity) int {
	for i, wk := range b.config.Weeks {
		if act.StartDate.After(wk.Start) && act.StartDate.Before((wk.End)) {
			return i + 1
		}
	}
	return 0
}

func (b *Scoreboard) calories(act *strava.Activity) int {
	for _, cal := range b.config.Calories {
		if cal.ID == act.ID {
			return cal.Override
		}
	}
	return int(act.Calories)
}

func (b *Scoreboard) scoreboard(acts []*Activity) []*Week {
	// group all activities into weeks
	w := make(map[int][]*Activity)
	for _, act := range acts {
		w[act.Week] = append(w[act.Week], act)
	}
	// summarize score and calories
	var res []*Week
	for wk, acts := range w {
		var cal, score int
		for _, act := range acts {
			score += act.Score
			cal += act.Calories
		}
		m := &Week{
			Week:       wk,
			Score:      score,
			Calories:   cal,
			Activities: acts,
		}
		res = append(res, m)
	}
	return res
}

// Scoreboard returns weekly scores and calories for the current athlete
func (b *Scoreboard) Scoreboard(c context.Context, client *strava.Client) ([]*Week, error) {
	ctx, cancel := context.WithTimeout(c, 2*time.Minute)
	defer cancel()

	var ok bool
	var scores []*Activity
	var res *strava.ActivityResult
	acts := client.Activity.Activities(ctx, activity.Pagination{Total: 100})
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case res, ok = <-acts:
			if !ok {
				return b.scoreboard(scores), nil
			}
			if res.Err != nil {
				return nil, res.Err
			}
			wk := b.week(res.Activity)
			if wk == 0 {
				continue
			}
			log.Info().Str("name", res.Activity.Name).Int64("id", res.Activity.ID).Msg("query")
			act, err := client.Activity.Activity(ctx, res.Activity.ID)
			if err != nil {
				return nil, err
			}
			scores = append(scores, &Activity{
				ID:       act.ID,
				Type:     act.Type,
				Name:     act.Name,
				Week:     wk,
				Score:    b.score(act),
				Calories: b.calories(act),
			})
		}
	}
}
