package names

import (
	"fmt"
	"math/rand"
)

var adjectives = []string{
	"brave", "calm", "cool", "dark", "dawn", "deep", "fast", "fire",
	"free", "gold", "gray", "haze", "high", "iron", "keen", "kind",
	"late", "lean", "long", "loud", "mint", "near", "next", "odd",
	"pale", "pink", "pure", "red", "rich", "rust", "safe", "shy",
	"slim", "slow", "snow", "soft", "stark", "sun", "tall", "thin",
	"true", "vast", "warm", "west", "wide", "wild", "zen",
}

var animals = []string{
	"ant", "ape", "bat", "bee", "cat", "cod", "cow", "cub", "doe",
	"dog", "eel", "elk", "emu", "fly", "fox", "gnu", "hen", "hog",
	"jay", "kit", "koi", "lark", "lynx", "mink", "moth", "mule",
	"newt", "owl", "ox", "pug", "ram", "rat", "ray", "seal", "slug",
	"swan", "toad", "wasp", "wren", "yak", "bass", "bear", "boar",
	"crow", "dove", "duck", "fawn", "frog", "gull", "hare", "hawk",
}

// Generate returns a human-readable session name like "shimmer-frog-8334".
func Generate() string {
	adj := adjectives[rand.Intn(len(adjectives))]
	animal := animals[rand.Intn(len(animals))]
	num := rand.Intn(9000) + 1000
	return fmt.Sprintf("%s-%s-%d", adj, animal, num)
}
