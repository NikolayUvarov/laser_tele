// Game bot: /game sends an HTML5 game, the "Play" button opens it, the game reports scores to this program.
//
//  1. Create a game with /newgame in @BotFather, its short name is GAME_SHORT_NAME.
//  2. Host the game page at GAME_URL. When the user finishes, the page calls
//     <this server>/score?user=<user>&chat=<chat>&message=<message>&inline=<inline>&score=<score>
//     with the parameters this bot added to GAME_URL.
//
// Run: TG_API_KEY=<token> GAME_SHORT_NAME=race GAME_URL=https://example.com/race go run ./examples/game
package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/laser_tele_api"
)

var gameShortName = os.Getenv("GAME_SHORT_NAME")
var gameURL = os.Getenv("GAME_URL")

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: time.Second})

	http.HandleFunc("/score", onScore)
	go http.ListenAndServe(":8080", nil)

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		var err error
		switch {
		case update.Type() == "message" && update.UpdateMessage.Text == "/game":
			_, err = laser_tele.SendGame(update.UpdateMessage.Chat.ID, gameShortName)
		case update.Type() == "message" && update.UpdateMessage.Text == "/top":
			err = sendTop(update.UpdateMessage)
		case update.CallbackQuery.GameShortName != "":
			err = openGame(update.CallbackQuery)
		}
		if err != nil {
			fmt.Println("Error:", err)
		}
	})
}

// openGame answers the "Play" button with the URL of the game, which knows the player and the message
func openGame(query laser_tele.CallbackQuery) error {
	params := url.Values{}
	params.Set("user", strconv.Itoa(query.From.ID))
	if query.InlineMessageID != "" {
		params.Set("inline", query.InlineMessageID) // the game was sent in inline mode
	} else {
		params.Set("chat", strconv.Itoa(query.Message.Chat.ID))
		params.Set("message", strconv.Itoa(query.Message.MessageID))
	}
	return laser_tele.AnswerCallbackQueryWithConfig(query.ID, laser_tele.CallbackAnswerConfig{URL: gameURL + "?" + params.Encode()})
}

// onScore receives the score from the game page and saves it in Telegram
func onScore(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	userID, _ := strconv.Atoi(query.Get("user"))
	score, _ := strconv.Atoi(query.Get("score"))
	chatID, _ := strconv.Atoi(query.Get("chat"))
	messageID, _ := strconv.Atoi(query.Get("message"))
	game := laser_tele.GameMessage{ChatID: chatID, MessageID: messageID, InlineMessageID: query.Get("inline")}

	// a real game must check that the score is not faked, e.g. sign the parameters
	if err := laser_tele.SetGameScore(userID, score, game, false); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Fprintln(w, "ok")
}

// sendTop shows the high scores of the game the command replies to
func sendTop(message laser_tele.Message) error {
	if message.ReplyToMessage == nil || message.ReplyToMessage.Game.Title == "" {
		return laser_tele.SendMessage(message.Chat.ID, "Reply with /top to a message with the game")
	}
	game := laser_tele.GameMessage{ChatID: message.Chat.ID, MessageID: message.ReplyToMessage.MessageID}
	scores, err := laser_tele.GetGameHighScores(message.From.ID, game)
	if err != nil {
		return err
	}
	lines := []string{"Top players:"}
	for _, row := range scores {
		lines = append(lines, fmt.Sprintf("%d. %s — %d", row.Position, row.User.FirstName, row.Score))
	}
	return laser_tele.SendMessage(message.Chat.ID, strings.Join(lines, "\n"))
}
