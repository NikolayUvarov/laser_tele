# Polls and quizzes

## Sending a poll

```go
sent, err := laser_tele.SendPoll(chatID, "Where shall we have lunch?", []string{"Pizza", "Sushi", "Burgers"},
	laser_tele.PollConfig{NotAnonymous: true, AllowsMultipleAnswers: true})
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println("Poll", sent.Poll.ID, "in message", sent.MessageID)
```

A poll has 1-12 options, the question is up to 300 characters. `PollConfig` fields are optional:

| Field | Meaning |
|---|---|
| `NotAnonymous` | votes are visible, the bot gets them as `poll_answer` updates (polls are anonymous by default) |
| `AllowsMultipleAnswers` | several options can be chosen |
| `Quiz`, `CorrectOptionIDs`, `Explanation` | a quiz, see below |
| `OpenPeriod` | seconds the poll is open after sending, 5-600 |
| `CloseDate` | unix time when the poll closes |
| `IsClosed` | send a closed poll |
| `ReplyMarkup` | an inline keyboard under the poll |

## A quiz

A quiz has correct answers: the user sees if they were right, `Explanation` is shown after a wrong answer.

```go
_, err := laser_tele.SendPoll(chatID, "What does `go f()` do?", []string{"Calls f", "Starts f in a goroutine", "Defers f"},
	laser_tele.PollConfig{
		Quiz:             true,
		CorrectOptionIDs: []int{1}, // 0-based indexes of the correct options
		Explanation:      "go starts a function in a new goroutine",
		NotAnonymous:     true,
		OpenPeriod:       30,
	})
if err != nil {
	fmt.Println(err)
}
```

## Votes

The bot gets votes only in non-anonymous polls it sent itself:

```go
laser_tele.LaserTeleRun(func(update laser_tele.Update) {
	switch update.Type() {
	case "poll_answer":
		answer := update.PollAnswer
		if len(answer.OptionIDs) == 0 {
			fmt.Println(answer.User.FirstName, "retracted the vote")
		} else {
			fmt.Println(answer.User.FirstName, "chose", answer.OptionIDs, "in poll", answer.PollID)
		}
	case "poll":
		// the new state of a poll sent by the bot: numbers of votes, closing
		poll := update.Poll
		for _, option := range poll.Options {
			fmt.Println(option.Text, option.VoterCount)
		}
	}
})
```

`PollAnswer.User.ID` is also the ID of the private chat with the user, the bot can write there
if the user started the bot. Votes of anonymous chat administrators have `VoterChat` instead of `User`.

## Closing a poll

`StopPoll` closes a poll sent by the bot and returns the results:

```go
poll, err := laser_tele.StopPoll(chatID, sent.MessageID)
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println(poll.TotalVoterCount, "votes")
for _, option := range poll.Options {
	fmt.Printf("%s: %d\n", option.Text, option.VoterCount)
}
```

A poll sent by a user comes in `Message.Poll` (`Poll.ID != ""`). See [examples/polls](../examples/polls/main.go).

Next: [Inline mode](inline.md).
