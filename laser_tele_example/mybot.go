package main

import (
	"fmt"
	"time"

	laser_tele "../laser_tele_api"
)

func myOnUpdate(u laser_tele.Update) {
	fmt.Println("my callback Update:", u)
}
func main() {
	fmt.Println("Started")

	//laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{})
	//laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 5 * time.Second})
	//laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 5 * time.Second, APIKEY: "1234567890"})

	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 5 * time.Second, CallbackOnUpdate: myOnUpdate})

	// laser_tele.LaserTeleRun()
	// laser_tele.TgChan = make(chan laser_tele.Update)
	// go func() {
	// 	for update := range laser_tele.TgChan {
	// 		botLogic(update)
	// 	}
	// }()
	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		fmt.Println("Callback: ", update)
		go botLogic(update)
	})
}

func botLogic(processedUpdate laser_tele.Update) {

	if len(processedUpdate.UpdateMessage.Entities) > 0 && processedUpdate.UpdateMessage.Entities[0].Type == "bot_command" {
		//if defined onUpdateCallbackFunc function, call it
		if laser_tele.OnUpdateCallbackFunc != nil {
			laser_tele.OnUpdateCallbackFunc(processedUpdate)
		}

		switch processedUpdate.UpdateMessage.Text {
		case "/test_pic":
			fmt.Println("command /test_pic")
			laser_tele.SendPhoto(processedUpdate.UpdateMessage.Chat.ID, "Test image", "test.jpg")
		case "/test_text":
			fmt.Println("command /test_text")
			laser_tele.SendMessage(processedUpdate.UpdateMessage.Chat.ID, "This is a test text")
		case "/test_video":
			fmt.Println("command /test_video")
			laser_tele.SendVideo(processedUpdate.UpdateMessage.Chat.ID, "Test video", "test.mp4")
		case "/test_pdf":
			fmt.Println("command /test_pdf")
			laser_tele.SendDocument(processedUpdate.UpdateMessage.Chat.ID, "Test pdf", "test.pdf")
			// case "/send_file":
			// 	fmt.Println("command /send_file")
			// 	getFile(processedUpdate.UpdateMessage.Chat.ID)
		}

	} else if len(processedUpdate.UpdateMessage.Photo) > 0 && processedUpdate.UpdateMessage.Photo[0].FileID != "" {
		fmt.Println("Got photo")
		laser_tele.LoadFile(processedUpdate.UpdateMessage.Chat.ID, processedUpdate.UpdateMessage.Photo[0].FileID)

	} else if processedUpdate.UpdateMessage.Video.FileID != "" {
		fmt.Println("Got Video")
		laser_tele.LoadFile(processedUpdate.UpdateMessage.Chat.ID, processedUpdate.UpdateMessage.Video.FileID)

	} else if processedUpdate.UpdateMessage.Document.FileID != "" {
		fmt.Println("Got Document")
		laser_tele.LoadFile(processedUpdate.UpdateMessage.Chat.ID, processedUpdate.UpdateMessage.Document.FileID)

	} else {
		laser_tele.SendMessage(processedUpdate.UpdateMessage.Chat.ID, fmt.Sprint(processedUpdate.UpdateMessage)+"%0AID:"+fmt.Sprint(processedUpdate.UpdateID))
		//if defined onUpdateCallbackFunc function, call it
		if laser_tele.OnUpdateCallbackFunc != nil {
			laser_tele.OnUpdateCallbackFunc(processedUpdate)
		}
	}

}
