// Group bot: approves join requests, greets new members, reacts to messages, reports reactions and boosts.
//
// Add the bot to a group as an administrator with the rights to invite users:
// Telegram sends chat_member, reactions and boosts only to administrators.
//
// Run: TG_API_KEY=<token> go run ./examples/groups
package main

import (
	"fmt"
	"strings"
	"time"

	laser_tele "github.com/NikolayUvarov/laser_tele/v2/laser_tele_api"
)

func main() {
	laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{Timeout: 2 * time.Second})

	laser_tele.LaserTeleRun(func(update laser_tele.Update) {
		if err := onUpdate(update); err != nil {
			fmt.Println("Error:", err)
		}
	})
}

func onUpdate(update laser_tele.Update) error {
	switch update.Type() {
	case "chat_join_request":
		// approve everybody except users without a username
		request := update.ChatJoinRequest
		if request.From.Username == "" {
			return laser_tele.DeclineChatJoinRequest(request.Chat.ID, request.From.ID)
		}
		return laser_tele.ApproveChatJoinRequest(request.Chat.ID, request.From.ID)

	case "chat_member":
		member := update.ChatMember
		if member.OldChatMember.Status == "left" && member.NewChatMember.Status == "member" {
			return laser_tele.SendMessage(member.Chat.ID, "Welcome, "+member.NewChatMember.User.FirstName+"!")
		}

	case "my_chat_member":
		// the bot was added to or removed from a chat, or blocked by a user in a private chat
		member := update.MyChatMember
		fmt.Printf("The bot is %s in %s %q\n", member.NewChatMember.Status, member.Chat.Type, member.Chat.Title)

	case "message":
		message := update.UpdateMessage
		if message.Chat.Type != "private" && strings.Contains(strings.ToLower(message.Text), "thank") {
			return laser_tele.SetMessageReaction(message.Chat.ID, message.MessageID, "❤")
		}
		if message.MigrateToChatID != 0 {
			fmt.Println("The group", message.Chat.ID, "is now the supergroup", message.MigrateToChatID)
		}

	case "message_reaction":
		reaction := update.MessageReaction
		for _, r := range reaction.NewReaction {
			fmt.Println(reaction.User.FirstName, "reacted with", r.Emoji, "to message", reaction.MessageID)
		}

	case "message_reaction_count":
		// reactions of anonymous administrators and in channels come only as counts
		for _, count := range update.MessageReactionCount.Reactions {
			fmt.Println(count.Type.Emoji, count.TotalCount)
		}

	case "chat_boost":
		boost := update.ChatBoost
		return laser_tele.SendMessage(boost.Chat.ID, "Thank you for the boost, "+boost.Boost.Source.User.FirstName+"!")
	}
	return nil
}
