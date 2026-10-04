// Business bot: answers customers on behalf of a Telegram Business account while its owner is away.
//
// The owner connects the bot in Telegram: Settings > Telegram Business > Chatbots.
//
// Run: TG_API_KEY=<token> go run ./examples/business
package main

import (
	"fmt"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/laser_tele_api"
)

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 2 * time.Second})

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		var err error
		switch update.Type() {
		case "business_connection":
			connection := update.BusinessConnection
			fmt.Println("Business account of", connection.User.FirstName, "enabled:", connection.IsEnabled,
				"can reply:", connection.Rights.CanReply)

		case "business_message":
			message := update.BusinessMessage
			// messages written by the owner of the account come too, answer only customers
			if message.From.ID != message.Chat.ID {
				return
			}
			_, err = laser_tele.SendMessageWithConfig(message.Chat.ID, "Thank you for your message! We will answer within an hour.",
				laser_tele.MessageConfig{BusinessConnectionID: message.BusinessConnectionID, ReplyToMessageID: message.MessageID})

		case "deleted_business_messages":
			deleted := update.DeletedBusinessMessages
			fmt.Println("Messages", deleted.MessageIDs, "were deleted in chat", deleted.Chat.ID)
		}
		if err != nil {
			fmt.Println("Error:", err)
		}
	})
}
