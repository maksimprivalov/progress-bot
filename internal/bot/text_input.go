package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"progress-bot/internal/chart"
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

	if _, err := b.storage.CreateFolder(userID, action.parentID, name); err != nil {
		log.Printf("greška pri kreiranju foldera (user_id=%d): %v", userID, err)
		b.reply(msg.Chat.ID, "Došlo je do greške pri kreiranju foldera.")
		return
	}

	b.sendItemListView(msg.Chat.ID, userID, action.parentID)
}

func (b *Bot) finishAddBox(msg *tgbotapi.Message, userID int64, action pendingAction, name string) {
	b.state.clear(userID)

	if _, err := b.storage.CreateBox(userID, action.parentID, action.boxType, name); err != nil {
		log.Printf("greška pri kreiranju box-a (user_id=%d): %v", userID, err)
		b.reply(msg.Chat.ID, "Došlo je do greške pri kreiranju box-a.")
		return
	}

	b.sendItemListView(msg.Chat.ID, userID, action.parentID)
}

func (b *Bot) sendItemListView(chatID, userID int64, parentID *int64) {
	items, err := b.storage.GetChildren(userID, parentID)
	if err != nil {
		log.Printf("greška pri čitanju sadržaja (user_id=%d): %v", userID, err)
		return
	}

	if parentID == nil {
		b.sendText(chatID, buildRootView(items))
		return
	}

	folder, err := b.storage.GetItem(*parentID)
	if err != nil {
		log.Printf("greška pri čitanju foldera %d: %v", *parentID, err)
		return
	}
	b.sendText(chatID, buildFolderView(folder, items))
}

func (b *Bot) finishAddEntry(msg *tgbotapi.Message, userID int64, action pendingAction, text string) {
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		b.reply(msg.Chat.ID, fmt.Sprintf("'%s' nije validan broj. Pošalji broj, npr. 82.5", text))
		return
	}
	if value < 0 || value > 1_000_000 {
		b.reply(msg.Chat.ID, "Vrednost mora biti realan broj između 0 i 1 000 000.")
		return
	}

	b.state.clear(userID)

	today := time.Now().UTC().Format(storage.DateFormat)
	if err := b.storage.UpsertMeasureEntry(action.boxID, today, value); err != nil {
		log.Printf("greška pri upisu zapisa (box_id=%d): %v", action.boxID, err)
		b.reply(msg.Chat.ID, "Došlo je do greške pri čuvanju zapisa.")
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

	keyboard := measureKeyboard(box)
	png, err := chart.GenerateProgressPNG(box.Name, pointsFromEntries(entries))
	if err != nil {
		log.Printf("greška pri generisanju grafikona (box_id=%d): %v", box.ID, err)
		b.reply(msg.Chat.ID, fmt.Sprintf("Zabeleženo: %.1f. (Grafikon trenutno nije uspeo da se generiše.)", value))
		return
	}

	caption := fmt.Sprintf("📊 %s\n\nPoslednji unos: %.1f (%s)", box.Name, value, formatDisplayDate(today))
	photo := tgbotapi.NewPhoto(msg.Chat.ID, tgbotapi.FileBytes{Name: "chart.png", Bytes: png})
	photo.Caption = caption
	photo.ReplyMarkup = keyboard
	if _, err := b.api.Send(photo); err != nil {
		log.Printf("greška pri slanju grafikona (chat_id=%d): %v", msg.Chat.ID, err)
	}
}
