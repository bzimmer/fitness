package fitness

type Credentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

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
