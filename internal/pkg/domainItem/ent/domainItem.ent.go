package ent

import "gorm.io/gorm"

type SomeEnum int

const (
	HELLO SomeEnum = iota
	WORLD
)

type Something struct {
	gorm.Model
	ID       int      `json:"id" gorm:"primaryKey;autoIncrement"`
	SomeEnum SomeEnum `json:"someEnum"`
}
