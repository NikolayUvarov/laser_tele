package laser_tele_api_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/v2/laser_tele_api"
)

// An echo bot. The token is taken from the TG_API_KEY environment variable or the .APIKEY file
func Example() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 2 * time.Second})
	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		if update.Type() != "message" {
			return
		}
		message := update.UpdateMessage
		if err := laser_tele.SendMessage(message.Chat.ID, "You said: "+message.Text); err != nil {
			fmt.Println("Can't send message:", err)
		}
	})
}

// Several bots in one program, each with its own token, logs and downloads
func ExampleNewBot() {
	news, err := laser_tele.NewBot(laser_tele.LaserTeleConfigT{
		APIKEY: os.Getenv("NEWS_BOT_KEY"), LogDir: "logs/news", DownloadDir: "files/news",
	})
	if err != nil {
		log.Fatal(err)
	}
	support, err := laser_tele.NewBot(laser_tele.LaserTeleConfigT{
		APIKEY: os.Getenv("SUPPORT_BOT_KEY"), LogDir: "logs/support", DownloadDir: "files/support",
	})
	if err != nil {
		log.Fatal(err)
	}

	go news.Run(func(update laser_tele.Update) {
		news.SendMessage(update.UpdateMessage.Chat.ID, "Subscribe to our news!")
	})
	support.Run(func(update laser_tele.Update) {
		support.SendMessage(update.UpdateMessage.Chat.ID, "We will answer soon")
	})
}

// Updates can be read from a channel instead of a callback
func ExampleMakeChan() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{})
	laser_tele.MakeChan()
	go laser_tele.LaserTeleRun(nil)

	for update := range laser_tele.TgChan {
		fmt.Println("Got update", update.UpdateID, "of type", update.Type())
	}
}

// Processing updates of different kinds
func ExampleUpdate_Type() {
	data := `{"update_id": 1, "poll_answer": {"poll_id": "p1", "user": {"id": 5, "first_name": "Ann"}, "option_ids": [1]}}`
	var update laser_tele.Update
	if err := json.Unmarshal([]byte(data), &update); err != nil {
		log.Fatal(err)
	}

	switch update.Type() {
	case "message":
		fmt.Println("Message:", update.UpdateMessage.Text)
	case "callback_query":
		fmt.Println("Button:", update.CallbackQuery.Data)
	case "poll_answer":
		fmt.Println(update.PollAnswer.User.FirstName, "voted for", update.PollAnswer.OptionIDs)
	default:
		fmt.Println("Skipped", update.Type())
	}
	// Output: Ann voted for [1]
}

// An inline keyboard with a button answered by the bot and a link
func ExampleSendKeyboard() {
	keyboard := laser_tele.InlineKeyboard{Keyboard: []laser_tele.Row{
		{laser_tele.AddButton("Yes", "answer_yes"), laser_tele.AddButton("No", "answer_no")},
		{{Text: "Website", URL: "https://example.com"}},
	}}
	chatID := 137511897
	if err := laser_tele.SendKeyboard(chatID, "Do you like the bot?", keyboard); err != nil {
		fmt.Println(err)
	}

	// when a button is pressed, the bot gets a callback_query update, which must be answered
	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		if update.Type() == "callback_query" {
			query := update.CallbackQuery
			laser_tele.AnswerCallbackQuery(query.ID, "Thank you!")
			laser_tele.EditMessageText(query.Message.Chat.ID, query.Message.MessageID, "You answered: "+query.Data)
		}
	})
}

// The keyboard is sent as JSON
func ExampleInlineKeyboard() {
	keyboard := laser_tele.InlineKeyboard{Keyboard: []laser_tele.Row{
		{laser_tele.AddButton("Yes", "yes"), {Text: "Site", URL: "https://example.com"}},
	}}
	data, _ := json.Marshal(keyboard)
	fmt.Println(string(data))
	// Output: {"inline_keyboard":[[{"text":"Yes","callback_data":"yes"},{"text":"Site","url":"https://example.com"}]]}
}

// A message with HTML, sent as a reply
func ExampleSendMessageWithConfig() {
	chatID, messageID := 137511897, 42
	sent, err := laser_tele.SendMessageWithConfig(chatID, "<b>Done</b>, see the <a href=\"https://example.com\">report</a>",
		laser_tele.MessageConfig{ParseMode: "HTML", ReplyToMessageID: messageID})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Sent message", sent.MessageID)
}

// Downloading a photo sent by the user
func ExampleLoadFile() {
	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		photo := update.UpdateMessage.Photo
		if len(photo) == 0 {
			return
		}
		// sizes of the photo, the last one is the largest
		path, err := laser_tele.LoadFile(update.UpdateMessage.Chat.ID, photo[len(photo)-1].FileID)
		if err != nil {
			fmt.Println("Can't download the photo:", err)
			return
		}
		fmt.Println("The photo is saved to", path)
	})
}

// A quiz and the votes of users. Votes come only for non-anonymous polls sent by the bot
func ExampleSendPoll() {
	chatID := 137511897
	_, err := laser_tele.SendPoll(chatID, "2 + 2 = ?", []string{"3", "4", "5"}, laser_tele.PollConfig{
		Quiz: true, CorrectOptionIDs: []int{1}, Explanation: "Simple math", NotAnonymous: true,
	})
	if err != nil {
		fmt.Println(err)
	}

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		if update.Type() == "poll_answer" {
			answer := update.PollAnswer
			fmt.Println(answer.User.FirstName, "chose", answer.OptionIDs)
		}
	})
}

// Inline mode: the user types "@your_bot text" in any chat. Enable it with /setinline in @BotFather
func ExampleAnswerInlineQuery() {
	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		if update.Type() != "inline_query" {
			return
		}
		query := update.InlineQuery
		upper := laser_tele.NewInlineArticle("upper", "Upper case", strings.ToUpper(query.Query))
		upper["description"] = strings.ToUpper(query.Query)
		lower := laser_tele.NewInlineArticle("lower", "Lower case", strings.ToLower(query.Query))
		err := laser_tele.AnswerInlineQuery(query.ID, []laser_tele.InlineQueryResult{upper, lower},
			laser_tele.InlineQueryConfig{CacheTime: 10})
		if err != nil {
			fmt.Println(err)
		}
	})
}

// Results of an inline query are sent as JSON
func ExampleNewInlineArticle() {
	result := laser_tele.NewInlineArticle("1", "Hello", "Hello, world!")
	result["description"] = "Sends a greeting"
	data, _ := json.Marshal(result)
	fmt.Println(string(data))
	// Output: {"description":"Sends a greeting","id":"1","input_message_content":{"message_text":"Hello, world!"},"title":"Hello","type":"article"}
}

// Selling for Telegram Stars: no payment provider is needed
func ExampleSendInvoice() {
	chatID := 137511897
	_, err := laser_tele.SendInvoice(chatID, laser_tele.InvoiceConfig{
		Title:       "Coffee",
		Description: "A big cup of coffee",
		Payload:     "order-1",
		Currency:    "XTR",
		Prices:      []laser_tele.LabeledPrice{{Label: "Coffee", Amount: 10}},
	})
	if err != nil {
		fmt.Println(err)
	}

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		switch {
		case update.Type() == "pre_checkout_query":
			// confirm the payment within 10 seconds, or cancel it with an error message
			laser_tele.AnswerPreCheckoutQuery(update.PreCheckoutQuery.ID, "")
		case update.UpdateMessage.SuccessfulPayment.Currency != "":
			payment := update.UpdateMessage.SuccessfulPayment
			fmt.Println("Paid", payment.TotalAmount, payment.Currency, "for", payment.InvoicePayload)
			laser_tele.SendMessage(update.UpdateMessage.Chat.ID, "Here is your coffee ☕")
		}
	})
}

// A game created with /newgame in @BotFather
func ExampleSendGame() {
	chatID := 137511897
	if _, err := laser_tele.SendGame(chatID, "race"); err != nil {
		fmt.Println(err)
	}

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		query := update.CallbackQuery
		if query.GameShortName == "race" {
			// the "Play" button opens the page of the game
			laser_tele.AnswerCallbackQueryWithConfig(query.ID, laser_tele.CallbackAnswerConfig{
				URL: "https://example.com/race?user=" + fmt.Sprint(query.From.ID),
			})
		}
	})
}

// Setting the score of a user after the game reported it to the server of the game
func ExampleSetGameScore() {
	userID, score := 5, 1200
	game := laser_tele.GameMessage{ChatID: 137511897, MessageID: 42}
	if err := laser_tele.SetGameScore(userID, score, game, false); err != nil {
		fmt.Println(err)
	}
	scores, err := laser_tele.GetGameHighScores(userID, game)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, row := range scores {
		fmt.Println(row.Position, row.User.FirstName, row.Score)
	}
}

// Reactions: the bot reacts to messages and sees reactions of users (only as a chat administrator)
func ExampleSetMessageReaction() {
	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		switch update.Type() {
		case "message":
			laser_tele.SetMessageReaction(update.UpdateMessage.Chat.ID, update.UpdateMessage.MessageID, "👍")
		case "message_reaction":
			reaction := update.MessageReaction
			fmt.Println(reaction.User.FirstName, "reacted with", reaction.NewReaction)
		}
	})
}

// Approving requests to join a group, the bot must be an administrator with the right to invite users
func ExampleApproveChatJoinRequest() {
	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		if update.Type() == "chat_join_request" {
			request := update.ChatJoinRequest
			if err := laser_tele.ApproveChatJoinRequest(request.Chat.ID, request.From.ID); err != nil {
				fmt.Println(err)
			}
		}
	})
}

// Calling a Bot API method, which has no own function in the library
func ExampleCall() {
	result, err := laser_tele.Call("getMe", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	var me laser_tele.User
	json.Unmarshal(result, &me)
	fmt.Println("I am", me.Username)

	// methods with parameters
	laser_tele.Call("sendDice", map[string]interface{}{"chat_id": 137511897, "emoji": "🎲"})
}

// Handling errors: Telegram refused the request or asked to wait
func ExampleAPIError() {
	// any function sending a request returns such errors, e.g. err := laser_tele.SendMessage(chatID, text);
	// for this example the error is created here
	var err error = &laser_tele.APIError{Method: "sendMessage", ErrorCode: 429, Description: "Too Many Requests: retry after 3", RetryAfter: 3}

	var apiErr *laser_tele.APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.ErrorCode == 403:
			fmt.Println("The user blocked the bot")
		case apiErr.RetryAfter > 0:
			fmt.Println("Too many requests, wait", apiErr.RetryAfter, "seconds")
		default:
			fmt.Println("Telegram refused:", apiErr.Description)
		}
	} else if err != nil {
		fmt.Println("Network error:", err)
	}
	// Output: Too many requests, wait 3 seconds
}
