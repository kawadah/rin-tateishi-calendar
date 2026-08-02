# Rin Tateishi calendar

A subscribable calendar (`.ics`) of 立石凛 (Rin Tateishi)'s public schedule, mirrored from [立石凛 ＠ フリカレ](https://freecalend.com/open/mem231613) and refreshed automatically.

## Subscribe

Add this URL as a calendar subscription:

```text
https://calendars.seiyuu.app/rin-tateishi/calendar.ics
```

- **Apple Calendar:** File → New Calendar Subscription
  - [Use iCloud calendar subscriptions - Apple Support](https://support.apple.com/en-us/102301)
- **Google Calendar:** Other calendars → From URL
  - [Subscribe to someone else's calendar - Computer - Google Calendar Help](https://support.google.com/calendar/answer/37100?hl=en)
- **Outlook:** Add calendar → Subscribe from web
  - [Import or subscribe to a calendar in Outlook.com or Outlook on the web | Microsoft Support](https://support.microsoft.com/en-us/outlook/import-or-subscribe-to-a-calendar-in-outlook-com-or-outlook-on-the-web)

The feed refreshes roughly every 6 hours.

## What's in the feed

- Events are **all-day**, dated in Japan time (JST).
- The source keeps one free-text note per day, so each event's text — including any showtimes (e.g. `13:30、18:30`) and `@venue` — appears in the event title exactly as written.
- When a day lists several events (separated by a blank line at the source), each becomes its own entry.

## Attribution

Schedule data belongs to its author and comes from the public [freecalend](https://freecalend.com/open/mem231613) page. This is an unofficial mirror.

## License

The project code is [MIT licensed](LICENSE) — the license covers the code only, not the schedule data, which remains the property of its author.

---

Building or contributing? See [CONTRIBUTING.md](CONTRIBUTING.md) for local development and [docs/design.md](docs/design.md) for the design.
