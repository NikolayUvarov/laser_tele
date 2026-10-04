// Files bot: saves photos, videos, documents and voice notes sent by users to downloadedFiles/
// and sends a document back by the /report command.
//
// Run: TG_API_KEY=<token> go run ./examples/files
package main

import (
	"fmt"
	"os"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/laser_tele_api"
)

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 2 * time.Second, DownloadDir: "downloadedFiles"})

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		if update.Type() != "message" {
			return
		}
		if err := onMessage(update.UpdateMessage); err != nil {
			fmt.Println("Error:", err)
		}
	})
}

func onMessage(message laser_tele.Message) error {
	chatID := message.Chat.ID

	if message.Text == "/report" {
		if err := os.WriteFile("report.txt", []byte("Report created at "+time.Now().Format(time.RFC1123)), 0644); err != nil {
			return err
		}
		return laser_tele.SendDocument(chatID, "Your report", "report.txt")
	}

	var fileID string
	switch {
	case len(message.Photo) > 0:
		// sizes of the photo, the last one is the largest
		fileID = message.Photo[len(message.Photo)-1].FileID
	case message.Video.FileID != "":
		fileID = message.Video.FileID
	case message.Document.FileID != "":
		fileID = message.Document.FileID
	case message.Voice.FileID != "":
		fileID = message.Voice.FileID
	default:
		return laser_tele.SendMessage(chatID, "Send me a photo, video, document or voice note, or /report")
	}

	// files up to 20 MB can be downloaded by bots
	path, err := laser_tele.LoadFile(chatID, fileID)
	if err != nil {
		return err
	}
	text := "Saved to " + path
	if message.Caption != "" {
		text += " with caption: " + message.Caption
	}
	return laser_tele.SendMessage(chatID, text)
}
