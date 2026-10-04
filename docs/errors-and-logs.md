# Errors and logs

## Errors

Every function sending a request returns an error. There are three kinds:

- `*laser_tele.APIError`: Telegram refused the request. `ErrorCode` and `Description` are from Telegram.
- A network error: no connection, a timeout. The token is removed from its text.
- An error of the program: a file to send doesn't exist (`errors.Is(err, os.ErrNotExist)`), a file can't be saved.

```go
err := laser_tele.SendMessage(chatID, "Hello")
var apiErr *laser_tele.APIError
switch {
case err == nil:
	// sent
case errors.As(err, &apiErr) && apiErr.ErrorCode == 403:
	fmt.Println("The user blocked the bot, stop writing to", chatID)
case errors.As(err, &apiErr) && apiErr.ErrorCode == 429:
	fmt.Println("Too many requests, wait", apiErr.RetryAfter, "seconds")
case errors.As(err, &apiErr):
	fmt.Println("Telegram refused:", apiErr.Description)
default:
	fmt.Println("Network error:", err)
}
```

Frequent errors:

| Code | Description | Reason |
|---|---|---|
| 400 | `Bad Request: chat not found` | wrong chat ID, or the user never started the bot |
| 400 | `Bad Request: message is not modified` | the edited message is the same as before |
| 400 | `Bad Request: message to edit not found` | the message was deleted |
| 400 | `Bad Request: query is too old` | a callback or inline query was answered too late |
| 400 | `Bad Request: group chat was upgraded to a supergroup chat` | use `MigrateToChatID` |
| 401 | `Unauthorized` | wrong token |
| 403 | `Forbidden: bot was blocked by the user` | the user blocked the bot |
| 403 | `Forbidden: bot is not a member of the group chat` | the bot was removed from the group |
| 409 | `Conflict: terminated by other getUpdates request` | another program receives updates of the same bot |
| 429 | `Too Many Requests: retry after N` | wait `RetryAfter` seconds |

### Retrying after 429

```go
func sendWithRetry(chatID int, text string) error {
	for attempt := 0; attempt < 3; attempt++ {
		err := laser_tele.SendMessage(chatID, text)
		var apiErr *laser_tele.APIError
		if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
			time.Sleep(time.Duration(apiErr.RetryAfter) * time.Second)
			continue
		}
		return err
	}
	return errors.New("too many requests")
}
```

Errors of requesting updates are printed and the bot tries again after `Timeout`.

## Logs

Every request and response is written to a log file named after the function: `sendMessage.log`,
`updateRequest.log`, `polls.log`, `payments.log`, `call.log`...

```
2026/10/04 09:15:02 sendMessage   REQV: GET sendMessage chat_id=137511897
2026/10/04 09:15:02 sendMessage   RESP: 200 OK
2026/10/04 09:15:07 updateRequest   UPDATES: 794872551 message, 794872552 callback_query
```

| Setting | Default | Meaning |
|---|---|---|
| `LogMode` | `LogWithoutContent` | requests and responses without texts of messages: methods, chat IDs, statuses, errors |
| | `LogFull` | full requests and responses with texts, for debugging |
| | `LogOff` | no logs |
| `LogDir` | the current directory | directory of the log files, created if needed |
| `LogMaxSize` | 10 MB | when a file is larger, it is renamed to `<name>.log.1` (the previous `.log.1` is deleted) |

```go
laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{
	LogMode:    laser_tele.LogFull, // while debugging
	LogDir:     "/var/log/mybot",
	LogMaxSize: 1 << 20, // 1 MB
})
```

The token is never written to the logs in any mode. The logs of the library don't change the output
of the standard `log` package of your program.

Besides the files, the bot prints to the standard output that it started and a line per request of updates.

Next: [Reference](reference.md).
