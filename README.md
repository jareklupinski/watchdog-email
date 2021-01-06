
# ⏰ 🐕 . 📧

## Watchdog.Email

Full source code for http://watchdog.email

Inspired by a lack of simple watchdog timers, I set out to see what makes them so difficult.

Currently deployed to Heroku with the following Free Apps:
- Heroku Redis
- Papertrail
- SendGrid

2 Free Dynos are used to run the service:
- web - handles frontend http requests
- worker - runs on a timer to send emails
