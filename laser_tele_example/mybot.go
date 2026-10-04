package main

import (
	"fmt"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/laser_tele_api"
)

func myOnUpdate(u laser_tele.Update) {
	fmt.Println("my callback Update:", u)
}
func main() {
	fmt.Println("Started")

	//laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{})
	//laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 5 * time.Second})
	//laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 5 * time.Second, APIKEY: "1234567890"})

	// myOnUpdate is called by the library for every new update
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 5 * time.Second, CallbackOnUpdate: myOnUpdate})

	// updates can also be received from the channel:
	// laser_tele.MakeChan()
	// go func() {
	// 	for update := range laser_tele.TgChan {
	// 		botLogic(update)
	// 	}
	// }()
	// laser_tele.LaserTeleRun(nil)
	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		fmt.Println("Callback: ", update)
		go botLogic(update)
	})
}

func botLogic(processedUpdate laser_tele.Update) {
	var err error

	if len(processedUpdate.UpdateMessage.Entities) > 0 && processedUpdate.UpdateMessage.Entities[0].Type == "bot_command" {
		switch processedUpdate.UpdateMessage.Text {
		case "/test_pic":
			fmt.Println("command /test_pic")
			err = laser_tele.SendPhoto(processedUpdate.UpdateMessage.Chat.ID, "Test image", "test.jpg")
		case "/test_text":
			fmt.Println("command /test_text")
			err = laser_tele.SendMessage(processedUpdate.UpdateMessage.Chat.ID, "This is a test text")
		case "/test_video":
			fmt.Println("command /test_video")
			err = laser_tele.SendVideo(processedUpdate.UpdateMessage.Chat.ID, "Test video", "test.mp4")
		case "/test_pdf":
			fmt.Println("command /test_pdf")
			err = laser_tele.SendDocument(processedUpdate.UpdateMessage.Chat.ID, "Test pdf", "test.pdf")
			// case "/send_file":
			// 	fmt.Println("command /send_file")
			// 	getFile(processedUpdate.UpdateMessage.Chat.ID)
		}

	} else if len(processedUpdate.UpdateMessage.Photo) > 0 && processedUpdate.UpdateMessage.Photo[0].FileID != "" {
		fmt.Println("Got photo")
		err = loadFile(processedUpdate.UpdateMessage.Chat.ID, processedUpdate.UpdateMessage.Photo[0].FileID)

	} else if processedUpdate.UpdateMessage.Video.FileID != "" {
		fmt.Println("Got Video")
		err = loadFile(processedUpdate.UpdateMessage.Chat.ID, processedUpdate.UpdateMessage.Video.FileID)

	} else if processedUpdate.UpdateMessage.Document.FileID != "" {
		fmt.Println("Got Document")
		err = loadFile(processedUpdate.UpdateMessage.Chat.ID, processedUpdate.UpdateMessage.Document.FileID)

	} else {
		err = laser_tele.SendMessage(processedUpdate.UpdateMessage.Chat.ID, fmt.Sprint(processedUpdate.UpdateMessage)+"\nID:"+fmt.Sprint(processedUpdate.UpdateID))
	}

	if err != nil {
		fmt.Println("Can't process update:", err)
	}
}

func loadFile(chatID int, fileID string) error {
	path, err := laser_tele.LoadFile(chatID, fileID)
	if err == nil {
		fmt.Println("File saved to", path)
	}
	return err
}
