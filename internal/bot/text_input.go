package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"progress-bot/internal/storage"
)

func (b *Bot) handleTextInput(msg *tgbotapi.Message) {
	userID := msg.From.ID
	action, ok := b.state.get(userID)
	if !ok {
		return
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}

	switch action.kind {
	case pendingFolderName:
		b.finishAddFolder(msg, userID, action, text)
	case pendingBoxName:
		b.finishAddBox(msg, userID, action, text)
	case pendingEntryValue:
		b.finishAddEntry(msg, userID, action, text)
	}
}

func (b *Bot) finishAddFolder(msg *tgbotapi.Message, userID int64, action pendingAction, name string) {
	b.state.clear(userID)
	b.deleteMessage(msg.Chat.ID, msg.MessageID)

	if _, err := b.storage.CreateFolder(userID, action.parentID, name); err != nil {
		log.Printf("greška pri kreiranju foldera (user_id=%d): %v", userID, err)
		b.editText(msg.Chat.ID, action.promptMessageID, textView{text: "Something went wrong while creating the folder."})
		return
	}

	b.replaceWithItemListView(msg.Chat.ID, action.promptMessageID, userID, action.parentID)
}

func (b *Bot) finishAddBox(msg *tgbotapi.Message, userID int64, action pendingAction, name string) {
	b.state.clear(userID)
	b.deleteMessage(msg.Chat.ID, msg.MessageID)

	if _, err := b.storage.CreateBox(userID, action.parentID, action.boxType, name); err != nil {
		log.Printf("greška pri kreiranju box-a (user_id=%d): %v", userID, err)
		b.editText(msg.Chat.ID, action.promptMessageID, textView{text: "Something went wrong while creating the box."})
		return
	}

	b.replaceWithItemListView(msg.Chat.ID, action.promptMessageID, userID, action.parentID)
}

func (b *Bot) replaceWithItemListView(chatID int64, messageID int, userID int64, parentID *int64) {
	items, err := b.storage.GetChildren(userID, parentID)
	if err != nil {
		log.Printf("greška pri čitanju sadržaja (user_id=%d): %v", userID, err)
		return
	}

	if parentID == nil {
		b.editText(chatID, messageID, buildRootView(items))
		return
	}

	folder, err := b.storage.GetItem(*parentID)
	if err != nil {
		log.Printf("greška pri čitanju foldera %d: %v", *parentID, err)
		return
	}
	b.editText(chatID, messageID, buildFolderView(folder, items))
}

func (b *Bot) finishAddEntry(msg *tgbotapi.Message, userID int64, action pendingAction, text string) {
	parts := strings.Fields(text)

	if len(parts) == 0 || len(parts) > 2 {
		b.reply(msg.Chat.ID, "Invalid format. Send a number or a number with a date, e.g. 82.5 or 82.5 19/09/2026")
		return
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		b.reply(msg.Chat.ID, fmt.Sprintf("'%s' is not a valid number. Send a number, e.g. 82.5", parts[0]))
		return
	}

	if value < 0 || value > 1_000_000 {
		b.reply(msg.Chat.ID, "The value must be a real number between 0 and 1,000,000.")
		return
	}

	date := time.Now().UTC().Format(storage.DateFormat)

	if len(parts) == 2 {
		parsedDate, err := time.Parse("02/01/2006", parts[1])
		if err != nil {
			b.reply(msg.Chat.ID, "Invalid date. Use the format dd/mm/yyyy, e.g. 19/09/2026")
			return
		}

		date = parsedDate.Format(storage.DateFormat)
	}

	b.state.clear(userID)
	b.deleteMessage(msg.Chat.ID, msg.MessageID)

	if err := b.storage.UpsertMeasureEntry(action.boxID, date, value); err != nil {
		log.Printf("greška pri upisu zapisa (box_id=%d): %v", action.boxID, err)
		b.editText(msg.Chat.ID, action.promptMessageID, textView{text: "Something went wrong while saving the entry."})
		return
	}

	box, err := b.storage.GetItem(action.boxID)
	if err != nil {
		log.Printf("greška pri čitanju box-a %d: %v", action.boxID, err)
		return
	}

	entries, err := b.storage.GetEntries(action.boxID)
	if err != nil {
		log.Printf("greška pri čitanju zapisa box-a %d: %v", action.boxID, err)
		return
	}

	b.editText(msg.Chat.ID, action.promptMessageID, buildMeasureSummaryView(box, entries))
}
