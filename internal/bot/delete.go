package bot

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"progress-bot/internal/storage"
)

func (b *Bot) showDeleteConfirm(cb *tgbotapi.CallbackQuery, userID, itemID int64) {
	item, err := b.storage.GetItem(itemID)
	if err != nil || item.UserID != userID {
		log.Printf("greška pri pripremi brisanja stavke %d (user_id=%d): %v", itemID, userID, err)
		return
	}

	childItems, entryCount, err := b.storage.CountDeletionImpact(itemID)
	if err != nil {
		log.Printf("greška pri računanju uticaja brisanja stavke %d: %v", itemID, err)
		return
	}

	cancelTarget := fmt.Sprintf("open_box:%d", itemID)
	if item.Type == storage.ItemFolder {
		cancelTarget = fmt.Sprintf("open_folder:%d", itemID)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Da, obriši", fmt.Sprintf("delete_do:%d", itemID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Otkaži", cancelTarget),
		),
	)
	b.renderText(cb, textView{text: buildDeleteConfirmText(item, childItems, entryCount), keyboard: keyboard})
}

func buildDeleteConfirmText(item storage.Item, childItems, entryCount int) string {
	icon := "📁"
	if item.Type == storage.ItemBox {
		icon = "📊"
		if item.BoxType != nil && *item.BoxType == storage.BoxCheck {
			icon = "✅"
		}
	}

	text := fmt.Sprintf("⚠️ Obrisati %s %s?", icon, item.Name)
	switch {
	case childItems > 0:
		text += fmt.Sprintf("\n\nOvo će obrisati i %d stavki i %d zapisa unutra.", childItems, entryCount)
	case entryCount > 0:
		text += fmt.Sprintf("\n\nOvo će obrisati i %d zapisa.", entryCount)
	}
	text += "\n\nOva akcija je nepovratna."
	return text
}

func (b *Bot) performDelete(cb *tgbotapi.CallbackQuery, userID int64, parts []string) string {
	itemID, ok := parseID(parts, 1)
	if !ok {
		return ""
	}

	item, err := b.storage.GetItem(itemID)
	if err != nil || item.UserID != userID {
		log.Printf("greška pri brisanju stavke %d (user_id=%d): %v", itemID, userID, err)
		return "Stavka nije pronađena."
	}

	if err := b.storage.DeleteItem(itemID); err != nil {
		log.Printf("greška pri brisanju stavke %d: %v", itemID, err)
		return "Došlo je do greške pri brisanju."
	}

	b.state.clear(userID)

	if item.ParentID == nil {
		b.showRoot(cb, userID)
	} else {
		b.showFolder(cb, userID, *item.ParentID)
	}

	return fmt.Sprintf("Obrisano: %s", item.Name)
}
