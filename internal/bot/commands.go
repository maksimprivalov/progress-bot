package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const helpText = `Bot za praćenje napretka.

Podaci su organizovani u stablo:
📁 Folder - grupiše foldere i box-ove (npr. "Zdravlje" -> "Trening" -> ...). Folderi mogu biti proizvoljno ugnježdeni.
📦 Box - list stabla, čuva zapise. Pri kreiranju biraš jedan od dva podtipa:
   📊 Measure - numerička vrednost po danu (kilaža, sekunde, ponavljanja...) - prikazuje se grafikon
   ✅ Check - navika koju samo obeležavaš kao odrađenu tog dana - prikazuje se niz (streak) i pregled poslednjih dana

Komande:
/menu - otvori glavni meni
/help - prikaži ovu poruku`

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
		b.reply(msg.Chat.ID, "Nepoznata komanda. Kucaj /help za listu komandi.")
	}
}

func (b *Bot) sendRootMenu(chatID, userID int64) {
	b.state.clear(userID)
	items, err := b.storage.GetChildren(userID, nil)
	if err != nil {
		log.Printf("greška pri čitanju root menija (user_id=%d): %v", userID, err)
		b.reply(chatID, "Došlo je do greške pri čitanju podataka.")
		return
	}
	b.sendText(chatID, buildRootView(items))
}

func (b *Bot) reply(chatID int64, text string) {
	if _, err := b.api.Send(tgbotapi.NewMessage(chatID, text)); err != nil {
		log.Printf("greška pri slanju poruke (chat_id=%d): %v", chatID, err)
	}
}
