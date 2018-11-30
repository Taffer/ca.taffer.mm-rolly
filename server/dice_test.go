package main

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

// -----------------------------------------------------------------------------
// Dice rollin'.
// -----------------------------------------------------------------------------

// TestHandleRoll - Make sure different sorts of rolls do things.
func TestHandleRoll(t *testing.T) {
	p := initTestPlugin(t)
	p.Init()

	// simplePattern matches
	rand.Seed(0) // Make these deterministic.
	response := p.HandleRoll("6", "")
	assert.EqualValues(t, response, `"1d6" = **1**`)

	response = p.HandleRoll("%", "")
	assert.EqualValues(t, response, `"1d%" = **15**`)

	response = p.HandleRoll("F", "")
	assert.EqualValues(t, response, `"1dF" = **0**`)

	// comboPattern matches
	rand.Seed(0) // Make these deterministic.
	response = p.HandleRoll("dnd", "")
	assert.EqualValues(t, response, "D&D standard:\n* 3d6 [1 1 2] = **4**\n* 3d6 [5 5 6] = **16**\n* 3d6 [1 2 6] = **9**\n* 3d6 [1 1 6] = **8**\n* 3d6 [1 1 6] = **8**\n* 3d6 [1 3 6] = **10**")

	response = p.HandleRoll("dnd+", "")
	assert.EqualValues(t, response, "D&D variant:\n* 4d6<1 [1 1 5 6] = **12**\n* 4d6<1 [1 1 3 6] = **10**\n* 4d6<1 [1 2 5 5] = **12**\n* 4d6<1 [3 4 5 6] = **15**\n* 4d6<1 [2 3 3 5] = **11**\n* 4d6<1 [1 1 2 6] = **9**")

	response = p.HandleRoll("open", "")
	assert.EqualValues(t, response, "Rolemaster open-ended: 1d% [77] = **77**")

	// rollPattern matches
	rand.Seed(0) // Make these deterministic.
	response = p.HandleRoll("d3", "")
	assert.EqualValues(t, response, `"d3" = **1**`)

	response = p.HandleRoll("2d3", "")
	assert.EqualValues(t, response, `"2d3" [1 2] = **3**`)

	response = p.HandleRoll("1000d6", "")
	assert.EqualValues(t, response, "1000 is too many, rolling 100.\n\"1000d6\" [1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 2 2 2 2 2 2 2 2 2 2 2 2 2 3 3 3 3 3 3 3 3 3 3 3 3 3 3 3 3 3 3 3 4 4 4 4 4 4 4 4 4 4 5 5 5 5 5 5 5 5 5 5 5 5 5 5 5 5 6 6 6 6 6 6 6 6 6 6 6 6 6 6 6] = **320**")

	response = p.HandleRoll("6!", "")
	assert.EqualValues(t, response, "\"6!\" [6 1] = **7**")

	response = p.HandleRoll("monkey", "")
	assert.EqualValues(t, response, "I have no idea what to do with this: monkey")

	response = p.HandleRoll("1", "")
	assert.EqualValues(t, response, "Your one-sided die rolls off into the shadows.")

	response = p.HandleRoll("1d1", "")
	assert.EqualValues(t, response, "Your one-sided die rolls off into the shadows.")
}

// TestRollDice - Make sure different combinations return correct values.
func TestRollDice(t *testing.T) {
	p := initTestPlugin(t)
	p.Init()

	// Different number of sides.
	rand.Seed(0) // Make these deterministic.
	rolls, total := p.RollDice(1, "%", "", 0)
	assert.EqualValues(t, rolls[0], 75)
	assert.EqualValues(t, total, 75)

	rolls, total = p.RollDice(1, "F", "", 0)
	assert.EqualValues(t, rolls[0], -1)
	assert.EqualValues(t, total, -1)

	rolls, total = p.RollDice(1, "1", "", 0)
	assert.EqualValues(t, rolls[0], 2)
	assert.EqualValues(t, total, 2)

	// Different modifiers.
	rand.Seed(0) // Make these deterministic.
	rolls, total = p.RollDice(1, "6", "+", 1)
	assert.EqualValues(t, rolls[0], 1)
	assert.EqualValues(t, total, 2)

	rolls, total = p.RollDice(1, "6", "-", 6)
	assert.EqualValues(t, rolls[0], 1)
	assert.EqualValues(t, total, 1)

	rolls, total = p.RollDice(1, "6", "/", 2)
	assert.EqualValues(t, rolls[0], 2)
	assert.EqualValues(t, total, 1)

	rolls, total = p.RollDice(1, "6", "x", 2)
	assert.EqualValues(t, rolls[0], 5)
	assert.EqualValues(t, total, 10)

	rand.Seed(0) // Make these deterministic.
	rolls, total = p.RollDice(2, "6", "<", 1)
	assert.EqualValues(t, rolls[0], 1)
	assert.EqualValues(t, rolls[1], 1)
	assert.EqualValues(t, total, 1)

	rolls, total = p.RollDice(2, "6", ">", 1)
	assert.EqualValues(t, rolls[0], 2)
	assert.EqualValues(t, rolls[1], 5)
	assert.EqualValues(t, total, 5)

	rolls, total = p.RollDice(2, "6", ">", 3)
	assert.EqualValues(t, rolls[0], 5)
	assert.EqualValues(t, rolls[1], 6)
	assert.EqualValues(t, total, 11)

	rolls, total = p.RollDice(2, "6", ">", 0)
	assert.EqualValues(t, rolls[0], 2)
	assert.EqualValues(t, rolls[1], 6)
	assert.EqualValues(t, total, 0)

	// Need to roll these several times before it actually explodes.
	rand.Seed(0) // Make these deterministic.
	rolls, total = p.RollDice(1, "F", "!", 0)
	rolls, total = p.RollDice(1, "F", "!", 0)
	rolls, total = p.RollDice(1, "F", "!", 0)
	rolls, total = p.RollDice(1, "F", "!", 0)
	rolls, total = p.RollDice(1, "F", "!", 0)
	assert.EqualValues(t, rolls[0], 1)
	assert.EqualValues(t, rolls[1], 0)
	assert.EqualValues(t, total, 1)
}
