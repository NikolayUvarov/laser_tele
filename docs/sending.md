# Sending messages, files and keyboards

All functions sending requests return an error, see [Errors and logs](errors-and-logs.md).

## Text

```go
err := laser_tele.SendMessage(chatID, "Hello!\nThe second line")
if err != nil {
	fmt.Println("Can't send:", err)
}
```

The text is sent as is: line breaks, `&`, `#`, `%` and emoji need no escaping. A message is up to 4096 characters.

`SendMessageWithConfig` adds formatting and other options and returns the sent message:

```go
sent, err := laser_tele.SendMessageWithConfig(chatID, "<b>Order 15</b> is ready, <a href=\"https://example.com/15\">details</a>",
	laser_tele.MessageConfig{
		ParseMode:           "HTML",    // or "MarkdownV2"
		ReplyToMessageID:    messageID, // reply to a message
		DisableNotification: true,      // without a sound
		ProtectContent:      true,      // can't be forwarded and saved
	})
if err != nil {
	fmt.Println("Can't send:", err)
	return
}
fmt.Println("Sent message", sent.MessageID)
```

With `ParseMode: "HTML"` escape `<`, `>` and `&` in texts from users with `html.EscapeString`.
`MessageThreadID` sends the message to a topic of a forum supergroup.

The bot can edit its messages:

```go
if err := laser_tele.EditMessageText(chatID, messageID, "Order 15 is delivered"); err != nil {
	fmt.Println("Can't edit:", err)
}
```

## Files

`SendPhoto`, `SendVideo` and `SendDocument` upload a local file, the text is the caption (up to 1024 characters):

```go
if err := laser_tele.SendPhoto(chatID, "Our office", "images/office.jpg"); err != nil {
	fmt.Println(err)
}
if err := laser_tele.SendDocument(chatID, "The report for May", "/var/reports/may.pdf"); err != nil {
	fmt.Println(err)
}
```

Bots upload files up to 50 MB. A file already in Telegram can be sent again by its `file_id` without uploading,
with `Call` (see [Reference](reference.md#any-other-method)):

```go
_, err := laser_tele.Call("sendPhoto", map[string]interface{}{"chat_id": chatID, "photo": message.Photo[0].FileID})
if err != nil {
	fmt.Println(err)
}
```

`LoadFile` downloads a file sent by a user to `DownloadDir` (`downloadedFiles` by default) and returns its path.
Bots download files up to 20 MB.

```go
if len(message.Photo) > 0 {
	largest := message.Photo[len(message.Photo)-1]
	path, err := laser_tele.LoadFile(chatID, largest.FileID)
	if err != nil {
		fmt.Println("Can't download:", err)
		return
	}
	fmt.Println("Saved to", path) // downloadedFiles/photos/file_12.jpg
}
```

See [examples/files](../examples/files/main.go).

## Inline keyboards

An inline keyboard is attached to a message. It is a list of rows, a row is a list of buttons:

```go
keyboard := laser_tele.InlineKeyboard{Keyboard: []laser_tele.Row{
	{laser_tele.AddButton("👍 Yes", "vote_yes"), laser_tele.AddButton("👎 No", "vote_no")},
	{{Text: "Open the site", URL: "https://example.com"}},
	{{Text: "Share", SwitchInlineQuery: "check this bot"}},
}}
if err := laser_tele.SendKeyboard(chatID, "Do you like it?", keyboard); err != nil {
	fmt.Println(err)
}
```

A button has `Text` and one action:

| Field | When the button is pressed |
|---|---|
| `CallbackData` (or `AddButton(text, data)`) | the bot gets a `callback_query` update with `Data`, up to 64 bytes |
| `URL` | the link is opened |
| `SwitchInlineQuery` | the user chooses a chat, "@your_bot text" is inserted there |
| `SwitchInlineQueryCurrentChat` | "@your_bot text" is inserted in the current chat |
| `CallbackGame` | the game sent by `SendGame` is launched, see [Games](games.md) |
| `Pay` | the invoice is paid, see [Payments](payments.md) |

`Style` colors the button: `"danger"` (red), `"success"` (green) or `"primary"` (blue).

### Pressed buttons

A pressed button with `CallbackData` sends a `callback_query` update. Always answer it,
otherwise Telegram shows progress on the button for some seconds:

```go
laser_tele.LaserTeleRun(func(update laser_tele.Update) {
	if update.Type() != "callback_query" {
		return
	}
	query := update.CallbackQuery
	// a notification at the top of the chat; "" answers without it
	laser_tele.AnswerCallbackQuery(query.ID, "Your vote is counted")

	// the message with the button
	chatID, messageID := query.Message.Chat.ID, query.Message.MessageID
	laser_tele.EditMessageText(chatID, messageID, "Thank you! You voted: "+query.Data)
})
```

`AnswerCallbackQueryWithConfig` shows the text in an alert window or opens a URL:

```go
laser_tele.AnswerCallbackQueryWithConfig(query.ID, laser_tele.CallbackAnswerConfig{Text: "Not enough money", ShowAlert: true})
```

`EditMessageReplyMarkup` changes the buttons of a sent message, an empty keyboard removes them:

```go
laser_tele.EditMessageReplyMarkup(chatID, messageID, laser_tele.InlineKeyboard{Keyboard: []laser_tele.Row{}})
```

`EditMessageText` without a keyboard removes the buttons too. To keep them, change the text with `Call`
and pass `reply_markup`. See [examples/keyboard](../examples/keyboard/main.go).

## Limits

Telegram limits how fast a bot sends messages: about one message per second in a chat, 20 messages per minute
in a group and about 30 messages per second in total. A request over the limit returns an error
with `RetryAfter`, see [Errors and logs](errors-and-logs.md).

Next: [Polls and quizzes](polls.md).
