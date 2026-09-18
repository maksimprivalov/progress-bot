package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const helpText = `Progress tracking bot.

Your data is organized as a tree:
📁 Folder - groups folders and boxes (e.g. "Health" -> "Workout" -> ...). Folders can be nested arbitrarily deep.
📦 Box - a leaf of the tree, stores entries. When creating one, you pick one of two subtypes:
   📊 Measure - a numeric value per day (weight, seconds, reps...) - shown as a chart
   ✅ Check - a habit you just mark as done for the day - shown as a streak and a recent-days overview

Commands:
/menu - open the main menu
/help - show this message`

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	if msg.IsCommand() {
		b.handleCommand(msg)
		return
	}
	b.handleTextInput(msg)
}

func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start", "help":
		b.reply(msg.Chat.ID, helpText)
	case "menu":
		b.sendRootMenu(msg.Chat.ID, msg.From.ID)
	default:
		b.reply(msg.Chat.ID, "Unknown command. Type /help for the list of commands.")
	}
}

func (b *Bot) sendRootMenu(chatID, userID int64) {
	b.state.clear(userID)
	items, err := b.storage.GetChildren(userID, nil)
	if err != nil {
		log.Printf("greška pri čitanju root menija (user_id=%d): %v", userID, err)
		b.reply(chatID, "Something went wrong while reading your data.")
		return
	}
	b.sendText(chatID, buildRootView(items))
}

func (b *Bot) reply(chatID int64, text string) {
	if _, err := b.api.Send(tgbotapi.NewMessage(chatID, text)); err != nil {
		log.Printf("greška pri slanju poruke (chat_id=%d): %v", chatID, err)
	}
}
