// Inline bot: type "@your_bot some text" in any chat and choose how to send the text.
//
// Enable inline mode with /setinline in @BotFather,
// and /setinlinefeedback to get chosen_inline_result updates.
//
// Run: TG_API_KEY=<token> go run ./examples/inline
package main

import (
	"fmt"
	"strings"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/v2/laser_tele_api"
)

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: time.Second})

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		switch update.Type() {
		case "inline_query":
			if err := onInlineQuery(update.InlineQuery); err != nil {
				fmt.Println("Error:", err)
			}
		case "chosen_inline_result":
			result := update.ChosenInlineResult
			fmt.Println(result.From.FirstName, "sent result", result.ResultID, "for query", result.Query)
		}
	})
}

func onInlineQuery(query laser_tele.InlineQuery) error {
	text := query.Query
	if text == "" {
		// a button above the results opens the private chat with the bot
		return laser_tele.AnswerInlineQuery(query.ID, nil, laser_tele.InlineQueryConfig{
			ButtonText: "How to use the bot", ButtonStartParameter: "help",
		})
	}

	upper := laser_tele.NewInlineArticle("upper", "UPPER CASE", strings.ToUpper(text))
	upper["description"] = strings.ToUpper(text)
	lower := laser_tele.NewInlineArticle("lower", "lower case", strings.ToLower(text))
	lower["description"] = strings.ToLower(text)
	reversed := laser_tele.NewInlineArticle("reversed", "Reversed", reverse(text))
	reversed["description"] = reverse(text)
	// any result type from https://core.telegram.org/bots/api#inlinequeryresult can be built as a map
	photo := laser_tele.NewInlinePhoto("photo", "https://telegram.org/img/t_logo.png", "https://telegram.org/img/t_logo.png")
	photo["caption"] = text

	return laser_tele.AnswerInlineQuery(query.ID, []laser_tele.InlineQueryResult{upper, lower, reversed, photo},
		laser_tele.InlineQueryConfig{CacheTime: 30, IsPersonal: true})
}

func reverse(text string) string {
	runes := []rune(text)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
