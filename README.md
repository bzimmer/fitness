## fitness

Compute Fitness Challenge weekly scores from Strava activities.

### running locally

1. Install [Taskfile](https://taskfile.dev/#/taskfile_versions?id=version-3).

2. Create a `.fitness.env` configuration file:

```sh
FITNESS_SESSION_KEY=<session key>
STRAVA_CLIENT_ID=<strava client id>
STRAVA_CLIENT_SECRET=<strave client secret>
BASE_URL=http://localhost:9010/fitness
```

3. To run:

```sh
$ task fitness
```

Make sure the Strava API callback url is configured for the domain in the Strava settings.

### running at netlify

Publish to netlify via github integration.
