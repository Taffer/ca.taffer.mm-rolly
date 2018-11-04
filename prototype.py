#!/usr/bin/env python3
'''
Prototype dice roller for mm-rolly, a "deluxe" dice rolling plugin for
Mattermost.

Written by chrish@pobox.com; MIT licensed, see LICENSE for details.

From the "Goals" section of the README.md:

Support "any" reasonable dice rolling request:

* xdy or xDy to roll a y sided die x times
* modifiers: xdy+z, xdy-z (with a minimum of 1)
* xd% - same as xd100
* xdy/z - divide the result by z
* xdy<1 - discards the lowest roll (so 4d6<1 would return a value between 3 and 18)

(
    (?P<num_dice>
        [0-9]+
    )?
    d
)?
(?P<num_sides>
    [0-9\%]+
)
(
    (?P<modifier>
        [+-/<]
    )
    (?P<modifier_value>
        [0-9]+
    )
)?

Nerd combos:

* dnd - same as 3d6 six times
* dnd+ - same as 4d6<1 six times
* open - Rolemaster style open-ended roll

(
    (?P<combo_name>
        (d[n&]d|open)
    )
    (?P<combo_flag>
        \+
    )?
)

Number of dice per roll will be limited to 100 so malicious users can't flood
the channel with dice output.

Number of rolls per request (/roll 1d6 2d6 ... n) will be limited to 10 so
malicious users can't flood the channel with dice output.
'''

import random
import re
import secrets
import sys

SIMPLE_PATTERN = re.compile(
    r'''^(?P<num_sides>
        [0-9\%]+
    )$''', re.IGNORECASE | re.VERBOSE)

COMBO_PATTERN = re.compile(
    r'''^(
        (?P<combo_name>
            (d[n&]d|open)
        )
        (?P<combo_flag>
            \+
        )?
    )$''', re.IGNORECASE | re.VERBOSE)

ROLL_PATTERN = re.compile(
    r'''^(
        (?P<num_dice>
            [0-9]+
        )?
    d)?
    (?P<num_sides>
        [0-9\%]+
    )
    (
        (?P<modifier>
            [+-/<]
        )
        (?P<modifier_value>
            [0-9]+
        )
    )?$''',
    re.IGNORECASE | re.VERBOSE)


def do_xdy(rng, dice, sides, modifier=None, modifier_value=None):
    ''' Roll a number of dice with the given number of sides.

    modifier - one of [+-/<]
    modifier_value - an integer
    '''
    result = []
    total = 0
    minimum = sides + 1
    for i in range(dice):
        value = rng.randint(1, sides)

        total += value
        if value < minimum:
            minimum = value

        result.append(value)

    if modifier == '+':
        total += modifier_value
    elif modifier == '-':
        total -= modifier_value
    elif modifier == '/':
        total //= modifier_value
    elif modifier == '<':
        total -= minimum

    if total < 1:
        total = 1

    return (result, total)


def do_simple(rng, simple, output):
    ''' Do a simple roll request, just a number.
    '''
    try:
        sides = int(simple.group('num_sides'))
    except ValueError:
        if simple.group('num_sides') == '%':
            sides = 100
        else:
            output['error'] = True
            output['message'] = 'Roll a what?'
            return

    if sides < 1:
        output['error'] = True
        output['message'] = '{0}: Invalid number of sides.'.format(sides)
        return

    result, total = do_xdy(rng, 1, sides)

    output['rolls'].append(result[0])
    output['sum'] = total


def do_combo(rng, combo):
    ''' Do a combo!

    dnd -> 3d6 3d6 3d6 3d6 3d6 3d6
    dnd+ -> 4d6<1 4d6<1 4d6<1 4d6<1 4d6<1 4d6<1
    open -> d%, roll again and add if >= 95
    '''
    name = combo.group('combo_name')
    flag = combo.group('combo_flag')

    if name.lower() in ['dnd', 'd&d']:
        # Roll up a D&D/Pathfinder character; six stats from 3-18.
        if flag is None:
            # Standard: Roll 3d6 six times.
            combo_result = [do_roll(rng, '3d6') for x in range(6)]
        else:
            # House rule: Roll 4d6, throw out low die.
            combo_result = [do_roll(rng, '4d6<1') for x in range(6)]
    elif name.lower() in ['open']:
        # Roll a Rolemaster "open-ended" d%.
        #
        # TODO: Check to make sure these don't go negative if you roll <= 5;
        # that may have been a house rule.
        if flag is not None:
            output['warning'] = 'Flag ignored for open-ended rolls.'

        combo_result = do_roll(rng, 'd%')
        this_roll = combo_result['rolls'][0]

        while this_roll >= 95:
            this_result = do_roll(rng, 'd%')
            this_roll = this_result['rolls'][0]

            combo_result['rolls'].append(this_roll)
            combo_result['sum'] += this_roll

    return combo_result


def do_normal(rng, roll, output):
    ''' Do a normal roll.
    '''
    try:
        num_dice = int(roll.group('num_dice'))
    except TypeError:
        # roll.group('num_dice') was None because you used dxxx not 1dxxx.
        num_dice = 1

    try:
        num_sides = int(roll.group('num_sides'))
    except ValueError:
        # roll.group('num_sides') didn't convert because you used %.
        num_sides = 100

    modifier = roll.group('modifier')

    if modifier is not None:
        modifier_value = int(roll.group('modifier_value'))
    else:
        modifier_value = None

    if num_sides == 100:
        orig = '{0}d%'.format(num_dice)
    else:
        orig = '{0}d{1}'.format(num_dice, num_sides)
    if modifier is not None:
        orig = '{0}{1}{2}'.format(orig, modifier, modifier_value)

    rolls, total = do_xdy(rng, num_dice, num_sides, modifier, modifier_value)

    output['modifier'] = modifier
    output['modifier_value'] = modifier_value
    output['sum'] = total
    output['rolls'] += rolls


def do_roll(rng, roll):
    ''' Simulate a /roll request.

    Called once per incoming roll in the /roll request.
    '''
    output = {
        'original': roll,  # Original input.
        'error': False,  # Has a fatal error occurred?
        'message': None,  # More info.
        'warning': None,  # Messages from the plugin, not the result.
        'rolls': [],  # Results from each die roll.
        'modifier': None,
        'modifier_value': None,
        'sum': 0  # Result.
    }

    simple = SIMPLE_PATTERN.match(roll)
    if simple is not None:
        do_simple(rng, simple, output)
    else:
        normal = ROLL_PATTERN.match(roll)
        if normal is not None:
            do_normal(rng, normal, output)
        else:
            # You're doing it wrong.
            output['rolls'].append((roll, 'Incomprehensible roll.'))

    if len(output['rolls']) < 1:
        output['error'] = True
        output['message'] = 'That accomplished nothing.'

    return output


def test():
    ''' Try a bunch of combos.
    '''
    simple = ['1', '2', '3', '4', '6', '8', '10', '12', '20', '%', '100', '65536']
    combos = ['dnd', 'd&d', 'dnd+', 'd&d+', 'open']
    rolls = ['3d6', '3d6+1', '3d6-1', '3d6-16', '3d6/2', '3d6/3', '2d6<1']

    print('test_simple:')
    random.seed(0)
    for a_roll in simple:
        payload = do_roll(random, a_roll)
        print('\t{0} = {1}'.format(a_roll, payload))

    print('test_combos:')
    random.seed(0)
    for a_roll in combos:
        combo = COMBO_PATTERN.match(a_roll)
        if combo is not None:
            payload = do_combo(random, combo)
            print('\t{0} = {1}'.format(a_roll, payload))
        else:
            print('\tInvalid combo: {0}', a_roll)

    print('test_rolls:')
    random.seed(0)
    for a_roll in rolls:
        payload = do_roll(random, a_roll)
        print('\t{0} = {1}'.format(a_roll, payload))


def usage():
    ''' Print usage info and exit.
    '''
    print('Usage:')
    print('\t{0} [--help|--test] roll1 [roll2 ... rolln]'.format(sys.argv[0]))
    print('Supports "any" standard notation die roll.')


def main():
    ''' Try to parse the command-line args and produce dice rolls.
    '''
    if len(sys.argv) < 2:
        usage()
        raise SystemExit

    if sys.argv[1] == '--test':
        test()
        raise SystemExit
    elif sys.argv[1] == '--help':
        usage()
        raise SystemExit

    if len(sys.argv[1:]) > 10:
        print('{0} requests; truncated to 10.'.format(len(sys.argv[1:])))

    rng = secrets.SystemRandom()

    for arg in sys.argv[1:11]:
        combo = COMBO_PATTERN.match(arg)
        if combo is not None:
            payload = do_combo(rng, combo)
        else:
            payload = do_roll(rng, arg)

        print(payload)


if __name__ == '__main__':
    main()
