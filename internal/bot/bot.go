package bot

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"progress-bot/internal/storage"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	storage *storage.Storage
	state   *conversationState
}

func New(token string, store *storage.Storage) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("kreiranje Telegram API klijenta: %w", err)
	}
	return &Bot{api: api, storage: store, state: newConversationState()}, nil
}

func (b *Bot) Run() {
	log.Printf("Bot pokrenut kao @%s", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		switch {
		case update.CallbackQuery != nil:
			b.handleCallback(update.CallbackQuery)
		case update.Message != nil:
			b.handleMessage(update.Message)
		}
	}
}
