package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/v2/laser_tele_api"
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
		if query.GameShortName != "" {
			// the "Play" button of the game opens its page
			err = laser_tele.AnswerCallbackQueryWithConfig(query.ID, laser_tele.CallbackAnswerConfig{URL: os.Getenv("GAME_URL")})
			break
		}
		err = laser_tele.AnswerCallbackQuery(query.ID, "You pressed "+query.Data)
		if err == nil {
			err = laser_tele.SendMessage(query.Message.Chat.ID, "Button "+query.Data+" was pressed")
		}
	case "inline_query":
		// "@your_bot text" in any chat offers to send the text in upper case
		query := update.InlineQuery
		results := []laser_tele.InlineQueryResult{laser_tele.NewInlineArticle("upper", "Upper case", strings.ToUpper(query.Query))}
		err = laser_tele.AnswerInlineQuery(query.ID, results, laser_tele.InlineQueryConfig{CacheTime: 1})
	case "poll_answer":
		answer := update.PollAnswer
		err = laser_tele.SendMessage(answer.User.ID, fmt.Sprintf("You voted for options %v", answer.OptionIDs))
	case "pre_checkout_query":
		// must be answered within 10 seconds, "" confirms the payment
		err = laser_tele.AnswerPreCheckoutQuery(update.PreCheckoutQuery.ID, "")
	case "message_reaction":
		reaction := update.MessageReaction
		fmt.Println("Reaction to message", reaction.MessageID, "changed to", reaction.NewReaction)
	case "chat_join_request":
		request := update.ChatJoinRequest
		err = laser_tele.ApproveChatJoinRequest(request.Chat.ID, request.From.ID)
	case "business_message":
		// answer on behalf of the connected business account
		message := update.BusinessMessage
		_, err = laser_tele.SendMessageWithConfig(message.Chat.ID, "Thank you, we will answer soon",
			laser_tele.MessageConfig{BusinessConnectionID: message.BusinessConnectionID, ReplyToMessageID: message.MessageID})
	case "my_chat_member":
		member := update.MyChatMember
		fmt.Println("Status of the bot in chat", member.Chat.ID, "is", member.NewChatMember.Status)
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
		case "/test_poll":
			_, err := laser_tele.SendPoll(chatID, "Tea or coffee?", []string{"Tea", "Coffee"}, laser_tele.PollConfig{NotAnonymous: true})
			return err
		case "/test_quiz":
			_, err := laser_tele.SendPoll(chatID, "2 + 2 = ?", []string{"3", "4", "5"},
				laser_tele.PollConfig{Quiz: true, CorrectOptionIDs: []int{1}, Explanation: "Simple math"})
			return err
		case "/test_invoice":
			// payment in Telegram Stars, no payment provider is needed
			_, err := laser_tele.SendInvoice(chatID, laser_tele.InvoiceConfig{
				Title: "Coffee", Description: "A cup of coffee", Payload: "coffee-1", Currency: "XTR",
				Prices: []laser_tele.LabeledPrice{{Label: "Coffee", Amount: 1}},
			})
			return err
		case "/test_reaction":
			return laser_tele.SetMessageReaction(chatID, message.MessageID, "👍")
		case "/test_game":
			// the game must be created with /newgame in @BotFather
			_, err := laser_tele.SendGame(chatID, os.Getenv("GAME_SHORT_NAME"))
			return err
		case "/test_dice":
			// any Bot API method can be called with Call
			_, err := laser_tele.Call("sendDice", map[string]interface{}{"chat_id": chatID, "emoji": "🎲"})
			return err
		}
		return nil
	}

	if message.SuccessfulPayment.Currency != "" {
		payment := message.SuccessfulPayment
		fmt.Println("Payment", payment.TelegramPaymentChargeID, "for", payment.InvoicePayload)
		return laser_tele.SendMessage(chatID, "Thank you for the payment! Here is your coffee ☕")
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
