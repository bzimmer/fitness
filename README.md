## fitness

Compute Fitness Challenge weekly scores from Strava activities.

### running locally

Use a script similar to:

```#! /bin/sh
set -ex

GIN_MODE=release \
FITNESS_SESSION_KEY=<session key> \
STRAVA_CLIENT_ID=<strava client id> \
STRAVA_CLIENT_SECRET=<strave client secret> \
BASE_URL=http://localhost:9010/fitness \
go run cmd/fitness/main.go --port 9010
```

Make the Strava API callback url is configured for the domain in the Strava settings.

### running at netlify

Publish to netlify via github integration.
