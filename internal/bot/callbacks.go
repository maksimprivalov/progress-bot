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

const rootSentinel = "0"

// Nazivi "glagola" u callback_data - format je uvek "glagol:arg1[:arg2]".
// Npr. "open_box:12" ili "add_box_type:measure:5". Telegram ograničava
// callback_data na 64 bajta; svi naši payload-i su kratki (glagol + jedan
// ili dva broja/reči), pa smo daleko od tog limita.
const (
	verbRoot          = "root"
	verbOpenFolder    = "open_folder"
	verbOpenBox       = "open_box"
	verbAdd           = "add"
	verbAddFolder     = "add_folder"
	verbAddBox        = "add_box"
	verbAddBoxType    = "add_box_type"
	verbAddEntry      = "add_entry"
	verbBack          = "back"
	verbDeleteConfirm = "delete_confirm"
	verbDeleteDo      = "delete_do"
	verbShowChart     = "show_chart"
	verbStart         = "start"
)

// handleCallback rutira klik na inline dugme. Svi handleri ispod su
// namerno tolerantni na neočekivan callback_data (samo se tiho ignorišu uz
// log) - pošto MI generišemo svu dugmad koju korisnik vidi, do
// nevalidnog callback_data može doći jedino ako neko šalje ručno sastavljen
// zahtev mimo interfejsa koji nudimo.
func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
	userID := cb.From.ID
	parts := strings.Split(cb.Data, ":")

	// Telegram čeka da bot "odgovori" na klik (answerCallbackQuery) - dok se
	// to ne pozove, dugme na klijentu ostaje u stanju "učitava se" (mali sat
	// na dugmetu). To je odvojeno od slanja/editovanja same poruke - zato
	// ga radimo tačno jednom, na kraju, pošto svi handleri završe.
	// startAddEntry je jedini koji ima nešto specifično da poruči (npr.
	// "Already marked for today.") - drugi handleri ostavljaju toast prazan.
	var toast string
	switch parts[0] {
	case verbRoot:
		b.showRoot(cb, userID)
	case verbOpenFolder:
		if id, ok := parseID(parts, 1); ok {
			b.showFolder(cb, userID, id)
		}
	case verbOpenBox:
		if id, ok := parseID(parts, 1); ok {
			b.showBox(cb, userID, id)
		}
	case verbAdd:
		b.startAdd(cb, userID, parts)
	case verbAddFolder:
		b.startAddFolderName(cb, userID, parts)
	case verbAddBox:
		b.startAddBoxTypeChoice(cb, userID, parts)
	case verbAddBoxType:
		b.startAddBoxName(cb, userID, parts)
	case verbAddEntry:
		toast = b.startAddEntry(cb, userID, parts)
	case verbBack:
		b.handleBack(cb, userID, parts)
	case verbStart:
		b.handleStart(cb, userID)
	case verbDeleteConfirm:
		if id, ok := parseID(parts, 1); ok {
			b.showDeleteConfirm(cb, userID, id)
		}
	case verbDeleteDo:
		toast = b.performDelete(cb, userID, parts)
	case verbShowChart:
		if id, ok := parseID(parts, 1); ok {
			b.showChart(cb, userID, id)
		}
	default:
		log.Printf("nepoznat callback: %q", cb.Data)
	}

	b.answerCallback(cb.ID, toast)
}

func parseID(parts []string, idx int) (int64, bool) {
	if idx >= len(parts) {
		return 0, false
	}
	id, err := strconv.ParseInt(parts[idx], 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func encodeParent(id *int64) string {
	if id == nil {
		return rootSentinel
	}
	return strconv.FormatInt(*id, 10)
}

func parseParent(parts []string, idx int) (*int64, bool) {
	if idx >= len(parts) {
		return nil, false
	}
	if parts[idx] == rootSentinel {
		return nil, true
	}
	id, err := strconv.ParseInt(parts[idx], 10, 64)
	if err != nil {
		return nil, false
	}
	return &id, true
}

// ownsParent proverava da folder na koji callback_data ukazuje stvarno
// pripada korisniku koji je kliknuo dugme. Bez ove provere bi korisnik,
// slanjem ručno sastavljenog callback zahteva sa tuđim ID-jem foldera,
// mogao da pokuša da doda box u tuđi folder. parentID == nil (root nivo) je
// uvek dozvoljen - root ne pripada nikom posebno, svaki korisnik ima svoj.
func (b *Bot) ownsParent(userID int64, parentID *int64) bool {
	if parentID == nil {
		return true
	}
	item, err := b.storage.GetItem(*parentID)
	if err != nil || item.UserID != userID || item.Type != storage.ItemFolder {
		return false
	}
	return true
}

func (b *Bot) showRoot(cb *tgbotapi.CallbackQuery, userID int64) {
	b.state.clear(userID)
	items, err := b.storage.GetChildren(userID, nil)
	if err != nil {
		log.Printf("greška pri čitanju root menija (user_id=%d): %v", userID, err)
		b.renderText(cb, textView{text: "Something went wrong while reading your data."})
		return
	}
	b.renderText(cb, buildRootView(items))
}

func (b *Bot) showFolder(cb *tgbotapi.CallbackQuery, userID, folderID int64) {
	b.state.clear(userID)
	folder, err := b.storage.GetItem(folderID)
	if err != nil || folder.UserID != userID || folder.Type != storage.ItemFolder {
		log.Printf("greška pri otvaranju foldera %d (user_id=%d): %v", folderID, userID, err)
		b.renderText(cb, textView{text: "Folder not found."})
		return
	}
	items, err := b.storage.GetChildren(userID, &folderID)
	if err != nil {
		log.Printf("greška pri čitanju sadržaja foldera %d: %v", folderID, err)
		b.renderText(cb, textView{text: "Something went wrong while reading your data."})
		return
	}
	b.renderText(cb, buildFolderView(folder, items))
}

func (b *Bot) showBox(cb *tgbotapi.CallbackQuery, userID, boxID int64) {
	b.state.clear(userID)
	box, err := b.storage.GetItem(boxID)
	if err != nil || box.UserID != userID || box.Type != storage.ItemBox || box.BoxType == nil {
		log.Printf("greška pri otvaranju box-a %d (user_id=%d): %v", boxID, userID, err)
		b.renderText(cb, textView{text: "Tracker not found."})
		return
	}
	entries, err := b.storage.GetEntries(boxID)
	if err != nil {
		log.Printf("greška pri čitanju zapisa box-a %d: %v", boxID, err)
		b.renderText(cb, textView{text: "Something went wrong while reading your data."})
		return
	}

	switch *box.BoxType {
	case storage.BoxMeasure:
		b.showMeasureBox(cb, box, entries)
	case storage.BoxCheck:
		b.showCheckBox(cb, box, entries)
	}
}

// showChart obrađuje klik na "📊 Prikaži grafikon" sa summary ekrana
// measure box-a - učita box i zapise nezavisno od showBox (koji bi ovde
// samo opet prikazao summary, ne grafikon) i pozove showMeasureChart.
func (b *Bot) showChart(cb *tgbotapi.CallbackQuery, userID, boxID int64) {
	b.state.clear(userID)
	box, err := b.storage.GetItem(boxID)
	if err != nil || box.UserID != userID || box.Type != storage.ItemBox || box.BoxType == nil || *box.BoxType != storage.BoxMeasure {
		log.Printf("greška pri prikazu grafikona box-a %d (user_id=%d): %v", boxID, userID, err)
		return
	}
	entries, err := b.storage.GetEntries(boxID)
	if err != nil {
		log.Printf("greška pri čitanju zapisa box-a %d: %v", boxID, err)
		b.renderText(cb, textView{text: "Something went wrong while reading your data."})
		return
	}
	b.showMeasureChart(cb, box, entries)
}

// showMeasureBox se poziva kad se box otvori (npr. iz liste u folderu).
// Prikazuje samo tekstualni pregled - BEZ grafikona - i dugme "📊 Prikaži
// grafikon". Generisanje PNG-a (gonum/plot render + upload slike) je
// relativno skup posao za nešto što se radi na svaki ulazak u meni; ovako
// se radi samo kad korisnik eksplicitno zatraži grafikon (showMeasureChart
// ispod), umesto na svaki klik na box.
func (b *Bot) showMeasureBox(cb *tgbotapi.CallbackQuery, box storage.Item, entries []storage.Entry) {
	b.renderText(cb, buildMeasureSummaryView(box, entries))
}

// showMeasureChart generiše i prikazuje PNG grafikon (gonum/plot) svih
// zapisa box-a - poziva se na klik "📊 Prikaži grafikon". Pošto je
// prethodna poruka (summary) tekstualna, renderPhoto će je obrisati i
// poslati novu, identičnu po sadržaju ali sa slikom - Telegram ne dozvoljava
// da se tekstualna poruka editovanjem pretvori u poruku sa slikom.
func (b *Bot) showMeasureChart(cb *tgbotapi.CallbackQuery, box storage.Item, entries []storage.Entry) {
	keyboard := measureKeyboard(box)

	if len(entries) == 0 {
		b.renderText(cb, buildMeasureSummaryView(box, entries))
		return
	}

	points := pointsFromEntries(entries)
	png, err := chart.GenerateProgressPNG(box.Name, points)
	if err != nil {
		log.Printf("greška pri generisanju grafikona za Tracker %d: %v", box.ID, err)
		b.renderText(cb, textView{text: "Something went wrong while generating the chart.", keyboard: keyboard})
		return
	}

	last := entries[len(entries)-1]
	caption := fmt.Sprintf("📊 %s\n\nLatest entry: %.1f (%s)", box.Name, *last.Value, formatDisplayDate(last.EntryDate))
	b.renderPhoto(cb, caption, png, keyboard)
}

// showCheckBox prikazuje tekstualni pregled (streak, ukupno, mreža
// poslednjih dana) umesto grafikona - vidi obrazloženje u checkstats.go.
func (b *Bot) showCheckBox(cb *tgbotapi.CallbackQuery, box storage.Item, entries []storage.Entry) {
	keyboard := checkKeyboard(box)
	text := fmt.Sprintf("✅ %s\n\n%s", box.Name, formatCheckStats(entries))
	b.renderText(cb, textView{text: text, keyboard: keyboard})
}

// startAdd obrađuje "➕ Dodaj novo" - i na root nivou i unutar (proizvoljno
// ugnježdenog) foldera. Pošto folderi mogu sadržati i druge foldere i
// box-ove, izbor tipa (📁 Folder / 📦 Box) se uvek nudi, na svakom nivou.
func (b *Bot) startAdd(cb *tgbotapi.CallbackQuery, userID int64, parts []string) {
	parentID, ok := parseParent(parts, 1)
	if !ok || !b.ownsParent(userID, parentID) {
		return
	}

	parentArg := encodeParent(parentID)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📁 Folder", "add_folder:"+parentArg),
			tgbotapi.NewInlineKeyboardButtonData("📝 Tracker", "add_box:"+parentArg),
		),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⬅️ Back", "back:"+parentArg)),
	)
	b.renderText(cb, textView{text: "What do you want to add?", keyboard: keyboard})
}

func (b *Bot) startAddFolderName(cb *tgbotapi.CallbackQuery, userID int64, parts []string) {
	parentID, ok := parseParent(parts, 1)
	if !ok || !b.ownsParent(userID, parentID) {
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⬅️ Cancel", "back:"+encodeParent(parentID))),
	)
	messageID := b.renderText(cb, textView{text: "What should the new folder be called? Send me the name as a message.", keyboard: keyboard})
	b.state.set(userID, pendingAction{kind: pendingFolderName, parentID: parentID, promptMessageID: messageID})
}

func (b *Bot) startAddBoxTypeChoice(cb *tgbotapi.CallbackQuery, userID int64, parts []string) {
	parentID, ok := parseParent(parts, 1)
	if !ok || !b.ownsParent(userID, parentID) {
		return
	}
	b.showBoxTypeChoice(cb, parentID)
}

// showBoxTypeChoice pita korisnika da izabere podtip box-a - check (navika)
// ili measure (merenje). Ovaj izbor određuje sve dalje ponašanje box-a, pa
// se traži pre unosa imena, ne posle.
func (b *Bot) showBoxTypeChoice(cb *tgbotapi.CallbackQuery, parentID *int64) {
	parentArg := encodeParent(parentID)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Check (habit)", "add_box_type:check:"+parentArg),
			tgbotapi.NewInlineKeyboardButtonData("📊 Measure", "add_box_type:measure:"+parentArg),
		),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⬅️ Back", "back:"+parentArg)),
	)
	b.renderText(cb, textView{text: "What kind of box do you want to create?", keyboard: keyboard})
}

func (b *Bot) startAddBoxName(cb *tgbotapi.CallbackQuery, userID int64, parts []string) {
	if len(parts) < 3 {
		return
	}
	boxType := storage.BoxType(parts[1])
	if boxType != storage.BoxCheck && boxType != storage.BoxMeasure {
		return
	}
	parentID, ok := parseParent(parts, 2)
	if !ok || !b.ownsParent(userID, parentID) {
		return
	}

	label := "Measure"
	if boxType == storage.BoxCheck {
		label = "Check (habit)"
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⬅️ Cancel", "back:"+encodeParent(parentID))),
	)
	messageID := b.renderText(cb, textView{
		text:     fmt.Sprintf("What should the new %s Tracker be called? Send me the name as a message.", label),
		keyboard: keyboard,
	})
	b.state.set(userID, pendingAction{kind: pendingBoxName, parentID: parentID, boxType: boxType, promptMessageID: messageID})
}

func (b *Bot) startAddEntry(cb *tgbotapi.CallbackQuery, userID int64, parts []string) string {
	boxID, ok := parseID(parts, 1)
	if !ok {
		return ""
	}
	box, err := b.storage.GetItem(boxID)
	if err != nil || box.UserID != userID || box.Type != storage.ItemBox || box.BoxType == nil {
		log.Printf("greška pri dodavanju zapisa za Tracker  %d (user_id=%d): %v", boxID, userID, err)
		return "Tracker not found."
	}

	switch *box.BoxType {
	case storage.BoxMeasure:
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⬅️ Cancel", fmt.Sprintf("open_box:%d", box.ID))),
		)
		messageID := b.renderText(cb, textView{
			text:     fmt.Sprintf("📊 %s\n\nSend me the new value (a number), e.g. 82.5", box.Name),
			keyboard: keyboard,
		})
		b.state.set(userID, pendingAction{kind: pendingEntryValue, boxID: boxID, promptMessageID: messageID})
		return ""
	case storage.BoxCheck:
		return b.markCheckDoneAndRefresh(cb, box)
	default:
		return ""
	}
}

func (b *Bot) markCheckDoneAndRefresh(cb *tgbotapi.CallbackQuery, box storage.Item) string {
	today := time.Now().UTC().Format(storage.DateFormat)
	alreadyMarked, err := b.storage.MarkCheckDone(box.ID, today)
	if err != nil {
		log.Printf("greška pri obeležavanju Tracker-a %d: %v", box.ID, err)
		return "Something went wrong. Please try again."
	}

	entries, err := b.storage.GetEntries(box.ID)
	if err != nil {
		log.Printf("greška pri čitanju zapisa Tracker-a %d: %v", box.ID, err)
		return "Something went wrong. Please try again."
	}
	b.showCheckBox(cb, box, entries)

	if alreadyMarked {
		return "Already marked for today."
	}
	return "Marked as done!"
}

func (b *Bot) handleStart(cb *tgbotapi.CallbackQuery, userID int64) {
	b.state.clear(userID)
	b.showRoot(cb, userID)
}

func (b *Bot) handleBack(cb *tgbotapi.CallbackQuery, userID int64, parts []string) {
	parentID, ok := parseParent(parts, 1)
	if !ok {
		return
	}
	b.state.clear(userID)
	if parentID == nil {
		b.showRoot(cb, userID)
		return
	}
	b.showFolder(cb, userID, *parentID)
}
