package server

import (
//"time"
)

type Message struct {
	Id           int    `sql:"auto_increment;not null;primary_key"`
	Data         string `sql:"type:text;not null"`
	Priority     int    `sql:"default:0;not null;index:idx_priority"`
	NumFails     int    `sql:"default:0;not null;index:idx_num_fails"`
	LastFailedAt *int64 `sql:""`
	Failure      string `sql:""`
	SendAfter    *int64 `sql:"index:idx_send_after"`
	Sent         int    `sql:"default:0;not null;index:idx_sent"`
	SentAt       *int64 `sql:""`
	CreatedAt    int64  `sql:"not null; index:idx_created_at"`
	UpdatedAt    *int64 `sql:""`
}
