# Receiving updates

Everything that happens with the bot comes as an `Update`: a message, a pressed button, a vote, a payment...
`update.Type()` tells the kind, and only the field of this kind is filled:

```go
laser_tele.LaserTeleRun(func(update laser_tele.Update) {
	switch update.Type() {
	case "message":
		fmt.Println("Message:", update.UpdateMessage.Text)
	case "edited_message":
		fmt.Println("Edited:", update.EditedMessage.Text)
	case "callback_query":
		fmt.Println("Button:", update.CallbackQuery.Data)
	default:
		fmt.Println("Skipped update of type", update.Type())
	}
})
```

## Kinds of updates

| `Type()` | Field | Sent when | Conditions |
|---|---|---|---|
| `message` | `UpdateMessage` | a new message of any kind | in groups bots in privacy mode get only commands and replies, see `/setprivacy` in @BotFather |
| `edited_message` | `EditedMessage` | a message was edited | |
| `channel_post` | `ChannelPost` | a new post in a channel | the bot is an administrator of the channel |
| `edited_channel_post` | `EditedChannelPost` | a post was edited | |
| `callback_query` | `CallbackQuery` | an inline button was pressed | answer with `AnswerCallbackQuery` |
| `inline_query` | `InlineQuery` | "@your_bot text" typed in a chat | `/setinline` in @BotFather, see [Inline mode](inline.md) |
| `chosen_inline_result` | `ChosenInlineResult` | a result of an inline query was sent | `/setinlinefeedback` in @BotFather |
| `poll` | `Poll` | a poll sent by the bot changed | see [Polls](polls.md) |
| `poll_answer` | `PollAnswer` | a user voted | only non-anonymous polls sent by the bot |
| `shipping_query` | `ShippingQuery` | the user entered the address | invoices with `IsFlexible`, see [Payments](payments.md) |
| `pre_checkout_query` | `PreCheckoutQuery` | the user confirmed the payment | answer within 10 seconds |
| `purchased_paid_media` | `PurchasedPaidMedia` | paid media sent by the bot was bought | |
| `subscription` | `Subscription` | a paid subscription was changed | |
| `message_reaction` | `MessageReaction` | a user changed a reaction | the bot is an administrator, see [Groups](chats.md) |
| `message_reaction_count` | `MessageReactionCount` | anonymous reactions changed | the bot is an administrator |
| `my_chat_member` | `MyChatMember` | the status of the bot changed: added to a group, blocked by a user... | |
| `chat_member` | `ChatMember` | the status of a member changed | the bot is an administrator |
| `chat_join_request` | `ChatJoinRequest` | a user asked to join | the bot is an administrator with the right to invite |
| `chat_boost` | `ChatBoost` | a chat was boosted | the bot is an administrator |
| `removed_chat_boost` | `RemovedChatBoost` | a boost was removed | the bot is an administrator |
| `business_connection` | `BusinessConnection` | the bot was connected to a business account | |
| `business_message` | `BusinessMessage` | a message in a connected business account | |
| `edited_business_message` | `EditedBusinessMessage` | it was edited | |
| `deleted_business_messages` | `DeletedBusinessMessages` | messages were deleted there | |
| `guest_message` | `GuestMessage` | a guest message | answer with `AnswerGuestQuery` |
| `managed_bot` | `ManagedBot` | a bot managed by this bot was created or changed | |
| `stopped_message_generation` | `StoppedMessageGeneration` | the user asked to stop generating a message | |

The bot receives all of them by default (`laser_tele.AllUpdateTypes`). To receive only some kinds,
list them in `LaserTeleConfigT.AllowedUpdates`.

If Telegram adds a new kind of updates, `Type()` returns its name and the update can be read from `update.Raw`,
the JSON received from Telegram:

```go
if update.Type() == "some_new_update" {
	var data struct {
		SomeNewUpdate struct {
			ID string `json:"id"`
		} `json:"some_new_update"`
	}
	if err := json.Unmarshal(update.Raw, &data); err == nil {
		fmt.Println(data.SomeNewUpdate.ID)
	}
}
```

`Raw` also gives access to any field the library doesn't describe.

## Messages

`Message` has the fields of [the Bot API](https://core.telegram.org/bots/api#message) used by bots most often.
Absent objects have empty fields: a message has a video if `Video.FileID != ""`.
Only `ReplyToMessage`, `PinnedMessage` and `Location` are pointers and are `nil` if absent.

```go
message := update.UpdateMessage
chatID := message.Chat.ID // where to answer

switch {
case message.Text != "":
	fmt.Println(message.From.FirstName, "wrote:", message.Text)
case len(message.Photo) > 0:
	largest := message.Photo[len(message.Photo)-1] // sizes of the photo, the last one is the largest
	fmt.Println("Photo", largest.Width, "x", largest.Height, "with caption", message.Caption)
case message.Document.FileID != "":
	fmt.Println("Document", message.Document.FileName, message.Document.FileSize, "bytes")
case message.Voice.FileID != "":
	fmt.Println("Voice note", message.Voice.Duration, "seconds")
case message.Location != nil:
	fmt.Println("Location", message.Location.Latitude, message.Location.Longitude)
case message.Contact.PhoneNumber != "":
	fmt.Println("Contact", message.Contact.FirstName, message.Contact.PhoneNumber)
case message.Sticker.FileID != "":
	fmt.Println("Sticker", message.Sticker.Emoji)
case message.Poll.ID != "":
	fmt.Println("Poll", message.Poll.Question)
case message.Dice.Emoji != "":
	fmt.Println("Dice", message.Dice.Emoji, "shows", message.Dice.Value)
}
fmt.Println("Chat", chatID, message.Chat.Type, message.Chat.Title)
```

Other useful fields:

- `Chat.Type`: `private`, `group`, `supergroup` or `channel`; `Chat.Title` for groups and channels.
- `From`: the sender, empty for posts in channels; `SenderChat` for messages on behalf of a chat.
- `ReplyToMessage`: the message this one replies to.
- `ForwardOrigin`: where a forwarded message comes from, `ForwardOrigin.Type != ""` for forwarded messages.
- `MediaGroupID`: the same for photos and videos sent as one album, they come as separate messages.
- `NewChatMembers`, `LeftChatMember`, `NewChatTitle`, `PinnedMessage`: service messages in groups.
- `MigrateToChatID`: the group became a supergroup with a new ID, use it for new messages.
- `SuccessfulPayment`: the user paid an invoice, see [Payments](payments.md).

### Commands

Commands like `/start` are marked by an entity of type `bot_command`.
In groups a command may have the username of the bot: `/start@your_bot`.

```go
func command(message laser_tele.Message) string {
	if len(message.Entities) == 0 || message.Entities[0].Type != "bot_command" || message.Entities[0].Offset != 0 {
		return ""
	}
	command := strings.Fields(message.Text)[0]
	return strings.SplitN(command, "@", 2)[0] // "/start@your_bot" -> "/start"
}
```

`/start` comes when the user opens the bot first time. A link `https://t.me/your_bot?start=promo` sends `/start promo`.
Register the list of commands with `/setcommands` in @BotFather to show them in the menu of the chat.

Offsets and lengths of entities are in UTF-16 code units, not bytes. To cut the text of an entity
with non-English letters or emoji, convert the text:

```go
func entityText(text string, entity laser_tele.MessageEntity) string {
	units := utf16.Encode([]rune(text))
	return string(utf16.Decode(units[entity.Offset : entity.Offset+entity.Length]))
}
```

## Updates that the bot didn't get

- The bot gets only updates of the last 24 hours. Updates sent while the bot was stopped longer are lost.
- After a start the bot skips updates sent before it, so it doesn't answer old messages.
- Only one program can receive updates of one bot at the same time.

Next: [Sending messages, files and keyboards](sending.md).
