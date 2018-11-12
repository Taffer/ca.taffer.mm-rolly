package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/mattermost/mattermost-server/plugin"
)

func main() {
	rolly := &RollyPlugin{}

	// Seed the pseudo-random number generator with sort of random data.
	// This only needs to be done once per instance.
	rolly.SeedRng()

	if len(os.Args) > 1 {
		// Ad-hoc testing... runs HandleRoll() on command-line args.
		rolly.Init()
		rand.Seed(0) // Make these deterministic.

		for idx := 1; idx < len(os.Args); idx++ {
			fmt.Println(idx, "=", rolly.HandleRoll(os.Args[idx], "Arg: "))
		}
	} else {
		plugin.ClientMain(rolly)
	}
}
