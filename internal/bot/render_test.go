package bot

import (
	"encoding/json"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TestWithKeyboardNeverMarshalsNull čuva regresiju: bez ove normalizacije,
// editovanje/slanje poruke sa praznom (zero-value) tastaturom je padalo sa
// "Bad Request: field \"inline_keyboard\" must be of type Array", jer se
// nil slice u Go-u marshaluje u JSON kao "null", ne "[]". Do toga je došlo
// na svakom prompt-ekranu (unos imena, unos vrednosti) koji nije eksplicitno
// postavljao keyboard polje - zbog čega se folderi/box-ovi u praksi nisu ni
// kreirali (korisnik nikad nije video zahtev za unos imena).
func TestWithKeyboardNeverMarshalsNull(t *testing.T) {
	// Zero-value textView (kao npr. textView{text: "..."} bez keyboard polja).
	kb := withKeyboard(tgbotapi.InlineKeyboardMarkup{})

	raw, err := json.Marshal(kb)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	const want = `{"inline_keyboard":[]}`
	if string(raw) != want {
		t.Errorf("json.Marshal(withKeyboard(zero-value)) = %s, want %s", raw, want)
	}
}

func TestWithKeyboardPreservesExistingRows(t *testing.T) {
	original := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("A", "a")),
	)

	kb := withKeyboard(original)
	if len(kb.InlineKeyboard) != 1 || len(kb.InlineKeyboard[0]) != 1 {
		t.Fatalf("withKeyboard() = %+v, očekivan nepromenjen sadržaj sa 1 dugmetom", kb)
	}
}
