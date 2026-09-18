package bot

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"progress-bot/internal/chart"
	"progress-bot/internal/storage"
)

type textView struct {
	text     string
	keyboard tgbotapi.InlineKeyboardMarkup
}

func buildRootView(items []storage.Item) textView {
	text := "📋 Glavni meni"
	if len(items) == 0 {
		text += "\n\nPrazno je. Dodaj svoj prvi folder ili box."
	}

	const perRow = 3

	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for _, it := range items {
		row = append(row, itemButton(it))
		if len(row) == perRow {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
			row = nil
		}
	}

	if len(row) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ Dodaj novo", "add:0"),
	))

	return textView{text: text, keyboard: tgbotapi.NewInlineKeyboardMarkup(rows...)}
}

func buildFolderView(folder storage.Item, items []storage.Item) textView {
	text := "📁 " + folder.Name
	if len(items) == 0 {
		text += "\n\nPrazno je. Dodaj prvi folder ili box."
	}

	const perRow = 3

	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton

	for _, it := range items {
		row = append(row, itemButton(it))
		if len(row) == perRow {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
			row = nil
		}
	}

	if len(row) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
	}

	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("➕ Dodaj novo", fmt.Sprintf("add:%d", folder.ID))),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Obriši folder", fmt.Sprintf("delete_confirm:%d", folder.ID)),
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Nazad", "back:"+encodeParent(folder.ParentID)),
		),
	)

	return textView{text: text, keyboard: tgbotapi.NewInlineKeyboardMarkup(rows...)}
}

func itemButton(it storage.Item) tgbotapi.InlineKeyboardButton {
	if it.Type == storage.ItemFolder {
		return tgbotapi.NewInlineKeyboardButtonData("📁 "+it.Name, fmt.Sprintf("open_folder:%d", it.ID))
	}

	icon := "📊"
	if it.BoxType != nil && *it.BoxType == storage.BoxCheck {
		icon = "✅"
	}
	return tgbotapi.NewInlineKeyboardButtonData(icon+" "+it.Name, fmt.Sprintf("open_box:%d", it.ID))
}

func measureKeyboard(box storage.Item) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("➕ Unesi vrednost", fmt.Sprintf("add_entry:%d", box.ID))),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Obriši box", fmt.Sprintf("delete_confirm:%d", box.ID)),
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Nazad", "back:"+encodeParent(box.ParentID)),
		),
	)
}

func measureSummaryKeyboard(box storage.Item, hasEntries bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	if hasEntries {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Prikaži grafikon", fmt.Sprintf("show_chart:%d", box.ID)),
		))
	}
	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("➕ Unesi vrednost", fmt.Sprintf("add_entry:%d", box.ID))),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Obriši box", fmt.Sprintf("delete_confirm:%d", box.ID)),
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Nazad", "back:"+encodeParent(box.ParentID)),
		),
	)
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func buildMeasureSummaryView(box storage.Item, entries []storage.Entry) textView {
	text := "📊 " + box.Name
	if len(entries) == 0 {
		text += "\n\nNema još nijednog zapisa."
	} else {
		last := entries[len(entries)-1]
		text += fmt.Sprintf("\n\nPoslednji unos: %.1f (%s)\nBroj zapisa: %d", *last.Value, formatDisplayDate(last.EntryDate), len(entries))
	}
	return textView{text: text, keyboard: measureSummaryKeyboard(box, len(entries) > 0)}
}

func checkKeyboard(box storage.Item) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✅ Označi odrađeno danas", fmt.Sprintf("add_entry:%d", box.ID))),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Obriši box", fmt.Sprintf("delete_confirm:%d", box.ID)),
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Nazad", "back:"+encodeParent(box.ParentID)),
		),
	)
}

func pointsFromEntries(entries []storage.Entry) []chart.Point {
	points := make([]chart.Point, 0, len(entries))
	for _, e := range entries {
		if e.Value == nil {
			continue
		}
		date, err := time.Parse(storage.DateFormat, e.EntryDate)
		if err != nil {
			continue
		}
		points = append(points, chart.Point{Date: date, Value: *e.Value})
	}
	return points
}

func formatDisplayDate(dateStr string) string {
	d, err := time.Parse(storage.DateFormat, dateStr)
	if err != nil {
		return dateStr
	}
	return d.Format("02.01.2006")
}

// --- slanje/editovanje poruka ---

// withKeyboard garantuje da InlineKeyboardMarkup ima ne-nil InlineKeyboard
// polje.
//
// tgbotapi.InlineKeyboardMarkup{InlineKeyboard [][]InlineKeyboardButton}
// nema `omitempty` tag na tom polju, a Go-ova nula-vrednost textView{} (npr.
// kad kreiramo textView bez eksplicitnog keyboard polja, za obične
// tekstualne promptove ili poruke o grešci) ima InlineKeyboard == nil. Go-ov
// encoding/json marshaluje nil slice kao "null", ne kao "[]". Telegram API
// je strog po tom pitanju: očekuje niz (makar prazan) za "inline_keyboard"
// i odbija ceo zahtev greškom "field \"inline_keyboard\" must be of type
// Array" ako dobije null - i to tiho, kao grešku u logu, bez ikakve promene
// na strani korisnika (poruka jednostavno ostane neizmenjena). Zato ovo
// normalizujemo na jednom mestu, umesto da se oslanjamo da svaki pozivalac
// eksplicitno postavi bar prazan slice.
func withKeyboard(kb tgbotapi.InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	if kb.InlineKeyboard == nil {
		kb.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{}
	}
	return kb
}

func (b *Bot) sendText(chatID int64, v textView) {
	msg := tgbotapi.NewMessage(chatID, v.text)
	msg.ReplyMarkup = withKeyboard(v.keyboard)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("greška pri slanju poruke (chat_id=%d): %v", chatID, err)
	}
}

func (b *Bot) editText(chatID int64, messageID int, v textView) {
	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, v.text, withKeyboard(v.keyboard))
	if _, err := b.api.Send(edit); err != nil {
		log.Printf("greška pri editovanju poruke (chat_id=%d, message_id=%d): %v", chatID, messageID, err)
	}
}

func (b *Bot) deleteMessage(chatID int64, messageID int) {
	if _, err := b.api.Request(tgbotapi.NewDeleteMessage(chatID, messageID)); err != nil {
		log.Printf("greška pri brisanju poruke (chat_id=%d, message_id=%d): %v", chatID, messageID, err)
	}
}

func (b *Bot) answerCallback(id, text string) {
	if _, err := b.api.Request(tgbotapi.NewCallback(id, text)); err != nil {
		log.Printf("greška pri potvrdi callback-a: %v", err)
	}
}

func (b *Bot) renderText(cb *tgbotapi.CallbackQuery, v textView) {
	chatID := cb.Message.Chat.ID
	if len(cb.Message.Photo) == 0 {
		b.editText(chatID, cb.Message.MessageID, v)
		return
	}
	b.deleteMessage(chatID, cb.Message.MessageID)
	b.sendText(chatID, v)
}

// deleting the previous message and sending a new one is the only way to change a photo message into a text message, because Telegram does not allow editing a photo message into a text message.
func (b *Bot) renderPhoto(cb *tgbotapi.CallbackQuery, caption string, png []byte, keyboard tgbotapi.InlineKeyboardMarkup) {
	chatID := cb.Message.Chat.ID

	if len(cb.Message.Photo) > 0 {
		media := tgbotapi.NewInputMediaPhoto(tgbotapi.FileBytes{Name: "chart.png", Bytes: png})
		media.Caption = caption
		edit := tgbotapi.EditMessageMediaConfig{
			BaseEdit: tgbotapi.BaseEdit{
				ChatID:      chatID,
				MessageID:   cb.Message.MessageID,
				ReplyMarkup: &keyboard,
			},
			Media: media,
		}
		if _, err := b.api.Send(edit); err != nil {
			log.Printf("greška pri editovanju grafikona (chat_id=%d): %v", chatID, err)
		}
		return
	}

	b.deleteMessage(chatID, cb.Message.MessageID)

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: "chart.png", Bytes: png})
	photo.Caption = caption
	photo.ReplyMarkup = keyboard
	if _, err := b.api.Send(photo); err != nil {
		log.Printf("greška pri slanju grafikona (chat_id=%d): %v", chatID, err)
	}
}
