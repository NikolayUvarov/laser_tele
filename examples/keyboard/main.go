// Keyboard bot: /menu shows a message with buttons; pressing a button changes the message.
//
// Run: TG_API_KEY=<token> go run ./examples/keyboard
package main

import (
	"fmt"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/laser_tele_api"
)

func menu() laser_tele.InlineKeyboard {
	return laser_tele.InlineKeyboard{Keyboard: []laser_tele.Row{
		{laser_tele.AddButton("☕ Coffee", "coffee"), laser_tele.AddButton("🍵 Tea", "tea")},
		{{Text: "❌ Cancel", CallbackData: "cancel", Style: "danger"}},
		{{Text: "🌐 Website", URL: "https://core.telegram.org/bots"}},
	}}
}

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 2 * time.Second})

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		var err error
		switch update.Type() {
		case "message":
			if update.UpdateMessage.Text == "/menu" {
				err = laser_tele.SendKeyboard(update.UpdateMessage.Chat.ID, "What would you like?", menu())
			}
		case "callback_query":
			err = onButton(update.CallbackQuery)
		}
		if err != nil {
			fmt.Println("Error:", err)
		}
	})
}

func onButton(query laser_tele.CallbackQuery) error {
	chatID, messageID := query.Message.Chat.ID, query.Message.MessageID

	// the pressed button must be answered, otherwise Telegram shows progress on it
	if query.Data == "cancel" {
		if err := laser_tele.AnswerCallbackQueryWithConfig(query.ID, laser_tele.CallbackAnswerConfig{
			Text: "Order canceled", ShowAlert: true,
		}); err != nil {
			return err
		}
		// remove the buttons
		if err := laser_tele.EditMessageReplyMarkup(chatID, messageID, laser_tele.InlineKeyboard{Keyboard: []laser_tele.Row{}}); err != nil {
			return err
		}
		return laser_tele.EditMessageText(chatID, messageID, "Nothing ordered")
	}

	if err := laser_tele.AnswerCallbackQuery(query.ID, "You chose "+query.Data); err != nil {
		return err
	}
	return laser_tele.EditMessageText(chatID, messageID, "Your order: "+query.Data+". It will be ready in 5 minutes")
}
