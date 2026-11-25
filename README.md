# laser_tele

A Go library for Telegram bots. We started it for internal projects of our company, and it works for anyone who sends messages through Telegram.

Code Artisan: [https://github.com/LaserPes]

Visionary Director: [https://github.com/NikolayUvarov]

## Features

- Receiving all kinds of updates of the Telegram Bot API: messages, buttons, polls, inline queries, payments, reactions, chat members, business messages...
- Sending messages with formatting, photos, videos, documents, inline keyboards; downloading files from users
- Polls and quizzes, inline mode, payments in Telegram Stars and through providers, games, reactions, join requests, business accounts
- Several bots in one program
- Work through an HTTP or SOCKS5 proxy (`Proxy` in the config or the `TG_PROXY` environment variable)
- Errors carry Telegram's codes; logs rotate and leave out message texts; the token never gets into logs or errors
- `Call` for any other method of the Bot API

## Install

```
go get github.com/NikolayUvarov/laser_tele/v2/laser_tele_api
```

## Quick start

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
		switch update.Type() {
		case "message":
			err := laser_tele.SendMessage(update.UpdateMessage.Chat.ID, "You said: "+update.UpdateMessage.Text)
			if err != nil {
				fmt.Println("Can't send message:", err)
			}
		case "callback_query":
			laser_tele.AnswerCallbackQuery(update.CallbackQuery.ID, "Pressed "+update.CallbackQuery.Data)
		}
	})
}
```

Create a bot with [@BotFather](https://t.me/BotFather) and run the program with its token:

```
TG_API_KEY=123456789:AAH... go run .
```

The token is taken from `LaserTeleConfigT.APIKEY`, the `TG_API_KEY` environment variable or the `.APIKEY` file.

## Documentation

- [Guides](docs/README.md): [getting started](docs/getting-started.md), [updates](docs/updates.md),
  [sending messages, files and keyboards](docs/sending.md), [polls](docs/polls.md), [inline mode](docs/inline.md),
  [payments](docs/payments.md), [games](docs/games.md), [groups, reactions and business accounts](docs/chats.md),
  [errors and logs](docs/errors-and-logs.md)
- [Reference of all functions](docs/reference.md)
- [API on pkg.go.dev](https://pkg.go.dev/github.com/NikolayUvarov/laser_tele/v2/laser_tele_api) with examples

## Examples

Complete bots in [examples/](examples): [echo](examples/echo/main.go), [keyboard](examples/keyboard/main.go),
[files](examples/files/main.go), [polls](examples/polls/main.go), [inline](examples/inline/main.go),
[payments](examples/payments/main.go), [game](examples/game/main.go), [groups](examples/groups/main.go),
[business](examples/business/main.go), [several_bots](examples/several_bots/main.go), [channel](examples/channel/main.go),
and everything together in [laser_tele_example](laser_tele_example/mybot.go).

```
TG_API_KEY=123456789:AAH... go run ./examples/keyboard
```

## laser-mtproxy

[mtproxy/](mtproxy/README.md) is a separate program in this repository: an MTProto proxy server for Telegram
applications with a web interface, per-user secrets and connection limits. It also runs an HTTP CONNECT and SOCKS5
gateway, so bots built on this library can reach the Bot API through it:

```go
laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Proxy: "http://192.168.1.10:1080"})
```

## Versions

Versions are git tags `vX.Y.Z`. Since v2 the import path ends with `/v2`:
`github.com/NikolayUvarov/laser_tele/v2/laser_tele_api`. To update a program written for older commits,
change the import path and run `go get github.com/NikolayUvarov/laser_tele/v2/laser_tele_api@latest`.

## License

[Apache License 2.0](LICENSE)
