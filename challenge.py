import os
import json
import logging
import datetime
import subprocess
import dateutil.parser
from pythonjsonlogger import jsonlogger

weeks = (
    (datetime.date(2021, 6,  7), datetime.date(2021, 6, 13)),
    (datetime.date(2021, 6, 14), datetime.date(2021, 6, 20)),
    (datetime.date(2021, 6, 21), datetime.date(2021, 6, 27)),
    (datetime.date(2021, 6, 28), datetime.date(2021, 7,  4)),
)

def multiplier(act, suffer=False):
    """Return the multiplier for the activity, 1.0 is the default."""
    if suffer:
        efforts = (
            (  0,   25, 1.00),
            ( 25,   50, 1.25),
            ( 50,  125, 1.75),
            (125, 1000, 2.00),
        )
        effort = act["suffer_score"]
        for min, max, mul in efforts:
            if effort >= min and effort < max:
                return mul
        return 1.0

    efforts = {
        "Hike":1.75,
        "Ride":1.75,
        "Walk":1.00,
    }
    return efforts[act["type"]]

def week(date):
    """Return the week in which the activity falls, 0 if outside the competition."""
    for i, (start, end) in enumerate(weeks):
        if date >= start and date <= end:
            return i
    return None

if __name__ == "__main__":
    logHandler = logging.StreamHandler()
    formatter = jsonlogger.JsonFormatter()
    logHandler.setFormatter(formatter)
    logger = logging.getLogger()
    logger.addHandler(logHandler)
    logger.setLevel(logging.INFO)

    n = len(weeks)
    score    = dict((x, 0) for x in range(n))
    calories = dict((x, 0) for x in range(n))

    # query strava activities as json lines, parse out the interesting bits
    activities = subprocess.check_output(["gravl", "-c", "strava", "activities", "-N", "100"], text=True)
    for activity in (x for x in activities.split(os.linesep) if x):
        act = json.loads(activity)
        date = dateutil.parser.parse(act["start_date"]).date()
        wk = week(date)
        if wk is None:
            continue

        logger.info({"message":"activity", "name": act["name"]})

        aid = act["id"]
        mult = multiplier(act)
        minutes = act["moving_time"] / 60.0
        score[wk] += round(minutes * mult)

        cal = subprocess.check_output(["gravl", "-c", "strava", "activity", str(aid)], text=True)
        act = json.loads(cal)
        calories[wk] += round(act["calories"])

    print(json.dumps({"score": score, "calories":calories}))
