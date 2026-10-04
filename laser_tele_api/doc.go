// Package laser_tele_api is a library for Telegram bots (https://core.telegram.org/bots/api).
//
// Import it as laser_tele:
//
//	import laser_tele "github.com/NikolayUvarov/laser_tele/laser_tele_api"
//
// # Quick start
//
// Create a bot with @BotFather, put its token to the TG_API_KEY environment variable
// (or to the .APIKEY file) and run:
//
//	func main() {
//		laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 2 * time.Second})
//		laser_tele.LaserTeleRun(func(update laser_tele.Update) {
//			if update.Type() == "message" {
//				err := laser_tele.SendMessage(update.UpdateMessage.Chat.ID, "You said: "+update.UpdateMessage.Text)
//				if err != nil {
//					fmt.Println("Can't send message:", err)
//				}
//			}
//		})
//	}
//
// # Default bot and Bot
//
// Package-level functions (SendMessage, LaserTeleRun...) work with the default bot,
// configured by DoLaserTeleInit. To run several bots in one program create them with NewBot:
// a Bot has the same methods (bot.SendMessage, bot.Run...).
//
// # Updates
//
// LaserTeleRun (Bot.Run) requests updates every Timeout and passes each of them to the callback,
// to CallbackOnUpdate of the config and to the channel created by MakeChan.
// Update.Type returns the kind of the update ("message", "callback_query", "inline_query",
// "poll_answer", "pre_checkout_query"...), only the field of this kind is filled
// (UpdateMessage, CallbackQuery, InlineQuery, PollAnswer, PreCheckoutQuery...).
// Update.Raw is the JSON received from Telegram.
//
// # Sending
//
// SendMessage, SendMessageWithConfig, SendKeyboard, SendPhoto, SendVideo, SendDocument, SendPoll,
// SendInvoice, SendGame and others return an error; when Telegram refused the request it is *APIError.
// Any Bot API method without its own function can be called with Call.
//
// # Logs
//
// Requests and responses are written to *.log files without texts of messages and without the token;
// see LogMode, LaserTeleConfigT.LogDir and LaserTeleConfigT.LogMaxSize.
//
// Guides with examples: https://github.com/NikolayUvarov/laser_tele/tree/main/docs,
// example bots: https://github.com/NikolayUvarov/laser_tele/tree/main/examples.
package laser_tele_api
