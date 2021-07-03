package fitness

import "time"

type Activity struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Week     int    `json:"week"`
	Score    int    `json:"score"`
	Calories int    `json:"calories"`
}

type Week struct {
	Week       int         `json:"week"`
	Score      int         `json:"score"`
	Calories   int         `json:"calories"`
	Activities []*Activity `json:"activities"`
}

type Config struct {
	Weeks []struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"weeks"`
	Epic []struct {
		Type       string  `json:"type"`
		Minutes    int     `json:"minutes"`
		Multiplier float64 `json:"multiplier"`
	} `json:"epic"`
	Calories []struct {
		ID       int64 `json:"id"`
		Override int   `json:"override"`
	} `json:"calories"`
}
