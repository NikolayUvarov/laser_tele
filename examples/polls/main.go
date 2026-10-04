// Polls bot: /poll sends a poll, /quiz sends a quiz, /stop closes the last poll and shows the results.
// Votes come to the bot only for non-anonymous polls sent by it.
//
// Run: TG_API_KEY=<token> go run ./examples/polls
package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/v2/laser_tele_api"
)

// the last poll in every chat: chat ID -> message ID
var lastPolls = map[int]int{}
var lastPollsMutex sync.Mutex

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 2 * time.Second})

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		var err error
		switch update.Type() {
		case "message":
			err = onCommand(update.UpdateMessage)
		case "poll_answer":
			answer := update.PollAnswer
			if len(answer.OptionIDs) == 0 {
				fmt.Println(answer.User.FirstName, "retracted the vote in poll", answer.PollID)
			} else {
				fmt.Println(answer.User.FirstName, "voted for options", answer.OptionIDs, "in poll", answer.PollID)
			}
		case "poll":
			// a new state of a poll sent by the bot: the number of votes or closing
			poll := update.Poll
			fmt.Println("Poll", poll.ID, "has", poll.TotalVoterCount, "votes, closed:", poll.IsClosed)
		}
		if err != nil {
			fmt.Println("Error:", err)
		}
	})
}

func onCommand(message laser_tele.Message) error {
	chatID := message.Chat.ID
	var sent laser_tele.Message
	var err error

	switch message.Text {
	case "/poll":
		sent, err = laser_tele.SendPoll(chatID, "Which languages do you use?", []string{"Go", "Python", "JavaScript", "Other"},
			laser_tele.PollConfig{NotAnonymous: true, AllowsMultipleAnswers: true})
	case "/quiz":
		sent, err = laser_tele.SendPoll(chatID, "Which keyword starts a goroutine?", []string{"async", "go", "spawn"},
			laser_tele.PollConfig{Quiz: true, CorrectOptionIDs: []int{1}, Explanation: "go f() runs f in a new goroutine", NotAnonymous: true, OpenPeriod: 60})
	case "/stop":
		return stopPoll(chatID)
	default:
		return nil
	}
	if err != nil {
		return err
	}

	lastPollsMutex.Lock()
	lastPolls[chatID] = sent.MessageID
	lastPollsMutex.Unlock()
	return nil
}

func stopPoll(chatID int) error {
	lastPollsMutex.Lock()
	messageID, ok := lastPolls[chatID]
	delete(lastPolls, chatID)
	lastPollsMutex.Unlock()
	if !ok {
		return laser_tele.SendMessage(chatID, "There is no poll to stop, send /poll or /quiz")
	}

	poll, err := laser_tele.StopPoll(chatID, messageID)
	if err != nil {
		return err
	}
	results := []string{"Results of \"" + poll.Question + "\":"}
	for _, option := range poll.Options {
		results = append(results, fmt.Sprintf("%s: %d", option.Text, option.VoterCount))
	}
	return laser_tele.SendMessage(chatID, strings.Join(results, "\n"))
}
