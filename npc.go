package main

import (
	_ "embed"
)

// NPCS
const (
	NPC_SIGN      = iota // 254
	NPC_INNKEEPER        // 253
	NPC_PERSON           // 252
)

//go:embed art/sign
var signArt string

//go:embed art/npc
var guyArt string

//go:embed art/innkeeper
var innkeeperArt string

type NPC struct {
	name     string
	art      string
	dialogue string
}

func createNPC(c byte, dialogue string) *NPC {
	switch c {
	case NPC_SIGN:
		return &NPC{
			name:     "a sign",
			art:      leftpad(signArt, 8),
			dialogue: dialogue,
		}
	case NPC_INNKEEPER:
		return &NPC{
			name:     "the innkeeper",
			art:      leftpad(innkeeperArt, 5),
			dialogue: dialogue,
		}
	case NPC_PERSON:
		return &NPC{
			name:     "a person",
			art:      leftpad(guyArt, 8),
			dialogue: dialogue,
		}
	default:
		// this should never happen!
		return &NPC{
			name:     "badcop_",
			dialogue: "wtf u found me",
		}
	}
}
