# Payments

Bots sell digital goods and services for [Telegram Stars](https://core.telegram.org/bots/payments-stars)
and physical goods through [payment providers](https://core.telegram.org/bots/payments).

Telegram Stars need no setup: the currency is `XTR`, the provider token is empty, the price is in Stars.
For other currencies connect a provider with `/mybots` > Payments in @BotFather and pass its token
in `ProviderToken`; prices are in the smallest units of the currency (cents, kopecks).

## The flow

1. The bot sends an invoice: `SendInvoice`, or a link `CreateInvoiceLink`.
2. For invoices with `IsFlexible` the user enters the address, the bot gets `shipping_query`
   and answers with the ways of delivery: `AnswerShippingQuery`.
3. The user presses Pay, the bot gets `pre_checkout_query` and must confirm or cancel it
   within 10 seconds: `AnswerPreCheckoutQuery`.
4. The bot gets a message with `SuccessfulPayment` and delivers the goods.

## Selling for Stars

```go
_, err := laser_tele.SendInvoice(chatID, laser_tele.InvoiceConfig{
	Title:       "Premium for a month",
	Description: "All features of the bot for 30 days",
	Payload:     "premium-30", // your ID of the order, not shown to the user
	Currency:    "XTR",
	Prices:      []laser_tele.LabeledPrice{{Label: "Premium", Amount: 100}}, // 100 Stars, exactly one price
})
if err != nil {
	fmt.Println(err)
}
```

```go
laser_tele.LaserTeleRun(func(update laser_tele.Update) {
	switch {
	case update.Type() == "pre_checkout_query":
		query := update.PreCheckoutQuery
		if query.InvoicePayload == "premium-30" {
			laser_tele.AnswerPreCheckoutQuery(query.ID, "") // "" confirms the payment
		} else {
			laser_tele.AnswerPreCheckoutQuery(query.ID, "This offer has expired") // the text is shown to the user
		}

	case update.UpdateMessage.SuccessfulPayment.Currency != "":
		message := update.UpdateMessage
		payment := message.SuccessfulPayment
		// save payment.TelegramPaymentChargeID: it is needed for refunds
		fmt.Println(message.From.ID, "paid", payment.TotalAmount, payment.Currency, "for", payment.InvoicePayload)
		laser_tele.SendMessage(message.Chat.ID, "Premium is activated, thank you!")
	}
})
```

Deliver the goods only after `SuccessfulPayment`: a confirmed pre-checkout query doesn't mean the payment succeeded.

## Invoice options

| `InvoiceConfig` field | Meaning |
|---|---|
| `Title`, `Description` | 1-32 and 1-255 characters |
| `Payload` | your data of the order, returned in `PreCheckoutQuery` and `SuccessfulPayment` |
| `Currency`, `Prices` | `XTR` with one price for Stars; for providers several parts: goods, delivery, tax |
| `ProviderToken` | from @BotFather, empty for Stars |
| `PhotoURL`, `PhotoWidth`, `PhotoHeight`, `PhotoSize` | a picture of the goods |
| `NeedName`, `NeedPhoneNumber`, `NeedEmail`, `NeedShippingAddress` | ask the user for the data, it comes in `OrderInfo` |
| `IsFlexible` | the price depends on the delivery, the bot gets `shipping_query` |
| `MaxTipAmount`, `SuggestedTipAmounts` | tips (not for Stars) |
| `StartParameter` | for `SendInvoice`: forwarded invoices open the bot with `/start <parameter>` |
| `SubscriptionPeriod` | for `CreateInvoiceLink` with Stars: `2592000` (30 days) makes a monthly subscription |
| `ReplyMarkup` | for `SendInvoice`: a keyboard, its first button must be `{Text: "Pay", Pay: true}` |

A link to pay can be sent anywhere, for example to a website:

```go
link, err := laser_tele.CreateInvoiceLink(laser_tele.InvoiceConfig{
	Title: "Premium", Description: "Monthly subscription", Payload: "premium-sub",
	Currency: "XTR", Prices: []laser_tele.LabeledPrice{{Label: "Premium", Amount: 100}},
	SubscriptionPeriod: 2592000,
})
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println("Pay here:", link)
```

## Delivery

```go
if update.Type() == "shipping_query" {
	query := update.ShippingQuery
	if query.ShippingAddress.CountryCode != "RU" {
		laser_tele.AnswerShippingQuery(query.ID, nil, "Sorry, we deliver only in Russia")
		return
	}
	options := []laser_tele.ShippingOption{
		{ID: "post", Title: "Post", Prices: []laser_tele.LabeledPrice{{Label: "Delivery", Amount: 30000}}},
		{ID: "courier", Title: "Courier", Prices: []laser_tele.LabeledPrice{{Label: "Delivery", Amount: 50000}}},
	}
	laser_tele.AnswerShippingQuery(query.ID, options, "")
}
```

The chosen option comes in `PreCheckoutQuery.ShippingOptionID` and `SuccessfulPayment.ShippingOptionID`.

## Refunds and subscriptions

```go
// return the Stars of a payment
if err := laser_tele.RefundStarPayment(userID, payment.TelegramPaymentChargeID); err != nil {
	fmt.Println(err)
}
```

When a subscription is canceled, renewed or its payment fails, the bot gets a `subscription` update:

```go
if update.Type() == "subscription" {
	subscription := update.Subscription
	fmt.Println(subscription.User.ID, subscription.InvoicePayload, subscription.State) // "canceled", "active" or "failed"
}
```

`EditUserStarSubscription(userID, chargeID, true)` cancels a subscription of a user, `false` enables it again.
A refunded payment comes as a message with `RefundedPayment`.

See [examples/payments](../examples/payments/main.go).

Next: [Games](games.md).
