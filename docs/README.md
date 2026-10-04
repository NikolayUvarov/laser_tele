# laser_tele documentation

laser_tele is a Go library for Telegram bots. It covers the whole flow of a bot: receiving updates,
answering messages and buttons, files, polls, inline mode, payments in Telegram Stars, games, reactions,
group management and business accounts. Methods of the Bot API without their own function can be called with `Call`.

```
go get github.com/NikolayUvarov/laser_tele/v2/laser_tele_api
```

```go
import laser_tele "github.com/NikolayUvarov/laser_tele/v2/laser_tele_api"
```

## Guides

1. [Getting started](getting-started.md): the first bot, configuration, several bots
2. [Receiving updates](updates.md): kinds of updates, messages, commands, media
3. [Sending messages, files and keyboards](sending.md)
4. [Polls and quizzes](polls.md)
5. [Inline mode](inline.md)
6. [Payments](payments.md): Telegram Stars and payment providers
7. [Games](games.md)
8. [Groups, reactions and business accounts](chats.md)
9. [Errors and logs](errors-and-logs.md)
10. [Reference](reference.md): all functions

## Example bots

Every example is a complete program in [examples/](../examples). Run it with the token of your bot:

```
TG_API_KEY=123456:ABC-DEF go run ./examples/echo
```

| Example | What it shows |
|---|---|
| [echo](../examples/echo/main.go) | the smallest bot |
| [keyboard](../examples/keyboard/main.go) | buttons, answers to buttons, editing messages |
| [files](../examples/files/main.go) | downloading files from users, sending documents |
| [polls](../examples/polls/main.go) | polls, quizzes, votes, results |
| [inline](../examples/inline/main.go) | inline mode |
| [payments](../examples/payments/main.go) | a shop selling for Telegram Stars, refunds |
| [game](../examples/game/main.go) | an HTML5 game with high scores |
| [groups](../examples/groups/main.go) | join requests, greetings, reactions, boosts |
| [business](../examples/business/main.go) | answering on behalf of a business account |
| [several_bots](../examples/several_bots/main.go) | two bots in one program |
| [channel](../examples/channel/main.go) | processing updates in several goroutines |
| [laser_tele_example](../laser_tele_example/mybot.go) | everything together |

The API reference with examples is also on [pkg.go.dev](https://pkg.go.dev/github.com/NikolayUvarov/laser_tele/v2/laser_tele_api).
