The development of this library was initiated for internal company projects, but it can be useful for anyone using Telegram to send messages. The library was designed to be user-friendly and accessible.

Code Artisan: [https://github.com/LaserPes]

Visionary Director: [https://github.com/NikolayUvarov]


## Usage

```
go get github.com/NikolayUvarov/laser_tele/laser_tele_api
```

```go
import laser_tele "github.com/NikolayUvarov/laser_tele/laser_tele_api"

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 5 * time.Second})
	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		laser_tele.SendMessage(update.UpdateMessage.Chat.ID, "Got: "+update.UpdateMessage.Text)
	})
}
```

The bot token is taken from `LaserTeleConfigT.APIKEY`, the `TG_API_KEY` environment variable or the `.APIKEY` file (in this order).
The polling interval is taken from `LaserTeleConfigT.Timeout` or the `TIMEOUT` environment variable (seconds, default 10).

A full example is in [laser_tele_example/mybot.go](laser_tele_example/mybot.go).
