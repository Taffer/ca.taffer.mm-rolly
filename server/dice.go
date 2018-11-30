package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// HandleRoll - Handle a rolling command.
//
// Returns the adjusted roll output.
func (p *RollyPlugin) HandleRoll(rollArg string, rollText string) string {
	if p.simplePattern.MatchString(rollArg) == true {
		// Simple roll (number only).
		matches := FindNamedSubstrings(p.simplePattern, rollArg)

		if matches["num_sides"] == "1" {
			rollText += "Your one-sided die rolls off into the shadows."
		} else {
			dice, total := p.RollDice(1, matches["num_sides"], "", 0)

			if len(dice) == 1 {
				rollText += fmt.Sprintf("\"1d%v\" = **%d**", rollArg, total)
			} else {
				rollText += fmt.Sprintf("%q %v = **%d**", rollArg, dice, total)
			}
		}

	} else if p.comboPattern.MatchString(rollArg) == true {
		// C-C-C-C-COMBO roll.
		matches := FindNamedSubstrings(p.comboPattern, rollArg)

		comboName := strings.ToLower(matches["combo_name"])
		switch comboName {
		case "dnd", "d&d":
			// D&D/Pathfinder: 3d6 for each stat.
			rollText += "D&D standard:"

			for idx := 0; idx < 6; idx++ {
				dice, total := p.RollDice(3, "6", "", 0)
				rollText += fmt.Sprintf("\n* 3d6 %v = **%d**", dice, total)
			}
		case "dnd+", "d&d+":
			// Common D&D/Pathfinder house rule: 4d6<1 for each stat.
			rollText += "D&D variant:"

			for idx := 0; idx < 6; idx++ {
				dice, total := p.RollDice(4, "6", "<", 1)
				rollText += fmt.Sprintf("\n* 4d6<1 %v = **%d**", dice, total)
			}
		case "open":
			// Rolemaster open-ended d%.
			dice, total := p.RollDice(1, "%", "", 0)
			allDice := dice
			for total >= 95 {
				dice, total = p.RollDice(1, "%", "", 0)
				allDice = append(allDice, dice[0])
			}

			sort.Ints(allDice)
			total = sum(allDice)
			rollText += fmt.Sprintf("Rolemaster open-ended: 1d%% %v = **%d**", allDice, total)
		default:
			// You can't actually reach this with the current regex.
			rollText += fmt.Sprintf("Combo **%v** isn't implemented yet, sorry.", rollArg)
		}

	} else if p.rollPattern.MatchString(rollArg) == true {
		// Typical roll (number of dice, sides, optional modifiers).
		matches := FindNamedSubstrings(p.rollPattern, rollArg)

		numDice, err := strconv.Atoi(matches["num_dice"])
		if err != nil {
			// This is optional, so it might be empty.
			numDice = 1
		}
		if numDice > 100 {
			rollText += fmt.Sprintf("%v is too many, rolling 100.\n", numDice)
			numDice = 100
		}
		if numDice < 1 {
			rollText += fmt.Sprintf("%v is too few, rolling 1.\n", numDice)
			numDice = 1
		}
		sides := matches["num_sides"] // Left as string for d% rolls.
		if sides == "1" {
			rollText += "Your one-sided die rolls off into the shadows."
		} else {
			modifier := matches["modifier"]
			modifierValue, err := strconv.Atoi(matches["modifier_value"])
			if err != nil {
				modifierValue = 0 // One wasn't specified. Blame the ! modifier.
			}

			dice, total := p.RollDice(numDice, sides, modifier, modifierValue)

			if len(dice) == 1 {
				rollText += fmt.Sprintf("%q = **%d**", rollArg, total)
			} else {
				rollText += fmt.Sprintf("%q %v = **%d**", rollArg, dice, total)
			}
		}
	} else {
		rollText += fmt.Sprintf("I have no idea what to do with this: %v", rollArg)
	}

	return rollText
}

// RollDice - Roll {dice}d{sides}{modifier}{modifier_value}.
//
// Returns an array of rolls, and the (modified) total.
func (p *RollyPlugin) RollDice(dice int, sides string, modifier string, modifierValue int) ([]int, int) {
	var rolls []int

	// Valid dieSides are digits, or %.
	var dieSides int
	if sides == "%" {
		dieSides = 100
	} else if sides == "F" {
		dieSides = 3
	} else {
		dieSides, _ = strconv.Atoi(sides)
		if dieSides < 2 {
			dieSides = 2
		}
	}

	for idx := 0; idx < dice; idx++ {
		value := p.GetRandom(dieSides)
		if sides == "F" {
			value -= 2 // FUDGE dice product -1, 0, 1
		}
		rolls = append(rolls, value)
	}

	sort.Ints(rolls)
	total := sum(rolls)

	// Most of the supported modifiers are trivial.
	switch modifier {
	case "+":
		total += modifierValue
	case "-":
		total -= modifierValue
		if total < 1 && sides != "F" {
			total = 1 // Clamp to 1, unless FUDGE.
		}
	case "/":
		if modifierValue > 0 {
			total /= modifierValue
		}
	case "x", "*":
		total *= modifierValue
	case "<": // Ignore the lowest modifierValue rolls.
		cutoff := min(modifierValue, len(rolls)-1)

		total = sum(rolls[cutoff:])
	case ">": // Keep the best modifierValue rolls.
		if modifierValue >= len(rolls) {
			total = sum(rolls)
		} else if modifierValue < 1 {
			total = 0
		} else {
			cutoff := len(rolls) - modifierValue

			total = sum(rolls[cutoff:])
		}
	case "!": // Exploding dice!
		explode := 0
		for idx := len(rolls) - 1; idx >= 0; idx-- {
			if (sides == "F" && rolls[idx] == 1) || rolls[idx] == dieSides {
				explode++
			}
		}

		for idx := 0; idx < explode; idx++ {
			boom := p.GetRandom(dieSides)
			if sides == "F" { // FUDGE is a special case.
				boom -= 2
				if boom == 1 {
					explode++
				}
			} else { // Normal case.
				if boom == dieSides {
					explode++
				}
			}
			rolls = append(rolls, boom)
		}

		total = sum(rolls)
	}

	return rolls, total
}
