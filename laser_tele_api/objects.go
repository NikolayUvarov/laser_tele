package laser_tele_api

import (
	"fmt"
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
	APIKEY  string        // bot token, if empty it is taken from TG_API_KEY env or .APIKEY file
	Timeout time.Duration // interval between requests of updates, if 0 it is taken from TIMEOUT env (default 10s)
	// CallbackOnUpdate (optional) is called for every new update,
	// in addition to the callback passed to LaserTeleRun
	CallbackOnUpdate Callback
}

// APIError is returned when Telegram refused the request,
// e.g. ErrorCode 403 if the bot was blocked by the user
type APIError struct {
	Method      string
	ErrorCode   int
	Description string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram %s: %d %s", e.Method, e.ErrorCode, e.Description)
}

// common part of all Bot API responses
type apiResponse struct {
	Ok          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

func AddButton(text, callback string) Button {
	var newButton Button
	newButton.Text = text
	newButton.CallbackData = callback

	return newButton

}
