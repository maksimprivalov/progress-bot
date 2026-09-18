package bot

import (
	"sync"

	"progress-bot/internal/storage"
)

// to store the current state of the conversation with each user, including what kind of input is expected next (folder name, box name, entry value) and any relevant context (parent ID, box type, box ID). This allows the bot to handle multi-step interactions with users in a stateful manner.
type pendingKind int

const (
	pendingNone pendingKind = iota
	pendingFolderName
	pendingBoxName
	pendingEntryValue
)

// for navigationg through the callback menu.
type pendingAction struct {
	kind     pendingKind
	parentID *int64
	boxType  storage.BoxType
	boxID    int64

	// bot question id; when bot is waiting for user input, it sends a question to the user and stores the message ID of that question. When the user responds, the bot can edit that specific message instead of sending a new one, keeping the chat clean and organized.
	promptMessageID int
}

type conversationState struct {
	mu      sync.Mutex
	pending map[int64]pendingAction
}

func newConversationState() *conversationState {
	return &conversationState{pending: make(map[int64]pendingAction)}
}

func (c *conversationState) set(userID int64, action pendingAction) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pending[userID] = action
}

func (c *conversationState) get(userID int64) (pendingAction, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	action, ok := c.pending[userID]
	return action, ok
}

func (c *conversationState) clear(userID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pending, userID)
}
