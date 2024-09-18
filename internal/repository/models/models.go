package models

type Ticket struct {
	ID          uint64 `db:"id"`
	ChatID      int64  `db:"chat_id"`
	Unit        string `db:"unit"`
	Name        string `db:"name"`
	Description string `db:"description"`
}

type Admin struct {
	ID     uint64 `db:"id"`
	ChatID int64  `db:"chat_id"`
	Unit   string `db:"unit"`
}
