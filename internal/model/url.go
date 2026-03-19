package model

import "time"

type URL struct {
	Code      string     `json:"code"`
	Original  string     `json:"original_url"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}
