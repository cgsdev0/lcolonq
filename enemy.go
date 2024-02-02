package main

import (
	_ "embed"
)

// ENEMIES
const (
	ENEMY_BAT      = iota // 255
	ENEMY_SKELETON        // 254
	ENEMY_MINOTAUR        // 253
	ENEMY_GHOSTS          // 252
	ENEMY_KING            // 251
)

//go:embed art/bat
var batArt string

//go:embed art/minotaur
var minotaurArt string

//go:embed art/skeleton
var skeletonArt string

//go:embed art/ghosts
var ghostsArt string

//go:embed art/king
var kingArt string

type Enemy struct {
	id        int
	health    int
	maxhealth int
	name      string
	art       string
	level     int
	ac        int
	attack    string
	damage    string
}

func createEnemy(c byte) *Enemy {
	switch c {
	case ENEMY_BAT:
		return &Enemy{
			id:        ENEMY_BAT,
			name:      "a bat",
			level:     1,
			health:    5,
			maxhealth: 5,
			art:       leftpad(batArt, 5),
			ac:        12,
			damage:    "1d1",
			attack:    "1d20",
		}
	case ENEMY_SKELETON:
		return &Enemy{
			id:        ENEMY_SKELETON,
			name:      "a skeleton",
			level:     2,
			health:    10,
			maxhealth: 10,
			art:       leftpad(skeletonArt, 8),
			ac:        12,
			damage:    "1d4+1",
			attack:    "1d20+1",
		}
	case ENEMY_GHOSTS:
		return &Enemy{
			id:        ENEMY_GHOSTS,
			name:      "the ghosts",
			level:     3,
			health:    22,
			maxhealth: 22,
			art:       leftpad(ghostsArt, 3),
			ac:        15,
			damage:    "1d6",
			attack:    "1d20+1",
		}
	case ENEMY_MINOTAUR:
		return &Enemy{
			id:        ENEMY_MINOTAUR,
			name:      "the minotaur",
			level:     4,
			health:    18,
			maxhealth: 18,
			art:       leftpad(minotaurArt, 5),
			ac:        12,
			damage:    "1d10",
			attack:    "1d20+1",
		}
	case ENEMY_KING:
		return &Enemy{
			name:      "the mad king",
			id:        ENEMY_KING,
			level:     20,
			health:    45,
			maxhealth: 45,
			art:       leftpad(kingArt, 10),
			ac:        15,
			damage:    "2d6+2",
			attack:    "1d20+2",
		}
	default:
		// this should never happen!
		return &Enemy{
			id:        -1,
			name:      "MISSINGNO",
			health:    100,
			maxhealth: 100,
		}
	}
}
