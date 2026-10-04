# Reference

All functions of the package, generated from the sources. Examples are in the [guides](README.md),
[examples/](../examples) and on [pkg.go.dev](https://pkg.go.dev/github.com/NikolayUvarov/laser_tele/v2/laser_tele_api).

```go
import laser_tele "github.com/NikolayUvarov/laser_tele/v2/laser_tele_api"
```

Package-level functions work with the default bot (`DoLaserTeleInit`). A `Bot` created by `NewBot` has a method
with the same name and parameters for each of them, marked ✓ in the column Bot; `bot.LoadFile` has no `chatID`,
`bot.Run` is `LaserTeleRun`, `bot.MakeChan` returns the channel.

## Functions

### Configuration and updates

| Function | Bot | Description |
|---|---|---|
| `DoLaserTeleInit(config LaserTeleConfigT)` |  | DoLaserTeleInit configures the default bot with config. It exits the program if the token is not found |
| `NewBot(config LaserTeleConfigT) (*Bot, error)` |  | NewBot creates a bot. If config.APIKEY is empty, it is taken from TG_API_KEY env or .APIKEY file, if config.Timeout is 0, it is taken from TIMEOUT env (seconds, default 10) |
| `LaserTeleRun(callback Callback)` | ✓ | LaserTeleRun requests updates of the default bot every Timeout and passes each of them to OnUpdateCallbackFunc, TgChan (if MakeChan was called) and callback. It never returns and exits the program if the token is not found |
| `bot.Run(callback Callback)` | ✓ | Run requests updates every Timeout and passes each of them like UpdateRequest. It never returns |
| `UpdateRequest(callback Callback)` | ✓ | UpdateRequest requests new updates of the default bot once and passes each of them to OnUpdateCallbackFunc, TgChan (if MakeChan was called) and callback |
| `MakeChan()` | ✓ | MakeChan creates the channel TgChan, to which updates of the default bot are sent. It must be called before LaserTeleRun. Updates must be read from TgChan, otherwise processing of updates is blocked |

### Messages

| Function | Bot | Description |
|---|---|---|
| `SendMessage(chatID int, text string) error` | ✓ | SendMessage sends a text message to the chat |
| `SendMessageWithConfig(chatID int, text string, config MessageConfig) (Message, error)` | ✓ | SendMessageWithConfig sends message with optional parameters and returns the sent message |
| `EditMessageText(chatID, messageID int, text string) error` | ✓ | EditMessageText changes text of a message sent by the bot |

### Keyboards and buttons

| Function | Bot | Description |
|---|---|---|
| `SendKeyboard(chatID int, text string, keyboard InlineKeyboard) error` | ✓ | SendKeyboard sends a message with text and an inline keyboard |
| `EditMessageReplyMarkup(chatID, messageID int, keyboard InlineKeyboard) error` | ✓ | EditMessageReplyMarkup changes the inline keyboard of a message sent by the bot, an empty keyboard removes it |
| `AddButton(text, callback string) Button` |  | AddButton creates a button, which sends callback (CallbackQuery.Data) to the bot when pressed |
| `AnswerCallbackQuery(callbackQueryID, text string) error` | ✓ | AnswerCallbackQuery must be called when the user pressed an inline button (update.CallbackQuery.ID != ""), otherwise Telegram shows progress on the button. text (optional) is shown to the user as a notification |
| `AnswerCallbackQueryWithConfig(callbackQueryID string, config CallbackAnswerConfig) error` | ✓ | AnswerCallbackQueryWithConfig answers the pressed inline button like AnswerCallbackQuery, with optional parameters |

### Files

| Function | Bot | Description |
|---|---|---|
| `SendPhoto(chatID int, text, photo string) error` | ✓ | SendPhoto uploads a local photo file to the chat, text is the caption |
| `SendVideo(chatID int, text, video string) error` | ✓ | SendVideo uploads a local video file to the chat, text is the caption |
| `SendDocument(chatID int, text, document string) error` | ✓ | SendDocument uploads a local file to the chat as a document, text is the caption |
| `LoadFile(chatID int, fileID string) (string, error)` | ✓ | LoadFile downloads a file sent by a user (its file_id) to DownloadDir and returns its path. chatID is not used |
| `FileDownload(reqString, filePath string) (resp *http.Response, data []byte, contentType string)` |  | FileDownload downloads the file by link reqString and saves it to filePath in DownloadDir of the default bot. On error resp is nil and data contains the error message |
| `StringToFile(fileName string, str string)` |  | StringToFile writes str to fileName, creating its directories |

### Polls

| Function | Bot | Description |
|---|---|---|
| `SendPoll(chatID int, question string, options []string, config PollConfig) (Message, error)` | ✓ | SendPoll sends a poll with 1-12 options and returns the sent message (its Poll.ID identifies the poll) |
| `StopPoll(chatID, messageID int) (Poll, error)` | ✓ | StopPoll closes the poll sent by the bot and returns its final results |

### Inline mode

| Function | Bot | Description |
|---|---|---|
| `AnswerInlineQuery(inlineQueryID string, results []InlineQueryResult, config InlineQueryConfig) error` | ✓ | AnswerInlineQuery sends up to 50 results for the inline query |
| `AnswerGuestQuery(guestQueryID string, result InlineQueryResult) error` | ✓ | AnswerGuestQuery answers a guest message (Message.GuestQueryID) with result |
| `NewInlineArticle(id, title, messageText string) InlineQueryResult` |  | NewInlineArticle creates a result, which sends a text message |
| `NewInlinePhoto(id, photoURL, thumbnailURL string) InlineQueryResult` |  | NewInlinePhoto creates a result, which sends a JPEG photo by URL |
| `NewInlineCached(fileType, id, fileID, title string) InlineQueryResult` |  | NewInlineCached creates a result, which sends a file already stored on Telegram servers by its file_id. fileType is "photo", "gif", "mpeg4_gif", "sticker", "document", "video", "voice" or "audio", title is required for "document", "video" and "voice" |

### Payments

| Function | Bot | Description |
|---|---|---|
| `SendInvoice(chatID int, invoice InvoiceConfig) (Message, error)` | ✓ | SendInvoice sends an invoice and returns the sent message |
| `CreateInvoiceLink(invoice InvoiceConfig) (string, error)` | ✓ | CreateInvoiceLink creates a link to pay the invoice, it can be sent anywhere |
| `AnswerShippingQuery(shippingQueryID string, options []ShippingOption, errorMessage string) error` | ✓ | AnswerShippingQuery answers the shipping query with options of delivery. errorMessage (shown to the user) means that the delivery to the address is impossible |
| `AnswerPreCheckoutQuery(preCheckoutQueryID, errorMessage string) error` | ✓ | AnswerPreCheckoutQuery confirms the payment (errorMessage "") or cancels it with errorMessage shown to the user. It must be called within 10 seconds after the query is received |
| `RefundStarPayment(userID int, telegramPaymentChargeID string) error` | ✓ | RefundStarPayment returns Telegram Stars of the payment (SuccessfulPayment.TelegramPaymentChargeID) to the user |
| `EditUserStarSubscription(userID int, telegramPaymentChargeID string, isCanceled bool) error` | ✓ | EditUserStarSubscription cancels (isCanceled true) or re-enables the subscription of the user paid in Telegram Stars |

### Games

| Function | Bot | Description |
|---|---|---|
| `SendGame(chatID int, gameShortName string) (Message, error)` | ✓ | SendGame sends the game created in @BotFather and returns the sent message |
| `SetGameScore(userID, score int, message GameMessage, force bool) error` | ✓ | SetGameScore sets the score of the user in the game. A score lower than the current one is set only with force |
| `GetGameHighScores(userID int, message GameMessage) ([]GameHighScore, error)` | ✓ | GetGameHighScores returns the high scores of the user and several of their neighbors in the game |

### Reactions and chats

| Function | Bot | Description |
|---|---|---|
| `SetMessageReaction(chatID, messageID int, emoji string) error` | ✓ | SetMessageReaction sets the reaction of the bot to a message, emoji "" removes it. Only emoji allowed for reactions can be used: "👍", "👎", "❤", "🔥", "🎉", "👏"... |
| `ApproveChatJoinRequest(chatID, userID int) error` | ✓ | ApproveChatJoinRequest lets the user (ChatJoinRequest.From.ID) join the chat |
| `DeclineChatJoinRequest(chatID, userID int) error` | ✓ | DeclineChatJoinRequest rejects the request of the user (ChatJoinRequest.From.ID) to join the chat |
| `GetManagedBotToken(botUserID int) (string, error)` | ✓ | GetManagedBotToken returns the token of the bot (ManagedBotUpdated.Bot.ID) managed by this bot |

### Any other method

| Function | Bot | Description |
|---|---|---|
| `Call(method string, params interface{}) (json.RawMessage, error)` | ✓ | Call calls any Bot API method (https://core.telegram.org/bots/api#available-methods) with params (a struct or a map, sent as JSON) and returns the "result" field of the response. Use it for methods that have no own function in the library |

### Update and APIError methods

| Method | Description |
|---|---|
| `update.Type() string` | Type returns the kind of the update, the name of its field in Telegram Bot API: "message", "callback_query", "poll", "inline_query", "pre_checkout_query"... (see AllUpdateTypes) |
| `err.Error() string` | Error returns the description of the error with the method and the code |

## Types

Fields of the types are described in the sources and on pkg.go.dev; types of the Bot API objects
(`User`, `Chat`, `PhotoSize`, `Video`...) have the fields of [the Bot API](https://core.telegram.org/bots/api#available-types).

### Configuration

| Type | Description |
|---|---|
| `LaserTeleConfigT` | LaserTeleConfigT is the configuration of a bot for NewBot and DoLaserTeleInit. All fields are optional |
| `LogMode` | LogMode sets what is written to log files |
| `Bot` | Bot is a Telegram bot, it must be created with NewBot. Several bots with different tokens can work in one program. Methods sending messages can be called from several goroutines |
| `Callback` | Callback processes an update |

### Options of functions

| Type | Description |
|---|---|
| `MessageConfig` | MessageConfig contains optional parameters of a message for SendMessageWithConfig |
| `CallbackAnswerConfig` | CallbackAnswerConfig contains optional parameters of AnswerCallbackQueryWithConfig |
| `PollConfig` | PollConfig contains optional parameters of a poll for SendPoll |
| `InlineQueryConfig` | InlineQueryConfig contains optional parameters of AnswerInlineQuery |
| `InlineQueryResult` | InlineQueryResult is one result for AnswerInlineQuery or AnswerGuestQuery. Create it with NewInlineArticle, NewInlinePhoto or NewInlineCached and add optional fields, e.g. result["description"] = "...", or fill the fields of any result type from https://core.telegram.org/bots/api#inlinequeryresult |
| `InvoiceConfig` | InvoiceConfig describes an invoice for SendInvoice and CreateInvoiceLink. For payments in Telegram Stars use Currency "XTR", empty ProviderToken and exactly one price |
| `LabeledPrice` | LabeledPrice is a part of the price, e.g. the goods, the delivery, the tax |
| `ShippingOption` | ShippingOption is a way of delivery for AnswerShippingQuery |
| `GameMessage` | GameMessage is the message with a game: ChatID and MessageID, or InlineMessageID (CallbackQuery.InlineMessageID) for a game sent in inline mode |

### Keyboards

| Type | Description |
|---|---|
| `InlineKeyboard` | InlineKeyboard is a keyboard attached to a message, see SendKeyboard |
| `Row` | Row is a row of buttons of an inline keyboard |
| `Button` | Button of an inline keyboard. Set one of the fields after Text |
| `CallbackGame` | CallbackGame is a placeholder for Button.CallbackGame |

### Updates

| Type | Description |
|---|---|
| `Update` | Update is an incoming update. Only the field of its kind (see Type) is filled |
| `Message` | Message is a message of any kind: text, photo, poll, payment, service message... |
| `CallbackQuery` | CallbackQuery is sent when the user pressed an inline button. It must be answered with AnswerCallbackQuery, otherwise Telegram shows progress on the button |
| `InlineQuery` | InlineQuery is sent when the user typed "@your_bot query" in any chat. Inline mode must be enabled for the bot with /setinline in @BotFather |
| `ChosenInlineResult` | ChosenInlineResult is sent when the user chose a result of an inline query. These updates must be enabled for the bot with /setinlinefeedback in @BotFather |
| `Poll` | Poll is a poll or a quiz (Message.Poll), also sent as a poll update when its state changes |
| `PollAnswer` | PollAnswer is sent when a user voted in a non-anonymous poll sent by the bot |
| `ShippingQuery` | ShippingQuery is sent for invoices with IsFlexible, answer it with AnswerShippingQuery |
| `PreCheckoutQuery` | PreCheckoutQuery is sent before the payment, answer it with AnswerPreCheckoutQuery within 10 seconds |
| `MessageReactionUpdated` | MessageReactionUpdated is sent when a user changed their reaction to a message |
| `MessageReactionCountUpdated` | MessageReactionCountUpdated is sent when anonymous reactions to a message were changed |
| `ChatMemberUpdated` | ChatMemberUpdated is sent when status of a member in a chat is changed, e.g. the user blocked the bot (NewChatMember.Status == "kicked") or the bot was added to a group |
| `ChatJoinRequest` | ChatJoinRequest is sent when a user asked to join the chat, answer it with ApproveChatJoinRequest or DeclineChatJoinRequest. The bot must be an administrator with can_invite_users right |
| `ChatBoostUpdated` | ChatBoostUpdated is sent when a boost was added to a chat or changed |
| `ChatBoostRemoved` | ChatBoostRemoved is sent when a boost was removed from a chat |
| `BusinessConnection` | BusinessConnection is sent when the bot was connected to or disconnected from a business account. Messages of the account come as business_message updates, answer them with SendMessageWithConfig and MessageConfig.BusinessConnectionID |
| `BusinessMessagesDeleted` | BusinessMessagesDeleted is sent when messages were deleted in a connected business account |
| `PaidMediaPurchased` | PaidMediaPurchased is sent when a user bought paid media with a payload sent by the bot |
| `BotSubscriptionUpdated` | BotSubscriptionUpdated is sent when a payment subscription of a user is changed |
| `ManagedBotUpdated` | ManagedBotUpdated is sent when a bot managed by this bot was created or its token or owner was changed, get its token with GetManagedBotToken |
| `MessageGenerationStopped` | MessageGenerationStopped is sent when the user asked the bot to stop generation of a message |

### Errors

| Type | Description |
|---|---|
| `APIError` | APIError is returned when Telegram refused the request, e.g. ErrorCode 403 if the bot was blocked by the user |

## Variables and constants

| Name | Description |
|---|---|
| `APIKEY` | APIKEY is the token of the default bot |
| `AllUpdateTypes` | AllUpdateTypes are all kinds of updates, the bot receives them by default (LaserTeleConfigT.AllowedUpdates) |
| `OnUpdateCallbackFunc` | OnUpdateCallbackFunc is called for every new update of the default bot (CallbackOnUpdate of the config) |
| `TgChan` | TgChan is the channel for updates of the default bot, it is created by MakeChan |
| `LogWithoutContent` | LogWithoutContent (default): requests and responses are logged without texts of messages |
| `LogFull` | LogFull: full requests and responses with texts of messages are logged, for debugging |
| `LogOff` | LogOff: nothing is logged |
