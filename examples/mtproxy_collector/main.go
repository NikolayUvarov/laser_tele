// The bot collecting MTProxy servers for the registry of the Bot API server
// https://github.com/NikolayUvarov/telegram-bot-api. It takes links from posts of channels, where it is an administrator,
// and from messages of its administrators, including posts forwarded from any channel: from texts, captions,
// hidden links and URL buttons ("Connect"). The server checks the servers and switches its bots to a working one.
//
// The server must be started with --mtproxy-admins=<identifier of this bot> (the number before ':' in the token),
// and the bot must connect to Telegram through the server once (e.g. with --mtproxy) before it can manage the registry.
//
// Commands of administrators: /proxies (the registry), /check, /use <id>, /remove <id>.
//
// Run: TG_API_KEY=<token> BOT_API_URL=http://localhost:8081 MTPROXY_ADMINS=<your user id> \
// MTPROXY_CHANNELS=<channel ids or @usernames, all channels if empty> go run ./examples/mtproxy_collector
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/v2/laser_tele_api"
)

// maxListLength keeps /proxies within the limit of a message
const maxListLength = 3500

// registryProxy is an MTProxy server from getMTProxies
type registryProxy struct {
	ID            int64    `json:"id"`
	Server        string   `json:"server"`
	Port          int      `json:"port"`
	SecretType    string   `json:"secret_type"`
	Domain        string   `json:"domain"`
	State         string   `json:"state"` // "unchecked", "working" or "failing"
	IsActive      bool     `json:"is_active"`
	Sources       []string `json:"sources"`
	LastError     string   `json:"last_error"`
	FailuresInRow int      `json:"failures_in_row"`
	Ping          float64  `json:"ping"`
}

type registry struct {
	ActiveID          int64           `json:"active_id"`
	BotCount          int             `json:"bot_count"`
	ConnectedBotCount int             `json:"connected_bot_count"`
	IsChecking        bool            `json:"is_checking"`
	Proxies           []registryProxy `json:"proxies"`
}

type addResult struct {
	Added  int     `json:"added"`
	Known  int     `json:"known"`
	IDs    []int64 `json:"ids"`
	Errors []struct {
		Link  string `json:"link"`
		Error string `json:"error"`
	} `json:"errors"`
}

type collector struct {
	bot      *laser_tele.Bot
	channels map[string]bool // identifiers and @usernames of channels, all channels if empty
	admins   map[int]bool    // users, who can add servers and use commands
}

func newCollector(bot *laser_tele.Bot, channels, admins string) (*collector, error) {
	c := &collector{bot: bot, channels: map[string]bool{}, admins: map[int]bool{}}
	for _, channel := range strings.Split(channels, ",") {
		if channel = strings.TrimSpace(channel); channel != "" {
			c.channels[strings.ToLower(channel)] = true
		}
	}
	for _, admin := range strings.Split(admins, ",") {
		if admin = strings.TrimSpace(admin); admin == "" {
			continue
		}
		id, err := strconv.Atoi(admin)
		if err != nil {
			return nil, fmt.Errorf("wrong user identifier %q in MTPROXY_ADMINS", admin)
		}
		c.admins[id] = true
	}
	return c, nil
}

func (c *collector) isWatched(chat laser_tele.Chat) bool {
	return len(c.channels) == 0 || c.channels[strconv.Itoa(chat.ID)] ||
		(chat.Username != "" && c.channels["@"+strings.ToLower(chat.Username)])
}

// proxyTexts returns the texts of the message, in which the server looks for MTProxy links:
// the text or the caption, hidden links and URL buttons
func proxyTexts(message laser_tele.Message) []string {
	var texts []string
	for _, text := range []string{message.Text, message.Caption} {
		if text != "" {
			texts = append(texts, text)
		}
	}
	for _, entities := range [][]laser_tele.MessageEntity{message.Entities, message.CaptionEntities} {
		for _, entity := range entities {
			if entity.Type == "text_link" {
				texts = append(texts, entity.URL)
			}
		}
	}
	for _, row := range message.ReplyMarkup.InlineKeyboard {
		for _, button := range row {
			if button.URL != "" {
				texts = append(texts, button.URL)
			}
		}
	}
	return texts
}

// withLinks leaves only texts with MTProxy links, so posts without them aren't sent to the server
func withLinks(texts []string) []string {
	var result []string
	for _, text := range texts {
		if strings.Contains(strings.ToLower(text), "proxy?") {
			result = append(result, text)
		}
	}
	return result
}

func (c *collector) handle(update laser_tele.Update) {
	switch update.Type() {
	case "channel_post":
		c.collectFromChannel(update.ChannelPost)
	case "edited_channel_post":
		c.collectFromChannel(update.EditedChannelPost)
	case "message":
		c.handleMessage(update.UpdateMessage)
	}
}

func (c *collector) collectFromChannel(post laser_tele.Message) {
	if !c.isWatched(post.Chat) {
		return
	}
	texts := withLinks(proxyTexts(post))
	if len(texts) == 0 {
		return
	}
	result, err := c.add(texts, fmt.Sprintf("channel:%d/%d", post.Chat.ID, post.MessageID))
	if err != nil {
		log.Println("Can't add MTProxy servers:", err)
		return
	}
	log.Printf("Post %d of %s: added %d, known %d, errors %d", post.MessageID, post.Chat.Title, result.Added,
		result.Known, len(result.Errors))
}

func (c *collector) handleMessage(message laser_tele.Message) {
	if message.Chat.Type != "private" {
		return
	}
	if !c.admins[message.From.ID] {
		c.reply(message, fmt.Sprintf("This bot collects MTProxy servers for its administrators. Your identifier is %d.",
			message.From.ID))
		return
	}

	words := strings.Fields(message.Text)
	if len(words) > 0 && strings.HasPrefix(words[0], "/") {
		command := strings.ToLower(strings.SplitN(words[0], "@", 2)[0])
		c.reply(message, c.runCommand(command, words[1:]))
		return
	}

	texts := proxyTexts(message)
	if len(texts) == 0 {
		c.reply(message, "Send or forward a post with MTProxy links")
		return
	}
	source := fmt.Sprintf("user:%d", message.From.ID)
	if origin := message.ForwardOrigin; origin.Type == "channel" {
		source = fmt.Sprintf("channel:%d/%d", origin.Chat.ID, origin.MessageID)
	}
	result, err := c.add(texts, source)
	if err != nil {
		c.reply(message, "Error: "+err.Error())
		return
	}
	c.reply(message, describeAddResult(result))
}

func (c *collector) runCommand(command string, args []string) string {
	switch command {
	case "/proxies":
		var reg registry
		if err := c.call("getMTProxies", nil, &reg); err != nil {
			return "Error: " + err.Error()
		}
		return describeRegistry(reg)
	case "/check":
		if err := c.call("checkMTProxies", nil, nil); err != nil {
			return "Error: " + err.Error()
		}
		return "Checking all servers, see /proxies in a minute"
	case "/use", "/remove":
		if len(args) != 1 {
			return "Usage: " + command + " <id from /proxies>"
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return "Usage: " + command + " <id from /proxies>"
		}
		method := map[string]string{"/use": "setMTProxy", "/remove": "removeMTProxy"}[command]
		if err := c.call(method, map[string]interface{}{"id": id}, nil); err != nil {
			return "Error: " + err.Error()
		}
		return "Done"
	default:
		return "Send or forward posts with MTProxy links, they are added to the registry of the server.\n" +
			"/proxies - the registry\n/check - check all servers now\n/use <id> - switch bots to the server\n" +
			"/remove <id> - remove the server"
	}
}

// add sends the texts with links to the registry of the server
func (c *collector) add(texts []string, source string) (addResult, error) {
	var result addResult
	err := c.call("addMTProxies", map[string]interface{}{"links": texts, "source": source}, &result)
	return result, err
}

func (c *collector) call(method string, params map[string]interface{}, result interface{}) error {
	if params == nil {
		params = map[string]interface{}{}
	}
	raw, err := c.bot.Call(method, params)
	if err != nil || result == nil {
		return err
	}
	return json.Unmarshal(raw, result)
}

func (c *collector) reply(message laser_tele.Message, text string) {
	if err := c.bot.SendMessage(message.Chat.ID, text); err != nil {
		log.Println("Can't send message:", err)
	}
}

func describeAddResult(result addResult) string {
	text := fmt.Sprintf("Added %d, already known %d", result.Added, result.Known)
	for _, e := range result.Errors {
		text += fmt.Sprintf("\n%s: %s", e.Link, e.Error)
	}
	return text
}

func describeRegistry(reg registry) string {
	if len(reg.Proxies) == 0 {
		return "The registry is empty, send or forward posts with MTProxy links"
	}
	// the active server first, then working ones by ping
	rank := map[string]int{"working": 0, "unchecked": 1, "failing": 2}
	proxies := append([]registryProxy{}, reg.Proxies...)
	sort.SliceStable(proxies, func(i, j int) bool {
		if proxies[i].IsActive != proxies[j].IsActive {
			return proxies[i].IsActive
		}
		if rank[proxies[i].State] != rank[proxies[j].State] {
			return rank[proxies[i].State] < rank[proxies[j].State]
		}
		return proxies[i].Ping < proxies[j].Ping
	})

	text := fmt.Sprintf("Bots connected: %d of %d. Servers: %d", reg.ConnectedBotCount, reg.BotCount, len(proxies))
	if reg.IsChecking {
		text += ", checking now"
	}
	for i, proxy := range proxies {
		line := fmt.Sprintf("\n%s #%d %s:%d %s", map[string]string{"working": "✅", "unchecked": "❔", "failing": "❌"}[proxy.State],
			proxy.ID, proxy.Server, proxy.Port, proxy.SecretType)
		if proxy.Domain != "" {
			line += " " + proxy.Domain
		}
		if proxy.State == "working" {
			line += fmt.Sprintf(", %.0f ms", proxy.Ping*1000)
		}
		if proxy.State == "failing" && proxy.LastError != "" {
			line += ", " + proxy.LastError
		}
		if proxy.IsActive {
			line += " ← active"
		}
		if len(text)+len(line) > maxListLength {
			text += fmt.Sprintf("\n... and %d more", len(proxies)-i)
			break
		}
		text += line
	}
	return text
}

func main() {
	apiURL := os.Getenv("BOT_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8081"
	}
	bot, err := laser_tele.NewBot(laser_tele.LaserTeleConfigT{
		APIURL:         apiURL,
		Timeout:        2 * time.Second,
		AllowedUpdates: []string{"message", "channel_post", "edited_channel_post"},
	})
	if err != nil {
		log.Fatal(err)
	}
	c, err := newCollector(bot, os.Getenv("MTPROXY_CHANNELS"), os.Getenv("MTPROXY_ADMINS"))
	if err != nil {
		log.Fatal(err)
	}
	if len(c.admins) == 0 {
		log.Println("MTPROXY_ADMINS is empty: only posts of channels are collected")
	}
	bot.Run(c.handle)
}
