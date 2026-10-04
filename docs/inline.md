# Inline mode

In inline mode the user types "@your_bot some text" in any chat, the bot offers results,
and the chosen result is sent to the chat on behalf of the user "via @your_bot".

Enable inline mode with `/setinline` in @BotFather. With `/setinlinefeedback` the bot also gets
`chosen_inline_result` updates about sent results.

## Answering a query

```go
laser_tele.LaserTeleRun(func(update laser_tele.Update) {
	if update.Type() != "inline_query" {
		return
	}
	query := update.InlineQuery // query.Query is the typed text

	results := []laser_tele.InlineQueryResult{
		laser_tele.NewInlineArticle("upper", "UPPER CASE", strings.ToUpper(query.Query)),
		laser_tele.NewInlineArticle("lower", "lower case", strings.ToLower(query.Query)),
	}
	if err := laser_tele.AnswerInlineQuery(query.ID, results, laser_tele.InlineQueryConfig{CacheTime: 10}); err != nil {
		fmt.Println(err)
	}
})
```

A query must be answered within some seconds, with up to 50 results. Each result has an ID unique within the answer.

## Results

| Function | Result |
|---|---|
| `NewInlineArticle(id, title, text)` | a text message |
| `NewInlinePhoto(id, photoURL, thumbnailURL)` | a JPEG photo by URL |
| `NewInlineCached(fileType, id, fileID, title)` | a file already in Telegram: `photo`, `gif`, `mpeg4_gif`, `sticker`, `document`, `video`, `voice` or `audio`; `title` is required for `document`, `video` and `voice` |

`InlineQueryResult` is a map: add optional fields of [the result type](https://core.telegram.org/bots/api#inlinequeryresult)
or build any other type of result:

```go
article := laser_tele.NewInlineArticle("1", "Weather", "It's sunny, +25°C")
article["description"] = "Sunny, +25°C"
article["thumbnail_url"] = "https://example.com/sun.png"
article["reply_markup"] = laser_tele.InlineKeyboard{Keyboard: []laser_tele.Row{{{Text: "Forecast", URL: "https://example.com"}}}}

venue := laser_tele.InlineQueryResult{
	"type": "venue", "id": "2", "title": "Our office", "address": "Main street 1",
	"latitude": 55.75, "longitude": 37.62,
}
laser_tele.AnswerInlineQuery(query.ID, []laser_tele.InlineQueryResult{article, venue}, laser_tele.InlineQueryConfig{})
```

## Options

| `InlineQueryConfig` field | Meaning |
|---|---|
| `CacheTime` | seconds Telegram may cache the results, 300 by default |
| `IsPersonal` | cache the results only for this user |
| `NextOffset` | the next page: Telegram sends it in `InlineQuery.Offset` when the user scrolls |
| `ButtonText`, `ButtonStartParameter` | a button above the results opening the private chat with the bot with `/start <parameter>` |

Paging:

```go
const pageSize = 20
offset, _ := strconv.Atoi(query.Offset) // "" for the first page
var results []laser_tele.InlineQueryResult
for i := offset; i < offset+pageSize && i < len(allItems); i++ {
	results = append(results, laser_tele.NewInlineArticle(strconv.Itoa(i), allItems[i], allItems[i]))
}
next := ""
if offset+pageSize < len(allItems) {
	next = strconv.Itoa(offset + pageSize)
}
laser_tele.AnswerInlineQuery(query.ID, results, laser_tele.InlineQueryConfig{NextOffset: next})
```

## Sent results

```go
if update.Type() == "chosen_inline_result" {
	result := update.ChosenInlineResult
	fmt.Println(result.From.FirstName, "sent", result.ResultID, "for", result.Query)
}
```

`InlineMessageID` is set if the result had an inline keyboard: the bot can edit such messages with `Call`
and `inline_message_id`.

## Guest messages

A `guest_message` update has `Message.GuestQueryID`. Answer it with a result:

```go
message := update.GuestMessage
err := laser_tele.AnswerGuestQuery(message.GuestQueryID, laser_tele.NewInlineArticle("1", "Answer", "Hello!"))
if err != nil {
	fmt.Println(err)
}
```

See [examples/inline](../examples/inline/main.go).

Next: [Payments](payments.md).
