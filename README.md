# progress-bot

A Telegram bot for tracking progress (gym, habits, measurements). Everything is done through inline buttons in a single message that keeps updating, so the chat never gets cluttered.

- **Folders** group things together (they can be nested to any depth).
- **Boxes** store entries, in two types:
  - ✅ **Check** - a habit, "done/not done" per day; shows a streak and a grid of the last 30 days.
  - 📊 **Measure** - a number per day (weight, seconds, reps); shows a PNG chart on demand.

## How it's built

- **Go 1.25**, no CGO
- [`go-telegram-bot-api`](https://github.com/go-telegram-bot-api/telegram-bot-api) - Telegram, long polling
- [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) - SQLite database (pure Go driver)
- [`gonum/plot`](https://github.com/gonum/plot) - charts
- [`godotenv`](https://github.com/joho/godotenv) - `.env` loading

```
main.go
# entry point, wires up dependencies

internal/bot/
# Telegram logic (menu, buttons, text input)

internal/storage/
# SQLite (folders, boxes, entries)

internal/chart/
# PNG chart generation
```

## How to use
Use the deployed version: [@mojprogress_bot](https://t.me/mojprogress_bot)


## License

[MIT](LICENSE) with an added attribution clause. You are free to use, modify and distribute it, but you **must keep the copyright notice and credit the original project**:

> Based on [progress-bot](https://github.com/maksimprivalov/progress-bot) - Maksim Privalov

Copyright (c) 2026 Maksim Privalov
