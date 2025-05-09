package utils

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

var (
	adjectives = []string{"Mystic", "Cosmic", "Lost", "Forgotten", "Hidden", "Ancient", "Future", "Electric", "Crimson", "Azure", "Golden", "Silent", "Whispering", "Eternal", "Fleeting"}
	nouns      = []string{"Echo", "Dream", "Journey", "Silence", "Melody", "Rhythm", "Star", "Nebula", "Odyssey", "Mirage", "Dawn", "Twilight", "Abyss", "Sanctuary", "Lullaby"}
	suffixes   = []string{"Sonata", "Etude", "Nocturne", "Rhapsody", "Ballad", "Serenade", "Overture", "Fantasy", "Caprice"}
	states     = []string{"in D Minor", "Reimagined", "Ascending", "Unfolding", "Awakening", "Drifting", "Resonating"}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateCreativeTitle(musicType, modelType string) string {
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]

	var titlePart string
	num := rand.Intn(100)

	switch rand.Intn(4) {
	case 0:
		titlePart = fmt.Sprintf("%s %s", adj, noun)
	case 1:
		suffix := suffixes[rand.Intn(len(suffixes))]
		titlePart = fmt.Sprintf("%s's %s", adj, suffix)
	case 2:
		state := states[rand.Intn(len(states))]
		titlePart = fmt.Sprintf("%s %s %s", noun, state, musicType)
	default:
		titlePart = fmt.Sprintf("%s %s of the %s", adj, noun, strings.Title(strings.ToLower(modelType)))
	}

	return fmt.Sprintf("%s #%d", titlePart, num)
}
