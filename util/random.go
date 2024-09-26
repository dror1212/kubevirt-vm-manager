package util

import (
	"math/rand"
	"time"
)

// generateRandomName generates a random VM name
func generateRandomName() string {
	rand.Seed(time.Now().UnixNano())
	letters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	name := make([]rune, 8)
	for i := range name {
		name[i] = letters[rand.Intn(len(letters))]
	}
	return string(name)
}