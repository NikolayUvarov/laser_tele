package laser_tele

import (
	"time"
)

type UpdateJSON struct {
	Ok     bool `json:"ok"`
	Result []struct {
		UpdateID int `json:"update_id"`
		Message  struct {
			MessageID int `json:"message_id"`
			From      struct {
				ID           int    `json:"id"`
				IsBot        bool   `json:"is_bot"`
				FirstName    string `json:"first_name"`
				LastName     string `json:"last_name"`
				Username     string `json:"username"`
				LanguageCode string `json:"language_code"`
			} `json:"from"`
			Chat struct {
				ID        int    `json:"id"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
				Username  string `json:"username"`
				Type      string `json:"type"`
			} `json:"chat"`
			Date     int    `json:"date"`
			Text     string `json:"text"`
			Entities []struct {
				Offset int    `json:"offset"`
				Length int    `json:"length"`
				Type   string `json:"type"`
			} `json:"entities"`
			Photo []struct {
				FileID string `json:"file_id"`
			} `json:"photo"`
			Video struct {
				FileID   string `json:"file_id"`
				FileName string `json:"file_name"`
			} `json:"video"`
			Document struct {
				FileID   string `json:"file_id"`
				FileName string `json:"file_name"`
			} `json:"document"`
		} `json:"message,omitempty"`
		CallbackQuery struct {
			ID   string `json:"id"`
			From struct {
				ID           int    `json:"id"`
				IsBot        bool   `json:"is_bot"`
				FirstName    string `json:"first_name"`
				Username     string `json:"username"`
				LanguageCode string `json:"language_code"`
			} `json:"from"`
			Message struct {
				MessageID int `json:"message_id"`
				From      struct {
					ID        int    `json:"id"`
					IsBot     bool   `json:"is_bot"`
					FirstName string `json:"first_name"`
					Username  string `json:"username"`
				} `json:"from"`
				Chat struct {
					ID        int    `json:"id"`
					FirstName string `json:"first_name"`
					Username  string `json:"username"`
					Type      string `json:"type"`
				} `json:"chat"`
				Date        int    `json:"date"`
				Text        string `json:"text"`
				ReplyMarkup struct {
					InlineKeyboard [][]struct {
						Text         string `json:"text"`
						CallbackData string `json:"callback_data"`
					} `json:"inline_keyboard"`
				} `json:"reply_markup"`
			} `json:"message"`
			ChatInstance string `json:"chat_instance"`
			Data         string `json:"data"`
		} `json:"callback_query"`
	} `json:"result"`
}

type File struct {
	Ok     bool `json:"ok"`
	Result struct {
		FileID   string `json:"file_id"`
		FilePath string `json:"file_path"`
	} `json:"result"`
}
type Button struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}
type Row []Button
type InlineKeyboard struct {
	Keyboard []Row `json:"inline_keyboard"`
}

// func init() {
// 	APIKEY = getEnv("TG_API_KEY", "")
// 	var timeoutInt, _ = strconv.Atoi(getEnv("TIMEOUT", "10"))
// 	timeout = time.Duration(timeoutInt) * time.Second
// }

type UpdateMessageT struct {
	MessageID int
	From      struct {
		ID           int
		IsBot        bool
		FirstName    string
		LastName     string
		Username     string
		LanguageCode string
	}
	Chat struct {
		ID        int
		FirstName string
		LastName  string
		Username  string
		Type      string
	}
	Date     int
	Text     string
	Entities []struct {
		Offset int
		Length int
		Type   string
	}
	Photo []struct {
		FileID string
	}
	Video struct {
		FileID   string
		FileName string
	}
	Document struct {
		FileID   string
		FileName string
	}
}
type UpdateCallBackQueryT struct {
	ID   string
	From struct {
		ID           int
		IsBot        bool
		FirstName    string
		Username     string
		LanguageCode string
	}
	Message struct {
		MessageID int
		From      struct {
			ID        int
			IsBot     bool
			FirstName string
			Username  string
		}
		Chat struct {
			ID        int
			FirstName string
			Username  string
			Type      string
		}
		Date        int
		Text        string
		ReplyMarkup struct {
			InlineKeyboard [][]struct {
				Text         string
				CallbackData string
			}
		}
	}
	ChatInstance string
	Data         string
}

type Update struct {
	UpdateID      int
	UpdateMessage UpdateMessageT
	CallbackQuery UpdateCallBackQueryT
}

type LaserTeleConfigT struct {
	APIKEY           string
	Timeout          time.Duration
	CallbackOnUpdate interface{}
}

func AddButton(text, callback string) Button {
	var newButton Button
	newButton.Text = text
	newButton.CallbackData = callback

	return newButton

}
