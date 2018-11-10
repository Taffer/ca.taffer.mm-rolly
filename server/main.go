package main

import (
	"math/rand"
	"time"

	"github.com/mattermost/mattermost-server/plugin"
)

func main() {
	// Seed the pseudo-random number generator with sort of random data.
	rand.Seed(time.Now().UnixNano())

	plugin.ClientMain(&RollyPlugin{})
}
