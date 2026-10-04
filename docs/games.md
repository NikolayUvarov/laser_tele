# Games

A Telegram game is an HTML5 page opened from a message with a game. Telegram keeps the high scores.
See [the Telegram guide](https://core.telegram.org/bots/games).

1. Create the game with `/newgame` in @BotFather, choose its short name, for example `race`.
2. Host the page of the game on your server.
3. The bot sends the game, the user presses Play, the bot answers with the URL of the page.
4. When the user finishes, the page sends the score to your server, the server saves it with `SetGameScore`.

## Sending the game

```go
sent, err := laser_tele.SendGame(chatID, "race")
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println("Game in message", sent.MessageID)
```

The message gets a Play button automatically. A custom keyboard must have the play button first,
send it with `Call`:

```go
keyboard := laser_tele.InlineKeyboard{Keyboard: []laser_tele.Row{
	{{Text: "🏎 Play", CallbackGame: &laser_tele.CallbackGame{}}},
	{{Text: "Rules", URL: "https://example.com/race/rules"}},
}}
_, err := laser_tele.Call("sendGame", map[string]interface{}{"chat_id": chatID, "game_short_name": "race", "reply_markup": keyboard})
if err != nil {
	fmt.Println(err)
}
```

## Opening the game

The Play button sends a `callback_query` with `GameShortName`. Answer it with the URL of the game;
add the IDs needed to save the score later:

```go
laser_tele.LaserTeleRun(func(update laser_tele.Update) {
	query := update.CallbackQuery
	if query.GameShortName != "race" {
		return
	}
	params := url.Values{}
	params.Set("user", strconv.Itoa(query.From.ID))
	if query.InlineMessageID != "" {
		params.Set("inline", query.InlineMessageID) // the game was sent in inline mode
	} else {
		params.Set("chat", strconv.Itoa(query.Message.Chat.ID))
		params.Set("message", strconv.Itoa(query.Message.MessageID))
	}
	laser_tele.AnswerCallbackQueryWithConfig(query.ID, laser_tele.CallbackAnswerConfig{
		URL: "https://example.com/race?" + params.Encode(),
	})
})
```

## Scores

`GameMessage` points to the message with the game: `ChatID` and `MessageID`, or `InlineMessageID`.

```go
game := laser_tele.GameMessage{ChatID: chatID, MessageID: messageID}

// a lower score than the current one is ignored unless force is true
if err := laser_tele.SetGameScore(userID, 1200, game, false); err != nil {
	fmt.Println(err)
}

scores, err := laser_tele.GetGameHighScores(userID, game)
if err != nil {
	fmt.Println(err)
	return
}
for _, row := range scores {
	fmt.Printf("%d. %s %d\n", row.Position, row.User.FirstName, row.Score)
}
```

`GetGameHighScores` returns the score of the user and of several neighbors in the table.
Check scores sent by the page on your server, users can fake requests.

See [examples/game](../examples/game/main.go): the bot with a small HTTP server receiving scores.

Next: [Groups, reactions and business accounts](chats.md).
