package main

import (
	_ "embed"
)

//go:embed art/bat
var batArt string

func createEnemy(c byte) *Enemy {
	switch c {
	case ENEMY_BAT:
		return &Enemy{
			name:      "a bat",
			level:     1,
			health:    5,
			maxhealth: 5,
			art:       leftpad(batArt, 5),
			ac:        12,
			damage:    "1d1",
			attack:    "1d20",
		}
	default:
		// this should never happen!
		return &Enemy{
			name:      "MISSINGNO",
			health:    100,
			maxhealth: 100,
		}
	}
}
