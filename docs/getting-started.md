# Getting started

## Create a bot

1. Open [@BotFather](https://t.me/BotFather) in Telegram and send `/newbot`.
2. Choose a name and a username, BotFather gives the token like `123456789:AAH...`.
3. Keep the token secret: anybody with it controls the bot.

## Install

The library needs Go 1.18 or newer.

```
go get github.com/NikolayUvarov/laser_tele/v2/laser_tele_api
```

## The first bot

```go
package main

import (
	"fmt"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/v2/laser_tele_api"
)

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 2 * time.Second})

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		if update.Type() != "message" {
			return
		}
		message := update.UpdateMessage
		err := laser_tele.SendMessage(message.Chat.ID, "Hello, "+message.From.FirstName+"! You said: "+message.Text)
		if err != nil {
			fmt.Println("Can't send message:", err)
		}
	})
}
```

Run it with the token and write to the bot in Telegram:

```
TG_API_KEY=123456789:AAH... go run .
```

`DoLaserTeleInit` configures the bot, `LaserTeleRun` requests new updates every `Timeout` and calls the function
for each of them. It never returns.

Updates sent to the bot before it started are skipped, so the bot doesn't answer old messages after a restart.

## The token

The token is taken from the first of:

1. `LaserTeleConfigT.APIKEY`;
2. the `TG_API_KEY` environment variable;
3. the `.APIKEY` file in the current directory (spaces and line breaks around the token are ignored).

Don't commit the token: `.APIKEY` is in `.gitignore` of the repository, add it to yours too.
`DoLaserTeleInit` and `LaserTeleRun` stop the program if the token is not found.
The token never gets to the logs and to the errors returned by the library.

## Configuration

All fields of `LaserTeleConfigT` are optional:

| Field | Default | Meaning |
|---|---|---|
| `APIKEY` | `TG_API_KEY` or `.APIKEY` | the token of the bot |
| `Timeout` | `TIMEOUT` env (seconds) or 10 s | interval between requests of updates |
| `CallbackOnUpdate` | | called for every update, in addition to the function passed to `LaserTeleRun` |
| `AllowedUpdates` | `AllUpdateTypes` | kinds of updates the bot receives |
| `LogMode` | `LogWithoutContent` | what is written to the logs, see [Errors and logs](errors-and-logs.md) |
| `LogDir` | the current directory | directory of the log files |
| `LogMaxSize` | 10 MB | size of a log file when it is rotated |
| `DownloadDir` | `downloadedFiles` | directory for files downloaded by `LoadFile` |
| `Proxy` | `HTTPS_PROXY` env | proxy for requests to Telegram, see below |
| `APIURL` | `https://api.telegram.org` | address of the Bot API server |

```go
laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{
	Timeout:        time.Second,
	AllowedUpdates: []string{"message", "callback_query"},
	LogDir:         "logs",
	DownloadDir:    "files",
})
```

### Proxy

Requests to Telegram go through the proxy from `HTTPS_PROXY` and `NO_PROXY` environment variables, if they are set.
`Proxy` sets a proxy for one bot, the environment is ignored then:

```go
bot, err := laser_tele.NewBot(laser_tele.LaserTeleConfigT{
	Proxy: "socks5://user:password@127.0.0.1:1080",
})
```

HTTP (`http://user:password@host:3128`, or just `host:3128`), HTTPS and SOCKS5 (`socks5://` or `socks5h://`,
host names are resolved by the proxy) proxies are supported, the user and the password are optional.
The password doesn't get to errors. MTProxy is a proxy only for Telegram apps, it can't be used by bots.

### Bot API server

`APIURL` sends requests to another [Bot API server](https://github.com/tdlib/telegram-bot-api) instead of
`https://api.telegram.org`, e.g. a local one or a reverse proxy:

```go
bot, err := laser_tele.NewBot(laser_tele.LaserTeleConfigT{APIURL: "http://localhost:8081"})
```

A local server started with `--local` returns paths of files on its disk instead of links,
`LoadFile` can't download such files: read them from the disk.

Where Telegram is blocked, the server from [the fork with MTProxy support](https://github.com/NikolayUvarov/telegram-bot-api#mtproxy)
connects to Telegram through MTProxy servers, checks them and switches to a working one. The example
[mtproxy_collector](../examples/mtproxy_collector/main.go) collects such servers from channels for it.

## Ways to receive updates

The function passed to `LaserTeleRun` is called in the loop requesting updates: while it works, new updates wait.
For slow jobs start a goroutine:

```go
laser_tele.LaserTeleRun(func(update laser_tele.Update) {
	go func() {
		time.Sleep(5 * time.Second) // a slow job
		laser_tele.SendMessage(update.UpdateMessage.Chat.ID, "Done")
	}()
})
```

or read updates from a channel in several workers, see [examples/channel](../examples/channel/main.go):

```go
laser_tele.MakeChan() // before LaserTeleRun
for worker := 0; worker < 4; worker++ {
	go func() {
		for update := range laser_tele.TgChan {
			fmt.Println("Processing", update.UpdateID)
		}
	}()
}
laser_tele.LaserTeleRun(nil)
```

`CallbackOnUpdate` of the config, the channel and the function of `LaserTeleRun` all get every update, in this order.
To request updates once, without the loop, call `UpdateRequest`.

## The default bot and several bots

Package-level functions (`SendMessage`, `LaserTeleRun`...) work with the default bot configured by `DoLaserTeleInit`.
If `DoLaserTeleInit` wasn't called, the default bot is created on the first call with the token from the environment;
then functions return an error instead of stopping the program if the token is not found.
This makes the library handy for programs that only send notifications:

```go
func notify(text string) error {
	adminChatID := 137511897
	return laser_tele.SendMessage(adminChatID, text)
}
```

To run several bots in one program, create each with `NewBot`. A `Bot` has the same methods as the package-level functions:

```go
shop, err := laser_tele.NewBot(laser_tele.LaserTeleConfigT{APIKEY: os.Getenv("SHOP_BOT_KEY"), LogDir: "logs/shop", DownloadDir: "files/shop"})
if err != nil {
	log.Fatal(err)
}
admin, err := laser_tele.NewBot(laser_tele.LaserTeleConfigT{APIKEY: os.Getenv("ADMIN_BOT_KEY"), LogDir: "logs/admin", DownloadDir: "files/admin"})
if err != nil {
	log.Fatal(err)
}

go shop.Run(func(update laser_tele.Update) {
	if update.Type() == "message" {
		shop.SendMessage(update.UpdateMessage.Chat.ID, "Welcome to the shop!")
		admin.SendMessage(137511897, "New customer: "+update.UpdateMessage.From.FirstName)
	}
})
admin.Run(nil)
```

Give each bot its own `LogDir` and `DownloadDir`, otherwise their files mix.
`NewBot` returns an error instead of stopping the program. Updates of a `Bot` can be read from the channel
returned by `bot.MakeChan()`.

Next: [Receiving updates](updates.md).
