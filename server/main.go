package main

import (
	"github.com/mattermost/mattermost-server/plugin"
)

func main() {
	rolly := &RollyPlugin{}

	// Seed the pseudo-random number generator with sort of random data.
	// This only needs to be done once per instance.
	rolly.SeedRng()

	plugin.ClientMain(rolly)
}
