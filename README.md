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

`update.Type()` returns the kind of the update: `message`, `edited_message`, `channel_post`, `edited_channel_post`,
`callback_query` (an inline button was pressed), `my_chat_member` (for example, the user blocked the bot) or `""` for other kinds.
The fields of the update are `UpdateMessage`, `EditedMessage`, `ChannelPost`, `EditedChannelPost`, `CallbackQuery` and `MyChatMember`.
A message has text, caption, entities, photo, video, animation, audio, document, voice, video note, sticker, contact, location,
reply and forward info, new and left chat members and the inline keyboard.
Absent objects have empty fields (a message has a video if `Video.FileID != ""`), only `ReplyToMessage` and `Location` are nil if absent.

When the user pressed an inline button, answer it with `AnswerCallbackQuery(update.CallbackQuery.ID, "text")`,
otherwise Telegram shows progress on the button.

### Errors

The send functions (`SendMessage`, `SendKeyboard`, `EditMessageReplyMarkup`, `AnswerCallbackQuery`, `SendPhoto`, `SendVideo`, `SendDocument`) return an error.
If Telegram refused the request, it is `*laser_tele.APIError` with Telegram's `ErrorCode` and `Description`
(for example, 403 if the user blocked the bot).

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
