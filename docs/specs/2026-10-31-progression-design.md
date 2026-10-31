# Progression, modes, About and releases

Goal: give players a reason to come back every day and to play one more game,
while keeping multiplayer fair. There is no server and no real money.

## Decisions

- The shop sells cosmetics and one convenience item (a streak freeze). Nothing
  bought affects a match, so multiplayer stays fair and `match` needs no economy.
- Coins are only earned. Saves are local JSON, so nothing here pretends to be
  tamper-proof, and the ladder is personal rather than a global leaderboard.
- Old saves keep working: every new field defaults to zero.
- New modes are appended to `match.Mode`, and `ProtocolVersion` goes to 2.

## Modes (`internal/match`)

| Mode | Players | Rules |
|---|---|---|
| Blitz | 1 | 16x16, 2 minutes. Clearing deals a new board and adds time: 20s at first, shrinking by 2s per board to a floor of 2s, so that a run always ends. A mine costs 15s and a 1s stun. Won if at least one board was cleared. |
| Survival | 1 | Expert board, 3 lives. A mine costs a life and stays exposed. Losing the last life ends the run. |
| Last Stand | 2-4 | 16x16 shared board, turn based, private flags. Open safely on your 10s turn. A mine or a timeout puts you out. Last player in wins. If the board is cleared, most tiles among survivors wins; a tie is a draw. |

Bots play Last Stand. `PlayerView` gains `Lives` and `Out`; `View` gains `Boards`.

## Economy (`internal/profile`)

`Record` stays the single place a result becomes rewards.

- **Coins:** from a game's XP, boosted by win streak, plus a random lucky roll
  (x2 or x5) passed in on `GameResult.Roll` so that it stays testable. Also from
  level-ups, awards, quests, promotions and the login bonus.
- **Shop:** bought themes, flag, mine and smiley skins, titles, streak freezes.
- **Quests:** three a day, picked from a pool by hashing the date, with a bonus
  for finishing all three.
- **Login bonus:** a 7-day cycle ending in a big reward. A missed day restarts it.
- **Streak freeze:** covers one missed daily challenge.
- **Ladder:** rank points move a player through Bronze to Master, with three
  divisions per tier. Points are lost on a defeat, but never below the current
  tier's floor. Reaching a tier for the first time pays coins.

## UI (`internal/ui`)

New scenes: Quests, Ladder, Shop (replaces Themes), About. The menu card shows
coins, rank and quests. Results show coins, rank movement and finished quests.
The login bonus is a popup on the menu. The Solo screen lists Blitz and Survival.

## Resize

The bars at non-4:3 sizes are drawn black by `DrawFinalScreen`; they become the
theme's desktop colour. The black flicker during a drag on XWayland is measured
but its cause is not established, so this work does not claim to fix it.

## Releases

- `internal/ui.Version` defaults to `dev`, and is set at link time.
- `scripts/build.sh` reads `VERSION` and writes it into the binaries and the
  macOS `Info.plist`. `make dist` passes `git describe`.
- CI runs vet and tests on every push and pull request. Pushing a `v*` tag
  tests, builds every platform and publishes a GitHub release.

## Music

An original 16-bar chiptune loop in A minor, synthesised at start-up from
`ui/music.go`: a pulse wave lead, triangle wave bass and noise drums. It follows
the volume setting at a lower level than the sound effects, and Settings can
turn it off.

## Testing

Rules and economy are tested in `match` and `profile`. Catalog entries are tested
against the art tables, so a skin can't silently fall back. New screens are
checked by driving the real UI with `-script` and viewing screenshots.
