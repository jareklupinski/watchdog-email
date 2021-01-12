
# ⏰ 🐕 . 📧

## Watchdog.Email

Full source code for http://watchdog.email

Inspired by a lack of simple watchdog timers, I set out to see what makes them so difficult.

Currently deployed to Heroku with the following Free Apps:
- Heroku Redis
- Heroku Scheduler
- Papertrail
- SendGrid

3 Free Dynos are used to run the service:
- web - handles frontend http requests
- worker - sends an email if the watchdog fired
- timer - pings the web dyno to wake it up once per hour

Heroku's idling rules tell a worker dyno to go to sleep when its associated web dyno sleeps. 🤷
