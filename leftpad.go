package main

import "strings"

func leftpad(art string, n int) string {
	thing := strings.Split(art, "\n")
	var pad = strings.Repeat(" ", n)
	for i := range thing {
		thing[i] = pad + thing[i]
	}
	return strings.Join(thing, "\n")
}
