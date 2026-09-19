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
	text := "📋 Main menu. Your Folders and Trackers start here."
	if len(items) == 0 {
		text += "\n\nIt's empty. Go on and add your first Folder or Tracker."
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
		tgbotapi.NewInlineKeyboardButtonData("➕ Add new", "add:0"),
	))

	return textView{text: text, keyboard: tgbotapi.NewInlineKeyboardMarkup(rows...)}
}

func buildStartView(helpText string) textView {

	var rows [][]tgbotapi.InlineKeyboardButton

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✔️ Got it!", "start"),
	))

	return textView{text: helpText, keyboard: tgbotapi.NewInlineKeyboardMarkup(rows...)}
}

func buildFolderView(folder storage.Item, items []storage.Item) textView {
	text := "📁 <b>" + folder.Name + "</b>"
	if len(items) == 0 {
		text += "\n\nIt's empty. Add your first Folder or Tracker."
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
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("➕ Add new", fmt.Sprintf("add:%d", folder.ID))),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Delete folder", fmt.Sprintf("delete_confirm:%d", folder.ID)),
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back", "back:"+encodeParent(folder.ParentID)),
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
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("➕ Enter value", fmt.Sprintf("add_entry:%d", box.ID))),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Delete box", fmt.Sprintf("delete_confirm:%d", box.ID)),
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back", "back:"+encodeParent(box.ParentID)),
		),
	)
}

func measureSummaryKeyboard(box storage.Item, hasEntries bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	if hasEntries {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Show chart", fmt.Sprintf("show_chart:%d", box.ID)),
		))
	}
	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("➕ Enter value", fmt.Sprintf("add_entry:%d", box.ID))),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Delete box", fmt.Sprintf("delete_confirm:%d", box.ID)),
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back", "back:"+encodeParent(box.ParentID)),
		),
	)
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func buildMeasureSummaryView(box storage.Item, entries []storage.Entry) textView {
	text := "📊 <b>" + box.Name + "</b>"
	if len(entries) == 0 {
		text += "\n\nNo entries yet."
	} else {
		last := entries[len(entries)-1]
		text += fmt.Sprintf("\n\nLatest entry: %.1f (%s)\nEntries: %d", *last.Value, formatDisplayDate(last.EntryDate), len(entries))
	}
	return textView{text: text, keyboard: measureSummaryKeyboard(box, len(entries) > 0)}
}

func checkKeyboard(box storage.Item) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✅ Mark done today", fmt.Sprintf("add_entry:%d", box.ID))),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Delete box", fmt.Sprintf("delete_confirm:%d", box.ID)),
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back", "back:"+encodeParent(box.ParentID)),
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

func withKeyboard(kb tgbotapi.InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	if kb.InlineKeyboard == nil {
		kb.InlineKeyboard = [][]tgbotapi.InlineKeyboardButton{}
	}
	return kb
}

// getting id of the message to edit it later, instead of sending a new message (which would be confusing for the user) - see finishAddFolder/finishAddBox/finishAddEntry in text_input.go
func (b *Bot) sendText(chatID int64, v textView) int {
	msg := tgbotapi.NewMessage(chatID, v.text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = withKeyboard(v.keyboard)

	sent, err := b.api.Send(msg)
	if err != nil {
		log.Printf("greška pri slanju poruke (chat_id=%d): %v", chatID, err)
		return 0
	}
	return sent.MessageID
}

func (b *Bot) editText(chatID int64, messageID int, v textView) {
	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, v.text, withKeyboard(v.keyboard))
	edit.ParseMode = tgbotapi.ModeHTML
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

// renderText vraća ID poruke koja je posle poziva zaista prikazana - isti
// cb.Message.MessageID ako je editovano na mestu, ili ID nove poruke ako je
// stara (sa slikom) morala da se obriše i pošalje nova tekstualna.
func (b *Bot) renderText(cb *tgbotapi.CallbackQuery, v textView) int {
	chatID := cb.Message.Chat.ID
	if len(cb.Message.Photo) == 0 {
		b.editText(chatID, cb.Message.MessageID, v)
		return cb.Message.MessageID
	}
	b.deleteMessage(chatID, cb.Message.MessageID)
	return b.sendText(chatID, v)
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
