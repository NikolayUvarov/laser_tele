package laser_tele_api

// Payments: https://core.telegram.org/bots/payments and https://core.telegram.org/bots/payments-stars.
// The flow: SendInvoice -> (ShippingQuery -> AnswerShippingQuery, only for IsFlexible invoices) ->
// PreCheckoutQuery -> AnswerPreCheckoutQuery within 10 seconds -> message with SuccessfulPayment -> deliver the goods

type Invoice struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	StartParameter string `json:"start_parameter"`
	Currency       string `json:"currency"`
	TotalAmount    int    `json:"total_amount"` // in the smallest units of the currency (cents...)
}

type ShippingAddress struct {
	CountryCode string `json:"country_code"`
	State       string `json:"state"`
	City        string `json:"city"`
	StreetLine1 string `json:"street_line1"`
	StreetLine2 string `json:"street_line2"`
	PostCode    string `json:"post_code"`
}

type OrderInfo struct {
	Name            string          `json:"name"`
	PhoneNumber     string          `json:"phone_number"`
	Email           string          `json:"email"`
	ShippingAddress ShippingAddress `json:"shipping_address"`
}

type SuccessfulPayment struct {
	Currency                   string    `json:"currency"`
	TotalAmount                int       `json:"total_amount"`
	InvoicePayload             string    `json:"invoice_payload"`
	SubscriptionExpirationDate int       `json:"subscription_expiration_date"`
	IsRecurring                bool      `json:"is_recurring"`
	IsFirstRecurring           bool      `json:"is_first_recurring"`
	ShippingOptionID           string    `json:"shipping_option_id"`
	OrderInfo                  OrderInfo `json:"order_info"`
	TelegramPaymentChargeID    string    `json:"telegram_payment_charge_id"` // pass it to RefundStarPayment
	ProviderPaymentChargeID    string    `json:"provider_payment_charge_id"`
}

type RefundedPayment struct {
	Currency                string `json:"currency"`
	TotalAmount             int    `json:"total_amount"`
	InvoicePayload          string `json:"invoice_payload"`
	TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
	ProviderPaymentChargeID string `json:"provider_payment_charge_id"`
}

// ShippingQuery is sent for invoices with IsFlexible, answer it with AnswerShippingQuery
type ShippingQuery struct {
	ID              string          `json:"id"`
	From            User            `json:"from"`
	InvoicePayload  string          `json:"invoice_payload"`
	ShippingAddress ShippingAddress `json:"shipping_address"`
}

// PreCheckoutQuery is sent before the payment, answer it with AnswerPreCheckoutQuery within 10 seconds
type PreCheckoutQuery struct {
	ID               string    `json:"id"`
	From             User      `json:"from"`
	Currency         string    `json:"currency"`
	TotalAmount      int       `json:"total_amount"`
	InvoicePayload   string    `json:"invoice_payload"`
	ShippingOptionID string    `json:"shipping_option_id"`
	OrderInfo        OrderInfo `json:"order_info"`
}

// PaidMediaPurchased is sent when a user bought paid media with a payload sent by the bot
type PaidMediaPurchased struct {
	From             User   `json:"from"`
	PaidMediaPayload string `json:"paid_media_payload"`
}

// BotSubscriptionUpdated is sent when a payment subscription of a user is changed
type BotSubscriptionUpdated struct {
	User           User   `json:"user"`
	InvoicePayload string `json:"invoice_payload"`
	State          string `json:"state"` // "canceled", "active" or "failed"
}

type LabeledPrice struct {
	Label  string `json:"label"`
	Amount int    `json:"amount"` // in the smallest units of the currency (cents...), in Stars for "XTR"
}

type ShippingOption struct {
	ID     string         `json:"id"`
	Title  string         `json:"title"`
	Prices []LabeledPrice `json:"prices"`
}

// InvoiceConfig describes an invoice for SendInvoice and CreateInvoiceLink.
// For payments in Telegram Stars use Currency "XTR", empty ProviderToken and exactly one price
type InvoiceConfig struct {
	Title       string `json:"title"`       // 1-32 characters
	Description string `json:"description"` // 1-255 characters
	// Payload is returned in PreCheckoutQuery and SuccessfulPayment, it is not shown to the user
	Payload       string         `json:"payload"`
	ProviderToken string         `json:"provider_token,omitempty"` // from @BotFather, empty for Telegram Stars
	Currency      string         `json:"currency"`                 // "XTR" for Telegram Stars, "USD", "EUR", "RUB"...
	Prices        []LabeledPrice `json:"prices"`
	// SubscriptionPeriod (only CreateInvoiceLink, Stars): 2592000 (30 days) for a monthly subscription
	SubscriptionPeriod        int    `json:"subscription_period,omitempty"`
	MaxTipAmount              int    `json:"max_tip_amount,omitempty"`
	SuggestedTipAmounts       []int  `json:"suggested_tip_amounts,omitempty"`
	StartParameter            string `json:"start_parameter,omitempty"` // only SendInvoice
	ProviderData              string `json:"provider_data,omitempty"`
	PhotoURL                  string `json:"photo_url,omitempty"`
	PhotoSize                 int    `json:"photo_size,omitempty"`
	PhotoWidth                int    `json:"photo_width,omitempty"`
	PhotoHeight               int    `json:"photo_height,omitempty"`
	NeedName                  bool   `json:"need_name,omitempty"`
	NeedPhoneNumber           bool   `json:"need_phone_number,omitempty"`
	NeedEmail                 bool   `json:"need_email,omitempty"`
	NeedShippingAddress       bool   `json:"need_shipping_address,omitempty"`
	SendPhoneNumberToProvider bool   `json:"send_phone_number_to_provider,omitempty"`
	SendEmailToProvider       bool   `json:"send_email_to_provider,omitempty"`
	IsFlexible                bool   `json:"is_flexible,omitempty"` // the price depends on shipping, ShippingQuery is sent
	// ReplyMarkup (only SendInvoice): the first button must be a Pay button, e.g. Button{Text: "Pay", Pay: true}
	ReplyMarkup *InlineKeyboard `json:"reply_markup,omitempty"`
}

// SendInvoice sends an invoice and returns the sent message
func (b *Bot) SendInvoice(chatID int, invoice InvoiceConfig) (Message, error) {
	invoice.SubscriptionPeriod = 0
	params := struct {
		ChatID int `json:"chat_id"`
		InvoiceConfig
	}{chatID, invoice}
	var message Message
	err := b.callInto("payments", "sendInvoice", params, &message)
	return message, err
}

// CreateInvoiceLink creates a link to pay the invoice, it can be sent anywhere
func (b *Bot) CreateInvoiceLink(invoice InvoiceConfig) (string, error) {
	invoice.StartParameter = ""
	invoice.ReplyMarkup = nil
	var link string
	err := b.callInto("payments", "createInvoiceLink", invoice, &link)
	return link, err
}

// AnswerShippingQuery answers the shipping query with options of delivery.
// errorMessage (shown to the user) means that the delivery to the address is impossible
func (b *Bot) AnswerShippingQuery(shippingQueryID string, options []ShippingOption, errorMessage string) error {
	params := map[string]interface{}{"shipping_query_id": shippingQueryID, "ok": errorMessage == ""}
	if errorMessage != "" {
		params["error_message"] = errorMessage
	} else {
		params["shipping_options"] = options
	}
	return b.callInto("payments", "answerShippingQuery", params, nil)
}

// AnswerPreCheckoutQuery confirms the payment (errorMessage "") or cancels it with errorMessage shown to the user.
// It must be called within 10 seconds after the query is received
func (b *Bot) AnswerPreCheckoutQuery(preCheckoutQueryID, errorMessage string) error {
	params := map[string]interface{}{"pre_checkout_query_id": preCheckoutQueryID, "ok": errorMessage == ""}
	if errorMessage != "" {
		params["error_message"] = errorMessage
	}
	return b.callInto("payments", "answerPreCheckoutQuery", params, nil)
}

// RefundStarPayment returns Telegram Stars of the payment (SuccessfulPayment.TelegramPaymentChargeID) to the user
func (b *Bot) RefundStarPayment(userID int, telegramPaymentChargeID string) error {
	params := map[string]interface{}{"user_id": userID, "telegram_payment_charge_id": telegramPaymentChargeID}
	return b.callInto("payments", "refundStarPayment", params, nil)
}

// EditUserStarSubscription cancels (isCanceled true) or re-enables the subscription of the user paid in Telegram Stars
func (b *Bot) EditUserStarSubscription(userID int, telegramPaymentChargeID string, isCanceled bool) error {
	params := map[string]interface{}{"user_id": userID, "telegram_payment_charge_id": telegramPaymentChargeID, "is_canceled": isCanceled}
	return b.callInto("payments", "editUserStarSubscription", params, nil)
}

// SendInvoice sends an invoice and returns the sent message
func SendInvoice(chatID int, invoice InvoiceConfig) (Message, error) {
	bot, err := getDefaultBot()
	if err != nil {
		return Message{}, err
	}
	return bot.SendInvoice(chatID, invoice)
}

// CreateInvoiceLink creates a link to pay the invoice, it can be sent anywhere
func CreateInvoiceLink(invoice InvoiceConfig) (string, error) {
	bot, err := getDefaultBot()
	if err != nil {
		return "", err
	}
	return bot.CreateInvoiceLink(invoice)
}

// AnswerShippingQuery answers the shipping query with options of delivery.
// errorMessage (shown to the user) means that the delivery to the address is impossible
func AnswerShippingQuery(shippingQueryID string, options []ShippingOption, errorMessage string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.AnswerShippingQuery(shippingQueryID, options, errorMessage)
}

// AnswerPreCheckoutQuery confirms the payment (errorMessage "") or cancels it with errorMessage shown to the user.
// It must be called within 10 seconds after the query is received
func AnswerPreCheckoutQuery(preCheckoutQueryID, errorMessage string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.AnswerPreCheckoutQuery(preCheckoutQueryID, errorMessage)
}

// RefundStarPayment returns Telegram Stars of the payment (SuccessfulPayment.TelegramPaymentChargeID) to the user
func RefundStarPayment(userID int, telegramPaymentChargeID string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.RefundStarPayment(userID, telegramPaymentChargeID)
}

// EditUserStarSubscription cancels (isCanceled true) or re-enables the subscription of the user paid in Telegram Stars
func EditUserStarSubscription(userID int, telegramPaymentChargeID string, isCanceled bool) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.EditUserStarSubscription(userID, telegramPaymentChargeID, isCanceled)
}
