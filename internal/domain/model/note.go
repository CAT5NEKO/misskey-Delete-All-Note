package model

import "time"

type NoteID string

type Note struct {
	ID          NoteID         `json:"id"`
	CreatedAt   time.Time      `json:"createdAt"`
	Text        *string        `json:"text"`
	CW          *string        `json:"cw"`
	RenoteID    *NoteID        `json:"renoteId"`
	Renote      *Note          `json:"renote"`
	Reactions   map[string]int `json:"reactions"`
	RenoteCount int            `json:"renoteCount"`
}

func (n *Note) hasOwnContent() bool {
	return (n.CW != nil && *n.CW != "") || (n.Text != nil && *n.Text != "")
}

func (n *Note) IsRenote() bool {
	return n.RenoteID != nil || n.Renote != nil
}

func (n *Note) IsQuoteRenote() bool {
	return n.IsRenote() && n.hasOwnContent()
}

func (n *Note) KindLabel() string {
	if n.IsQuoteRenote() {
		return "quote-renote"
	}
	if n.IsRenote() {
		return "renote"
	}
	return "note"
}

func (n *Note) ShouldKeep(config *AppConfig) bool {
	if config.KeepWithReactions && len(n.Reactions) > 0 {
		return true
	}
	if config.KeepWithRenotes && n.RenoteCount > 0 {
		return true
	}
	return false
}

func (n *Note) GetSummary() string {
	content := ""
	if n.CW != nil && *n.CW != "" {
		content = *n.CW
	} else if n.Text != nil && *n.Text != "" {
		content = *n.Text
	}

	runes := []rune(content)
	if len(runes) > 20 {
		return string(runes[:20]) + "..."
	}
	return string(runes)
}
