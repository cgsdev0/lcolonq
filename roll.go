package main

import "github.com/justinian/dice"

func Roll(what string) int {
	result, _, err := dice.Roll(what)
	if err != nil {
		panic(err)
	}
	return result.Int()
}
