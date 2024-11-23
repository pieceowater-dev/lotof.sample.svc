package ent

import "gorm.io/gorm"

type Todo struct {
	gorm.Model
	ID       int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Text     string `json:"text" gorm:"type:varchar(255);not null"`
	Category string `json:"category" gorm:"type:varchar(50);not null"`
	Done     bool   `json:"done" gorm:"default:false"`
}
