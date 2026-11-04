# Mine Party 95

Retro Minesweeper with direct online multiplayer for you and your friends.
There is no account, no server and no subscription: one player's game hosts
the match, and everything else lives on your own machine.

## Run it

Download the build for your system from the
[releases page](https://github.com/dozheiny/mineparty95/releases), or build them
all into `dist/` with `make dist`.

- **Windows:** run `mineparty-windows-amd64.exe`. The build is unsigned, so
  SmartScreen may warn you: choose *More info*, then *Run anyway*. Windows
  Firewall will ask to allow the game the first time you host; allow it.
- **Linux:** `chmod +x mineparty-linux-amd64`, then run it. It needs X11 or
  XWayland and OpenGL, which desktop systems already have.
- **macOS:** unzip the app, then in Terminal run
  `xattr -cr MineParty95-macos-arm64.app` and open it. The build isn't notarized
  by Apple, so without that step macOS blocks it; alternatively use *System
  Settings > Privacy & Security > Open Anyway*. Pick `arm64` for Apple Silicon
  and `amd64` for Intel Macs.

From source: `go run ./cmd/mineparty` (Go 1.26 or newer; Linux also needs the
X11 and OpenGL development headers).

## Modes

| Mode | Players | What happens |
|---|---|---|
| Solo | 1 | Beginner, Intermediate or Expert, with no-guess boards by default. |
| Daily Challenge | 1 | The same board for everyone each day. Keep your streak alive and share your time. |
| Blitz | 1 | Clear as many boards as you can before the clock runs out. Each clear buys a little less time; a mine costs 15 seconds. |
| Survival | 1 | One Expert board and three lives. |
| Race | 2-4 | Everyone gets the same board. First to clear wins; a mine stuns you for 5 seconds. |
| Mine Flags | 2 | Turn-based duel on one board. Finding a mine scores and keeps your turn. If you're behind, you get one 5x5 bomb. |
| Co-op | 2-4 | One board, live cursors. One mine ends the run for the team. |
| Territory | 2-4 | A huge shared board, in real time. Tiles you open are yours; mines stun you and cost tiles. |
| Last Stand | 2-4 | Take turns opening tiles on one board. Hit a mine or run out your 10 seconds and you're out. |

Every multiplayer mode can also be played against CPU opponents under
*Practice vs CPU*, or used to fill empty seats in a room.

## Progress

Every game earns XP and coins, win or lose, and a lucky game pays double or
five times as much.

- **Levels** pay coins and unlock themes.
- **The ladder** runs from Bronze to Master. Wins climb it, and losses slip back,
  but never below the tier you've reached. Reaching a tier pays a big bonus.
- **Quests** change every day, with a bonus for finishing all three.
- **The login bonus** builds over seven days. Missing a day starts it again.
- **The shop** sells themes, flag, mine and smiley skins, titles and streak
  freezes. A streak freeze covers one missed daily. Everything else is
  cosmetic, so nothing you buy helps you in a match.

Coins can only be earned; there is nothing to pay for.

## Controls

| Input | Action |
|---|---|
| Left click | Open a tile, or chord on a number |
| Right click, or Ctrl+click | Flag, then mark `?`, then clear. Chording leaves `?` tiles shut |
| Space or F2 | New game or rematch |
| P or middle click | Ping a tile in multiplayer |
| B | Arm your bomb in Mine Flags |
| F11 | Fullscreen |
| Esc | Leave a solo or practice game |

## Playing with friends

**Same network:** one player chooses *Host Game*, and the room appears for
everyone else under *Join Game*.

**Over the internet:** the host's game asks the router to open a port using
UPnP, then shows an *Internet code*. Friends type that into *Join Game*.

If the lobby says automatic setup failed:

1. Forward the TCP port shown in the lobby on your router to your computer,
   then type your public IP into the lobby to get an Internet code.
2. Install [Tailscale](https://tailscale.com) or ZeroTier on everyone's
   machine. The lobby then offers a *VPN* code that behaves like a LAN game.
   This is also the only option if your ISP uses carrier-grade NAT, where
   port forwarding can't work at all.

To host without touching your router at all, start the game with `-lan`.

Things worth knowing:

- A room code contains the host's IP address, so only share it with friends.
- The host's game is the referee. If the host quits, the match ends.
- Clients are only ever sent tiles that have been opened, so they can't read
  mine positions from network traffic. The host's own machine does hold the
  layout, which is an acceptable risk among friends.

## Your save

Progress is stored in `mineparty/profile.json` under your user config folder
(`~/.config` on Linux, `%AppData%` on Windows, `~/Library/Application Support`
on macOS). It holds your XP, coins, rank, quests, records, rival scores and
unlocks.

## Code layout

| Package | Purpose |
|---|---|
| `internal/engine` | Boards, reveal and chord logic, the no-guess solver |
| `internal/match` | Rules for every mode as a pure state machine, plus the CPU player |
| `internal/netplay` | Hosting, joining, room codes, LAN discovery, UPnP |
| `internal/profile` | The local save: XP, streaks, achievements, unlocks |
| `internal/ui` | Screens, generated art and sound |
| `cmd/mineparty` | The executable |

Run `make` to list the build, test and lint commands.

## Releases

Pushing a version tag builds every platform and publishes a GitHub release:

```sh
git tag v1.0.0
git push origin v1.0.0
```

A tag with a suffix, like `v1.0.0-rc1`, is published as a prerelease. The version
shows on the About page and from `mineparty -version`.
