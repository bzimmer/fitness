package fitness

import (
	"context"
	"math"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bzimmer/gravl/pkg/providers/activity"
	"github.com/bzimmer/gravl/pkg/providers/activity/strava"
)

var _weeks = [][]time.Time{
	{
		time.Date(2021, time.June, 07, 0, 0, 0, 0, time.UTC),
		time.Date(2021, time.June, 14, 0, 0, 0, 0, time.UTC),
	},
	{
		time.Date(2021, time.June, 14, 0, 0, 0, 0, time.UTC),
		time.Date(2021, time.June, 21, 0, 0, 0, 0, time.UTC),
	},
	{
		time.Date(2021, time.June, 21, 0, 0, 0, 0, time.UTC),
		time.Date(2021, time.June, 28, 0, 0, 0, 0, time.UTC),
	},
	{
		time.Date(2021, time.June, 28, 0, 0, 0, 0, time.UTC),
		time.Date(2021, time.July, 05, 0, 0, 0, 0, time.UTC),
	},
}

func score(act *strava.Activity) int {
	var val float64
	movingTime := time.Minute * time.Duration(math.Ceil(act.MovingTime.Minutes()))
	switch act.Type {
	case "Hike":
		val = 1.75
		if movingTime > 5*time.Hour {
			val = 2.0
		}
	case "Ride":
		val = 1.75
		if movingTime > 4*time.Hour {
			val = 2.0
		}
	case "Run":
		val = 1.75
		if movingTime > 2*time.Hour {
			val = 2.0
		}
	default:
		val = 1.0
	}
	return int(movingTime.Minutes() * val)
}

func week(act *strava.Activity) int {
	for i := 0; i < len(_weeks); i++ {
		if act.StartDate.After(_weeks[i][0]) && act.StartDate.Before(_weeks[i][1]) {
			return i + 1
		}
	}
	return 0
}

func calories(act *strava.Activity) int {
	switch act.ID {
	case 5497755660:
		// fenix 6x didn't sync with wahoo bolt so the ride only shows 249 calories -- bah!
		return int(3594)
	}
	return int(act.Calories)
}

func scoreboard(acts []*Activity) []*Week {
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
func Scoreboard(c context.Context, client *strava.Client) ([]*Week, error) {
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
				return scoreboard(scores), nil
			}
			if res.Err != nil {
				return nil, res.Err
			}
			wk := week(res.Activity)
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
				Score:    score(act),
				Calories: calories(act),
			})
		}
	}
}
