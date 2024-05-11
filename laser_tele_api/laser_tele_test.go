package laser_tele

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

// Mock the http.Get function
func mockHTTPGet(url string) (*http.Response, error) {
	responseData := `{
        "ok": true,
        "result": [
            {
                "update_id": 794872550,
                "message": {
                    "message_id": 295,
                    "from": {
                        "id": 137511897,
                        "is_bot": false,
                        "first_name": "Nikolos",
                        "username": "nikolosu",
                        "language_code": "ru"
                    },
                    "chat": {
                        "id": 137511897,
                        "first_name": "Nikolos",
                        "username": "nikolosu",
                        "type": "private"
                    },
                    "date": 1692347648,
                    "text": "ddd"
                }
            },
            {
                "update_id": 794872551,
                "message": {
                    "message_id": 296,
                    "from": {
                        "id": 137511897,
                        "is_bot": false,
                        "first_name": "Nikolos",
                        "username": "nikolosu",
                        "language_code": "ru"
                    },
                    "chat": {
                        "id": 137511897,
                        "first_name": "Nikolos",
                        "username": "nikolosu",
                        "type": "private"
                    },
                    "date": 1692347652,
                    "text": "asd"
                }
            },
            {
                "update_id": 794872552,
                "message": {
                    "message_id": 297,
                    "from": {
                        "id": 137511897,
                        "is_bot": false,
                        "first_name": "Nikolos",
                        "username": "nikolosu",
                        "language_code": "ru"
                    },
                    "chat": {
                        "id": 137511897,
                        "first_name": "Nikolos",
                        "username": "nikolosu",
                        "type": "private"
                    },
                    "date": 1692347691,
                    "text": "asdfa"
                }
            }
        ]
    }`

	r := ioutil.NopCloser(bytes.NewReader([]byte(responseData)))
	return &http.Response{
		StatusCode: 200,
		Body:       r,
	}, nil
}

func TestMakeRequest(t *testing.T) {
	// Backup the original http.Get function and replace it with the mock function
	originalGet := http.Get
	httpGet = mockHTTPGet
	defer func() { httpGet = originalGet }() // Restore the original function after the test

	// Call the makeRequest function
	UpdateRequest()

	// Assertions
	// For simplicity, we'll just check if the newUpdate.UpdateID is set correctly
	// based on the provided data. You can add more assertions as needed.
	if newUpdate.UpdateID != 794872552 {
		t.Errorf("Expected newUpdate.UpdateID to be 794872552, but got %d", newUpdate.UpdateID)
	}
}
