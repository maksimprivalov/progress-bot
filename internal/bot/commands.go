package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const helpText = `<b>Progress Tracking Bot</b>

Organize and track your progress in a simple tree:

📁 <b>Folder</b> - organizes Folders and Trackers. Folders can be nested freely.
📝 <b>Tracker</b> - tracks one thing. Choose one type:
- 📊 <b>Measure</b> - a daily number (weight, reps, time, etc.), shown as a chart.
- ✅ <b>Check</b> - mark a habit as done each day, shown as a streak and recent activity.

Example:
📁 Health → 📁 Workout → 📁 Chest → 📁 Bench Press → 📊 8-12 reps → 70
📁 Health → 📁 Morning Routine → ✅ Plan the day → ✅`

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
		b.sendRootStartMessage(msg.Chat.ID, msg.From.ID)
	case "menu":
		b.sendRootMenu(msg.Chat.ID, msg.From.ID)
	default:
		b.reply(msg.Chat.ID, "Unknown command. Type /help for the list of commands.")
	}
	b.deleteMessage(msg.Chat.ID, msg.MessageID)
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

func (b *Bot) sendRootStartMessage(chatID, userID int64) {
	b.state.clear(userID)

	b.sendText(chatID, buildStartView(helpText))
}

func (b *Bot) reply(chatID int64, text string) {
	if _, err := b.api.Send(tgbotapi.NewMessage(chatID, text)); err != nil {
		log.Printf("greška pri slanju poruke (chat_id=%d): %v", chatID, err)
	}
}
