package main

import (
	"fmt"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/laser_tele_api"
)

func myOnUpdate(u laser_tele.Update) {
	fmt.Println("my callback Update:", u.UpdateID, u.Type())
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
		fmt.Println("Callback: ", update.UpdateID, update.Type())
		go botLogic(update)
	})
}

func botLogic(update laser_tele.Update) {
	var err error

	switch update.Type() {
	case "message":
		err = processMessage(update)
	case "edited_message":
		err = laser_tele.SendMessage(update.EditedMessage.Chat.ID, "Message was edited: "+update.EditedMessage.Text)
	case "callback_query":
		// the pressed button must be answered, otherwise Telegram shows progress on it
		query := update.CallbackQuery
		err = laser_tele.AnswerCallbackQuery(query.ID, "You pressed "+query.Data)
		if err == nil {
			err = laser_tele.SendMessage(query.Message.Chat.ID, "Button "+query.Data+" was pressed")
		}
	default:
		fmt.Println("Skipped update of type", update.Type())
	}

	if err != nil {
		fmt.Println("Can't process update:", err)
	}
}

func processMessage(update laser_tele.Update) error {
	message := update.UpdateMessage
	chatID := message.Chat.ID

	if len(message.Entities) > 0 && message.Entities[0].Type == "bot_command" {
		switch message.Text {
		case "/test_pic":
			fmt.Println("command /test_pic")
			return laser_tele.SendPhoto(chatID, "Test image", "test.jpg")
		case "/test_text":
			fmt.Println("command /test_text")
			return laser_tele.SendMessage(chatID, "This is a test text")
		case "/test_video":
			fmt.Println("command /test_video")
			return laser_tele.SendVideo(chatID, "Test video", "test.mp4")
		case "/test_pdf":
			fmt.Println("command /test_pdf")
			return laser_tele.SendDocument(chatID, "Test pdf", "test.pdf")
		case "/test_keyboard":
			fmt.Println("command /test_keyboard")
			keyboard := laser_tele.InlineKeyboard{Keyboard: []laser_tele.Row{
				{laser_tele.AddButton("Yes", "yes"), laser_tele.AddButton("No", "no")},
			}}
			return laser_tele.SendKeyboard(chatID, "Choose", keyboard)
		}
		return nil
	}

	switch {
	case len(message.Photo) > 0:
		fmt.Println("Got photo")
		// sizes of the photo, the last one is the largest
		return loadFile(chatID, message.Photo[len(message.Photo)-1].FileID)
	case message.Video.FileID != "":
		fmt.Println("Got Video")
		return loadFile(chatID, message.Video.FileID)
	case message.Document.FileID != "":
		fmt.Println("Got Document")
		return loadFile(chatID, message.Document.FileID)
	case message.Voice.FileID != "":
		fmt.Println("Got Voice")
		return loadFile(chatID, message.Voice.FileID)
	}

	info := fmt.Sprintf("Text: %s\nFrom: %s %s (@%s)\nChat: %d %s\nID: %d",
		message.Text, message.From.FirstName, message.From.LastName, message.From.Username,
		chatID, message.Chat.Type, update.UpdateID)
	return laser_tele.SendMessage(chatID, info)
}

func loadFile(chatID int, fileID string) error {
	path, err := laser_tele.LoadFile(chatID, fileID)
	if err == nil {
		fmt.Println("File saved to", path)
	}
	return err
}
