package model

import (
	"time"

	"github.com/gocql/gocql"
)

type Click struct {
	Code      string     `json:"code"`
	Bucket    string     `json:"bucket"`
	ClickedAt time.Time  `json:"clicked_at"`
	ClickID   gocql.UUID `json:"click_id"`
	Country   string     `json:"country"`
	Device    string     `json:"device"`
	Referer   string     `json:"referer"`
}

type ClickCount struct {
	Code   string `json:"code"`
	Bucket string `json:"bucket"`
	Total  int64  `json:"total"`
}
