package models

import "time"

type Room struct {
	ID            uint `gorm:"primaryKey"`
	Name          string
	Type          string // private | group
	CreatedBy     uint
	LastMessage   string    `gorm:"type:text"`
	LastMessageAt time.Time `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
