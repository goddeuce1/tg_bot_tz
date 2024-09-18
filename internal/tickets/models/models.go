package models

type Ticket struct {
	ChatID      int64  `json:"chat_id"`
	Unit        string `json:"unit"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
