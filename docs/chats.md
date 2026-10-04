# Groups, reactions and business accounts

## The bot in groups

Add the bot to a group like a user. By default bots are in privacy mode and get only commands and replies
to their messages; `/setprivacy` in @BotFather turns it off, then the bot gets all messages.
An administrator bot gets all messages and the updates below.

`my_chat_member` tells about changes of the bot itself: it was added to a group or removed,
became an administrator, or a user blocked it in a private chat:

```go
if update.Type() == "my_chat_member" {
	change := update.MyChatMember
	switch change.NewChatMember.Status {
	case "member", "administrator":
		fmt.Println("The bot is added to", change.Chat.Title)
	case "left", "kicked":
		fmt.Println("The bot is removed from", change.Chat.ID, "or blocked") // stop sending there
	}
}
```

When a group becomes a supergroup, its ID changes: the message has `MigrateToChatID`,
and sending to the old ID returns an `*APIError` with `MigrateToChatID`.

## Members

`chat_member` comes when a member joins, leaves, is promoted or restricted (the bot must be an administrator):

```go
if update.Type() == "chat_member" {
	change := update.ChatMember
	user := change.NewChatMember.User
	if change.OldChatMember.Status == "left" && change.NewChatMember.Status == "member" {
		laser_tele.SendMessage(change.Chat.ID, "Welcome, "+user.FirstName+"!")
	}
	if change.NewChatMember.Status == "administrator" {
		fmt.Println(user.FirstName, "is an administrator now, can pin:", change.NewChatMember.CanPinMessages)
	}
}
```

Statuses: `creator`, `administrator`, `member`, `restricted`, `left`, `kicked`.

## Join requests

For a group or channel with join requests (an invite link with approval, or "Approve new members"),
the bot with the right to invite users gets `chat_join_request`:

```go
if update.Type() == "chat_join_request" {
	request := update.ChatJoinRequest
	if strings.Contains(strings.ToLower(request.Bio), "spam") {
		laser_tele.DeclineChatJoinRequest(request.Chat.ID, request.From.ID)
	} else {
		laser_tele.ApproveChatJoinRequest(request.Chat.ID, request.From.ID)
	}
}
```

`UserChatID` is the private chat with the user: the bot can write there within 5 minutes after the request,
for example to ask questions before approving.

## Reactions

The bot puts a reaction on a message, `""` removes it. Only the emoji allowed for reactions can be used:
❤ 👍 👎 🔥 🥰 👏 😁 🤔 🤯 😱 🤬 😢 🎉 🤩 🤮 💩 🙏 👌 🕊 🤡 🥱 🥴 😍 🐳 ❤‍🔥 🌚 🌭 💯 🤣 ⚡ 🍌 🏆 💔 🤨 😐 🍓 🍾 💋 🖕 😈 😴 😭 🤓 👻 👨‍💻 👀 🎃 🙈 😇 😨 🤝 ✍ 🤗 🫡 🎅 🎄 ☃ 💅 🤪 🗿 🆒 💘 🙉 🦄 😘 💊 🙊 😎 👾 🤷‍♂ 🤷 🤷‍♀ 😡

```go
if err := laser_tele.SetMessageReaction(chatID, messageID, "🔥"); err != nil {
	fmt.Println(err)
}
```

An administrator bot gets reactions of users:

```go
switch update.Type() {
case "message_reaction":
	reaction := update.MessageReaction
	for _, r := range reaction.NewReaction {
		fmt.Println(reaction.User.FirstName, "reacted with", r.Emoji, "to message", reaction.MessageID)
	}
case "message_reaction_count":
	// anonymous reactions (in channels) come only as counts
	for _, count := range update.MessageReactionCount.Reactions {
		fmt.Println(count.Type.Emoji, count.TotalCount)
	}
}
```

Reactions of other bots are not sent. `ReactionType.Type` is `emoji`, `custom_emoji` (with `CustomEmojiID`) or `paid`.

## Boosts

```go
switch update.Type() {
case "chat_boost":
	boost := update.ChatBoost
	fmt.Println(boost.Boost.Source.User.FirstName, "boosted", boost.Chat.Title, "until", boost.Boost.ExpirationDate)
case "removed_chat_boost":
	fmt.Println("Boost", update.RemovedChatBoost.BoostID, "removed")
}
```

## Business accounts

An owner of a Telegram Business account can connect the bot in Settings > Telegram Business > Chatbots,
then the bot answers customers on behalf of the account.

```go
switch update.Type() {
case "business_connection":
	connection := update.BusinessConnection
	fmt.Println(connection.User.FirstName, "connected the bot:", connection.IsEnabled, "can reply:", connection.Rights.CanReply)

case "business_message":
	message := update.BusinessMessage
	if message.From.ID == message.Chat.ID { // a customer, not the owner
		_, err := laser_tele.SendMessageWithConfig(message.Chat.ID, "We will answer within an hour",
			laser_tele.MessageConfig{BusinessConnectionID: message.BusinessConnectionID})
		if err != nil {
			fmt.Println(err)
		}
	}

case "deleted_business_messages":
	fmt.Println("Deleted messages", update.DeletedBusinessMessages.MessageIDs)
}
```

Other methods are sent on behalf of the account with `Call` and the `business_connection_id` parameter.

## Managed bots and message generation

`managed_bot` comes when a bot managed by this bot is created or its token changes;
`GetManagedBotToken(update.ManagedBot.Bot.ID)` returns the token of the managed bot.

`stopped_message_generation` comes when the user asked to stop generating a message (a draft streamed by the bot):

```go
if update.Type() == "stopped_message_generation" {
	stop := update.StoppedMessageGeneration
	fmt.Println("Stop generating draft", stop.DraftID, "in chat", stop.Chat.ID)
}
```

See [examples/groups](../examples/groups/main.go) and [examples/business](../examples/business/main.go).

Next: [Errors and logs](errors-and-logs.md).
