package main

import (
	"fmt"
	"math/rand"
)

// From MIT https://github.com/yelinaung/go-haikunator
var (
	ADJECTIVES = []string{"autumn", "hidden", "bitter", "misty", "silent", "empty", "dry", "dark", "summer",
		"icy", "delicate", "quiet", "white", "cool", "spring", "winter", "patient",
		"twilight", "dawn", "crimson", "wispy", "weathered", "blue", "billowing",
		"broken", "cold", "damp", "falling", "frosty", "green", "long", "late", "lingering",
		"bold", "little", "morning", "muddy", "old", "red", "rough", "still", "small",
		"sparkling", "throbbing", "shy", "wandering", "withered", "wild", "black",
		"young", "holy", "solitary", "fragrant", "aged", "snowy", "proud", "floral",
		"tough", "lucky", "evil", "bold", "little", "adament", "boorish", "zealous",
		"restless", "divine", "polished", "ancient", "purple", "lively", "nameless"}
	NOUNS = []string{"waterfall", "river", "breeze", "moon", "rain", "wind", "sea", "morning",
		"snow", "lake", "sunset", "pine", "shadow", "leaf", "dawn", "glitter", "forest",
		"hill", "cloud", "meadow", "sun", "glade", "bird", "brook", "butterfly",
		"bush", "dew", "dust", "field", "fire", "flower", "firefly", "feather", "grass",
		"haze", "mountain", "night", "pond", "darkness", "snowflake", "silence",
		"sound", "sky", "shape", "surf", "thunder", "violet", "water", "wildflower",
		"wave", "water", "resonance", "sun", "wood", "dream", "cherry", "tree", "fog",
		"frost", "voice", "paper", "frog", "smoke", "star", "dog", "cat",
		"ocean", "book"}
)

type Name interface {
	Haikunate() string
}

type RandomName struct {
	r *rand.Rand
}

func (r RandomName) Haikunate() string {
	return fmt.Sprintf("%v-%v", ADJECTIVES[r.r.Intn(len(ADJECTIVES))], NOUNS[r.r.Intn(len(NOUNS))])
}

func NewRandomName(seed int64) Name {
	name := RandomName{rand.New(rand.New(rand.NewSource(seed)))}
	name.r.Seed(seed)
	return name
}
