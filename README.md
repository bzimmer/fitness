## fitness

Compute Fitness Challenge weekly scores from Strava activities.

### configuration

```json
{
    "client_id":"client id from Strava",
    "client_secret":"client secret from Strava",
    "access_token":"access token",
    "refresh_token":"refresh token"
}
```

### invocation

```sh
~ > fitness -c .fitness.json
2021-06-17T08:49:30-07:00 INF do all=0 count=100 n=0 start=0 total=100
2021-06-17T08:49:33-07:00 INF do all=100 count=100 n=100 start=1 total=100
2021-06-17T08:49:33-07:00 INF query id=5482290987 name="Whales and ivy"
2021-06-17T08:49:34-07:00 INF query id=5477442296 name="Back on the barlows"
2021-06-17T08:49:35-07:00 INF query id=5471042320 name=Monday
2021-06-17T08:49:35-07:00 INF query id=5464233161 name="Wet root parade"
2021-06-17T08:49:36-07:00 INF query id=5459419166 name="Day Road"
2021-06-17T08:49:37-07:00 INF query id=5458278964 name=Rockaway
2021-06-17T08:49:38-07:00 INF query id=5474614434 name="Break the day"
2021-06-17T08:49:38-07:00 INF query id=5449549118 name="Across my path"
2021-06-17T08:49:39-07:00 INF query id=5443311043 name="A little carry"
2021-06-17T08:49:39-07:00 INF query id=5431323798 name="Colchuck Lake"
[
 {
  "week": 2,
  "score": 397,
  "calories": 3065,
  "activities": [
   {
    "id": 5482290987,
    "type": "Ride",
    "name": "Whales and ivy",
    "week": 2,
    "score": 245,
    "calories": 2029
   },
   {
    "id": 5477442296,
    "type": "Ride",
    "name": "Back on the barlows",
    "week": 2,
    "score": 121,
    "calories": 883
   },
   {
    "id": 5471042320,
    "type": "Walk",
    "name": "Monday",
    "week": 2,
    "score": 31,
    "calories": 153
   }
  ]
 },
 {
  "week": 1,
  "score": 996,
  "calories": 5331,
  "activities": [
   {
    "id": 5464233161,
    "type": "Ride",
    "name": "Wet root parade",
    "week": 1,
    "score": 54,
    "calories": 364
   },
   {
    "id": 5459419166,
    "type": "Ride",
    "name": "Day Road",
    "week": 1,
    "score": 187,
    "calories": 1436
   },
   {
    "id": 5458278964,
    "type": "Walk",
    "name": "Rockaway",
    "week": 1,
    "score": 72,
    "calories": 335
   },
   {
    "id": 5474614434,
    "type": "Walk",
    "name": "Break the day",
    "week": 1,
    "score": 90,
    "calories": 423
   },
   {
    "id": 5449549118,
    "type": "Ride",
    "name": "Across my path",
    "week": 1,
    "score": 104,
    "calories": 735
   },
   {
    "id": 5443311043,
    "type": "Ride",
    "name": "A little carry",
    "week": 1,
    "score": 145,
    "calories": 921
   },
   {
    "id": 5431323798,
    "type": "Hike",
    "name": "Colchuck Lake",
    "week": 1,
    "score": 344,
    "calories": 1117
   }
  ]
 }
]
```
