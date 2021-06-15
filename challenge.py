import os
import json
import datetime
import subprocess
import dateutil.parser

def multiplier(effort):
    """Return the multiplier for the effort, 1.0 is the default."""
    efforts = (
        (  0,   25, 1.00),
        ( 25,   50, 1.25),
        ( 50,  125, 1.75),
        (125, 1000, 2.00),
    )
    for min, max, mul in efforts:
        if effort >= min and effort < max:
            return mul
    return 1.0

def week(date):
    """Return the week in which the activity falls, 0 if outside the competition."""
    weeks = (
        (datetime.date(2021, 6,  7), datetime.date(2021, 6, 13)),
        (datetime.date(2021, 6, 14), datetime.date(2021, 6, 20)),
        (datetime.date(2021, 6, 21), datetime.date(2021, 6, 27)),
        (datetime.date(2021, 6, 28), datetime.date(2021, 7,  4)),
    )
    for i, (start, end) in enumerate(weeks):
        if date >= start and date < end:
            return i+1
    return 0

if __name__ == "__main__":
    score    = {1:0, 2:0, 3:0, 4:0}
    calories = {1:0, 2:0, 3:0, 4:0}

    # query strava activities as json lines, parse out the interesting bits
    activities = subprocess.check_output(["gravl", "-c", "strava", "activities", "-N", "100"], text=True)
    for activity in (x for x in activities.split(os.linesep) if x):
        act = json.loads(activity)
        date = dateutil.parser.parse(act["start_date"]).date()
        w = week(date)
        if not w in score:
            continue

        aid = act["id"]
        effort = act["suffer_score"]
        mult = multiplier(effort)
        minutes = act["moving_time"] / 60.0
        score[w] += minutes * mult

        cal = subprocess.check_output(["gravl", "-c", "strava", "activity", str(aid)], text=True)
        act = json.loads(cal)
        calories[w] += act["calories"]

    print(score)
    print(calories)
