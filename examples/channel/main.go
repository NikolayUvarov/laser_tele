// Updates processed by several workers: updates are read from a channel,
// so a slow answer to one user doesn't delay the others.
//
// Run: TG_API_KEY=<token> go run ./examples/channel
package main

import (
	"fmt"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/laser_tele_api"
)

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: time.Second})
	laser_tele.MakeChan() // must be called before LaserTeleRun

	for worker := 1; worker <= 4; worker++ {
		go func(worker int) {
			for update := range laser_tele.TgChan {
				process(worker, update)
			}
		}(worker)
	}

	// nil: updates go only to the channel
	laser_tele.LaserTeleRun(nil)
}

func process(worker int, update laser_tele.Update) {
	if update.Type() != "message" {
		return
	}
	message := update.UpdateMessage
	fmt.Println("Worker", worker, "processes message", message.MessageID)

	time.Sleep(3 * time.Second) // a slow job
	if err := laser_tele.SendMessage(message.Chat.ID, "Done: "+message.Text); err != nil {
		fmt.Println("Error:", err)
	}
}
