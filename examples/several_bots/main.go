// Two bots in one program: the news bot and the support bot, each with its own token, logs and downloads.
//
// Run: NEWS_BOT_KEY=<token 1> SUPPORT_BOT_KEY=<token 2> go run ./examples/several_bots
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/laser_tele_api"
)

func main() {
	news, err := laser_tele.NewBot(laser_tele.LaserTeleConfigT{
		APIKEY:      os.Getenv("NEWS_BOT_KEY"),
		Timeout:     5 * time.Second,
		LogDir:      "logs/news",
		DownloadDir: "files/news",
	})
	if err != nil {
		log.Fatal(err)
	}
	support, err := laser_tele.NewBot(laser_tele.LaserTeleConfigT{
		APIKEY:      os.Getenv("SUPPORT_BOT_KEY"),
		Timeout:     time.Second,
		LogDir:      "logs/support",
		DownloadDir: "files/support",
	})
	if err != nil {
		log.Fatal(err)
	}

	go news.Run(func(update laser_tele.Update) {
		if update.Type() == "message" {
			report(news.SendMessage(update.UpdateMessage.Chat.ID, "Today's news: the support bot works 24/7"))
		}
	})

	support.Run(func(update laser_tele.Update) {
		if update.Type() != "message" {
			return
		}
		message := update.UpdateMessage
		if document := message.Document; document.FileID != "" {
			path, err := support.LoadFile(document.FileID)
			report(err)
			fmt.Println("The support bot saved", path)
		}
		report(support.SendMessage(message.Chat.ID, "Your request is registered"))
	})
}

func report(err error) {
	if err != nil {
		fmt.Println("Error:", err)
	}
}
