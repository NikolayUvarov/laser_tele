The development of this library was initiated for internal company projects, but it can be useful for anyone using Telegram to send messages. The library was designed to be user-friendly and accessible.

Code Artisan: [https://github.com/LaserPes]

Visionary Director: [https://github.com/NikolayUvarov]


## Usage

```
go get github.com/NikolayUvarov/laser_tele/laser_tele_api
```

```go
import laser_tele "github.com/NikolayUvarov/laser_tele/laser_tele_api"

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 5 * time.Second})
	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		err := laser_tele.SendMessage(update.UpdateMessage.Chat.ID, "Got: "+update.UpdateMessage.Text)
		if err != nil {
			fmt.Println("Can't send message:", err)
		}
	})
}
```

The bot token is taken from `LaserTeleConfigT.APIKEY`, the `TG_API_KEY` environment variable or the `.APIKEY` file (in this order).
The polling interval is taken from `LaserTeleConfigT.Timeout` or the `TIMEOUT` environment variable (seconds, default 10).
`LaserTeleConfigT.CallbackOnUpdate` (optional) is called for every new update, in addition to the callback passed to `LaserTeleRun`.

### Updates

`update.Type()` returns the kind of the update, the name of its field in the Bot API:
`message`, `edited_message`, `callback_query`, `inline_query`, `poll_answer`, `pre_checkout_query`, `message_reaction`...
All kinds are listed in `laser_tele.AllUpdateTypes`, the bot receives all of them by default
(`LaserTeleConfigT.AllowedUpdates` limits them). Only the field of the update's kind is filled:
`UpdateMessage`, `EditedMessage`, `CallbackQuery`, `InlineQuery`, `Poll`, `PollAnswer`, `PreCheckoutQuery`, `MessageReaction`...

Absent objects have empty fields (a message has a video if `Video.FileID != ""`), only `ReplyToMessage`, `PinnedMessage` and `Location` are nil if absent.
`update.Raw` is the update as received from Telegram, any field not described in the library can be read from it with `json.Unmarshal`.

When the user pressed an inline button, answer it with `AnswerCallbackQuery(update.CallbackQuery.ID, "text")`,
otherwise Telegram shows progress on the button.

| Feature | Updates | Functions | Requirements |
|---|---|---|---|
| Polls and quizzes | `poll`, `poll_answer` | `SendPoll`, `StopPoll` | votes come only for non-anonymous polls sent by the bot (`PollConfig.NotAnonymous`) |
| Inline mode | `inline_query`, `chosen_inline_result` | `AnswerInlineQuery`, `NewInlineArticle`, `NewInlinePhoto`, `NewInlineCached` | `/setinline` in @BotFather; `/setinlinefeedback` for `chosen_inline_result` |
| Payments | `shipping_query`, `pre_checkout_query`, `purchased_paid_media`, `subscription`, `Message.SuccessfulPayment` | `SendInvoice`, `CreateInvoiceLink`, `AnswerShippingQuery`, `AnswerPreCheckoutQuery` (within 10 seconds), `RefundStarPayment`, `EditUserStarSubscription` | Telegram Stars: currency `XTR`, no provider; other currencies: provider token from @BotFather |
| Games | `callback_query` with `GameShortName` | `SendGame`, `SetGameScore`, `GetGameHighScores`, `AnswerCallbackQueryWithConfig` (URL of the game) | `/newgame` in @BotFather |
| Reactions | `message_reaction`, `message_reaction_count` | `SetMessageReaction` | the bot is an administrator in the chat |
| Chat members | `my_chat_member`, `chat_member`, `chat_join_request`, `chat_boost`, `removed_chat_boost` | `ApproveChatJoinRequest`, `DeclineChatJoinRequest` | the bot is an administrator (except `my_chat_member`) |
| Business accounts | `business_connection`, `business_message`, `edited_business_message`, `deleted_business_messages` | `SendMessageWithConfig` with `BusinessConnectionID` | the bot is connected to a business account |
| Guest messages | `guest_message` | `AnswerGuestQuery` | |
| Managed bots | `managed_bot` | `GetManagedBotToken` | |
| Message generation | `stopped_message_generation` | | |

`SendMessageWithConfig` sends a message with parse mode, reply, topic, keyboard and other options and returns the sent message,
`EditMessageText` changes it. Any Bot API method without its own function can be called with
`Call("methodName", params)`, which returns the `result` field of the response.

### Errors

All functions sending requests return an error.
If Telegram refused the request, it is `*laser_tele.APIError` with Telegram's `ErrorCode` and `Description`
(for example, 403 if the user blocked the bot); `RetryAfter` is the number of seconds to wait after 429 Too Many Requests.

`LoadFile` downloads a file from a user's message to `downloadedFiles/` (`LaserTeleConfigT.DownloadDir`) and returns its path and an error.

### Logs

Requests and responses are written to `<name>.log` files in the current directory (`LaserTeleConfigT.LogDir`).
By default the logs don't contain texts of messages (`LogMode: laser_tele.LogWithoutContent`),
`laser_tele.LogFull` adds them for debugging, `laser_tele.LogOff` turns the logs off.
A log file larger than `LogMaxSize` (default 10 MB) is renamed to `<name>.log.1`, the previous one is deleted.
The bot token never gets to the logs.

### Several bots

The package-level functions work with one default bot. To run several bots in one program use `NewBot`:

```go
news, err := laser_tele.NewBot(laser_tele.LaserTeleConfigT{APIKEY: newsKey, LogDir: "logs/news", DownloadDir: "files/news"})
if err != nil {
	log.Fatal(err)
}
support, err := laser_tele.NewBot(laser_tele.LaserTeleConfigT{APIKEY: supportKey, LogDir: "logs/support", DownloadDir: "files/support"})
if err != nil {
	log.Fatal(err)
}

go news.Run(func(update laser_tele.Update) { news.SendMessage(update.UpdateMessage.Chat.ID, "news") })
support.Run(func(update laser_tele.Update) { support.SendMessage(update.UpdateMessage.Chat.ID, "support") })
```

A bot has the same methods as the package-level functions: `Run`, `UpdateRequest`, `MakeChan`, `SendMessage`, `LoadFile` and others.
Give each bot its own `LogDir` and `DownloadDir`, so their files don't mix.

A full example is in [laser_tele_example/mybot.go](laser_tele_example/mybot.go).
