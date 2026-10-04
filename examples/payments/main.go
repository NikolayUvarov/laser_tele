// Shop bot selling for Telegram Stars: /buy sends an invoice, /refund returns the last payment.
// Payments in Stars (currency "XTR") need no payment provider.
//
// Run: TG_API_KEY=<token> go run ./examples/payments
package main

import (
	"fmt"
	"sync"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/v2/laser_tele_api"
)

// the last payment of every user: user ID -> TelegramPaymentChargeID
var payments = map[int]string{}
var paymentsMutex sync.Mutex

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: time.Second})

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		var err error
		switch update.Type() {
		case "message":
			err = onMessage(update.UpdateMessage)
		case "pre_checkout_query":
			// the last check before the payment, it must be answered within 10 seconds:
			// "" confirms the payment, a text cancels it and is shown to the user
			query := update.PreCheckoutQuery
			if query.InvoicePayload == "coffee" {
				err = laser_tele.AnswerPreCheckoutQuery(query.ID, "")
			} else {
				err = laser_tele.AnswerPreCheckoutQuery(query.ID, "Sorry, this product is sold out")
			}
		}
		if err != nil {
			fmt.Println("Error:", err)
		}
	})
}

func onMessage(message laser_tele.Message) error {
	chatID := message.Chat.ID

	// the payment is done, deliver the goods
	if payment := message.SuccessfulPayment; payment.Currency != "" {
		paymentsMutex.Lock()
		payments[message.From.ID] = payment.TelegramPaymentChargeID
		paymentsMutex.Unlock()
		return laser_tele.SendMessage(chatID, fmt.Sprintf("Thank you! You paid %d ⭐. Here is your coffee ☕", payment.TotalAmount))
	}

	switch message.Text {
	case "/buy":
		_, err := laser_tele.SendInvoice(chatID, laser_tele.InvoiceConfig{
			Title:       "Coffee",
			Description: "A big cup of hot coffee",
			Payload:     "coffee", // returned in PreCheckoutQuery and SuccessfulPayment
			Currency:    "XTR",
			Prices:      []laser_tele.LabeledPrice{{Label: "Coffee", Amount: 1}},
			PhotoURL:    "https://telegram.org/img/t_logo.png",
		})
		return err
	case "/link":
		// the link can be sent anywhere, e.g. to a website
		link, err := laser_tele.CreateInvoiceLink(laser_tele.InvoiceConfig{
			Title: "Coffee", Description: "A big cup of hot coffee", Payload: "coffee",
			Currency: "XTR", Prices: []laser_tele.LabeledPrice{{Label: "Coffee", Amount: 1}},
		})
		if err != nil {
			return err
		}
		return laser_tele.SendMessage(chatID, "Pay here: "+link)
	case "/refund":
		paymentsMutex.Lock()
		chargeID, ok := payments[message.From.ID]
		delete(payments, message.From.ID)
		paymentsMutex.Unlock()
		if !ok {
			return laser_tele.SendMessage(chatID, "You have no payments to refund")
		}
		if err := laser_tele.RefundStarPayment(message.From.ID, chargeID); err != nil {
			return err
		}
		return laser_tele.SendMessage(chatID, "Your Stars are returned")
	}
	return nil
}
