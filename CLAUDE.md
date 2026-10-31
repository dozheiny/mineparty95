# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Mine Party 95 is a retro Minesweeper with peer-hosted multiplayer, written in Go with Ebitengine. There is no server: one player's game is the referee. See `README.md` for modes, controls and player-facing networking.

## Commands

The `Makefile` wraps these; `make` lists its targets, and `make test RUN=TestName` runs one test.

```sh
go run ./cmd/mineparty                         # flags: -profile <path> -silent -script <path> -lan
go test -race ./internal/...                   # all tests; netplay takes ~30s (real sockets)
go test -race -run TestName ./internal/match/  # one test
go vet ./... && GOOS=windows go vet ./...      # ui has platform-specific files (hide_windows.go / hide_other.go)
scripts/build.sh                               # wipes dist/, then cross-builds all platforms with CGO_ENABLED=0
```

There is no golangci-lint config. A default `golangci-lint run` reports existing errcheck/staticcheck issues, so treat it as advisory rather than a gate.

`-version` prints the build's version. `internal/ui.Version` is set at link time from `VERSION`, which `make` takes from `git describe`. `scripts/build.sh` writes to `dist/`, or to `OUT` if set, so `OUT=/tmp/x make dist` tests it without replacing existing builds.

Pushing a `v*` tag runs `.github/workflows/release.yml`, which tests, builds every platform and publishes a GitHub release. `ci.yml` covers every push and pull request.

### Driving the real UI

`internal/ui` has almost no unit tests. To exercise a screen, script the real game loop; this needs a display. Always pass `-silent` and a throwaway `-profile`, otherwise the run can write to the real save.

```sh
go run ./cmd/mineparty -silent -profile /tmp/p.json -script flow.txt
```

Script verbs, one per line (`#` for comments): `wait N` (frames), `move X Y`, `click X Y`, `rclick X Y`, `key Enter`, `type text`, `autoplay` (a CPU plays the local seat), `shot out.png`, `quit`. Coordinates are in the 640x480 virtual screen, and screenshots are that size. A script that doesn't end in `quit` leaves the window open.

## Architecture

Dependencies run one way: `engine` <- `match` <- `netplay` <- `ui`. `profile` imports only `match`. Below `ui`, nothing imports Ebitengine, so the rules and networking test headlessly.

**`match` is a pure state machine.** `Apply(now, Action)` goes in, `View(player, now)` comes out. The clock is a `time.Duration` passed in, so a match never reads the wall clock or touches the network or screen. Illegal actions are silently ignored. Tests pin boards with `Config.Layout` and drive time by passing `now` directly.

**`View` is both the security boundary and the game-state protocol.** It is built only from what that player may see, so clients can't read mines from traffic. Never add hidden-mine data to `View` or `PlayerView`. `Bot` decides from a `View` alone, under the same limits as a person; it doesn't peek at the board.

**Every game runs through `netplay.Host`, including solo.** Solo and Practice vs CPU use `HostOptions{Offline: true}`. Screens play through `netplay.Session`, which is either a `Host` or a `Client`; only host controls (starting, configuring, adding bots, kicking) use `*Host` directly. A new rule belongs in `match`, not in `ui` or `netplay`.

**`Host` state belongs to one goroutine.** Anything touching seats or the match runs on `run()`, reached through `do` (queue) or `call` (queue and wait), so the rules need no locks. `Room` and `View` are published to other goroutines through atomic pointers.

**Seats are positional.** Match players are indexed by seat, so a player who leaves mid-match keeps the seat, marked `gone`, until the next start. Removing a seat between matches also drops the finished match. For remote actions the seat comes from the connection and overrides anything in the message; `Host.Send` always forces seat 0.

**Views are snapshots and can be skipped.** A peer's writer sends only the newest `Room` and `View`, so a slow connection drops frames. The UI must work out what happened by comparing state, never by assuming it saw every step.

**The wire format** (`wire.go`) is a single `packet` struct with exactly one field set, sent as length-prefixed JSON over one long-lived flate stream per connection. Hosts accept frames up to 4 KiB and clients up to 1 MiB. `ProtocolVersion` must match on both ends, so bump it for incompatible changes to `packet`, `Room` or `View`.

**Board fields vary by mode.** Classic and Race give each player their own `engine.Field` on the same board; other modes share one. Territory and Last Stand share the field but keep flags per player, which is why `Field.ChordWith` exists. Classic generates its board on the first click unless `Config.Start` or `Config.Layout` is set. Blitz replaces its board mid-match, so a board's tiles can go from open back to hidden within one round.

**Ask a mode what it is rather than naming it.** Use `Mode.Solo()`, `TurnBased()` and `FixedBoard()`, not comparisons with `Classic` or `Flags`. A new mode also needs an entry in `ModeNames`, and multiplayer modes need `lobbyModes` and `modeBlurbs`.

**Results are recorded by the UI.** `ui/results.go` calls `Profile.Record` once per round, using `Game.lastSession` and `lastRound` to avoid counting a result twice.

**`Profile.Record` is the only place a result becomes rewards.** XP, coins, rank points, quests, promotions and awards are all paid there, into one `Rewards` for the results panel. A replayed daily or a game with no personal contribution on a shared board pays only a token, so keep new rewards inside that guard or they become farmable. `profile` never reads the clock or a random source: dates and `GameResult.Roll` are passed in, which keeps it testable.

**The shop sells nothing that affects a match.** Equipped skins are local to `ui` and never cross the network, so `match` stays free of the economy. Saves are plain JSON, so nothing here is tamper-proof and the ladder is personal. New `Profile` maps must also go in `fillMaps`, or saves from older versions load with them nil.

**The Daily Challenge depends on deterministic generation.** Each machine builds the day's board locally from `dailySeed(date)` in `ui/menu.go`. Changing that function, or how `engine` turns a seed into a board, gives players on different builds different dailies. Seeds must stay non-zero, because `Host.begin` replaces a zero seed with a random one.

**There are no asset files.** Art (`ui/art.go`) is drawn from code per theme, and sound (`ui/audio.go`) and the music loop (`ui/music.go`) are synthesised at start-up, so changing either means editing code. All drawing targets the 640x480 canvas, which is full: the menu's twelve buttons and the Stats screen have no room left, so check new lines with a screenshot. Every shop item needs matching art, which `content_test.go` enforces, as a missing skin would otherwise fall back silently.
