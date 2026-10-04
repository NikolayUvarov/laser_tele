// Echo bot: answers every text message with the same text.
//
// Run: TG_API_KEY=<token from @BotFather> go run ./examples/echo
package main

import (
	"fmt"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/laser_tele_api"
)

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 2 * time.Second})

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		if update.Type() != "message" || update.UpdateMessage.Text == "" {
			return
		}
		message := update.UpdateMessage
		if err := laser_tele.SendMessage(message.Chat.ID, message.Text); err != nil {
			fmt.Println("Can't send message:", err)
		}
	})
}
