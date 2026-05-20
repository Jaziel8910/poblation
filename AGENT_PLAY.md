# POBLATION Agent Play

This is the no-excuses way for an agent to open the real game UI, move through the main player views, advance time, and save screenshots.

Run from `poblation/`:

```powershell
go run . --seed 420 --agent-play --agent-steps 14 --agent-out ..\agent-runs\manual-check
```

What it does:

- renders the actual Bubbletea UI model, not a fake mock;
- captures menu, map, map after time passes, mind, settlement, world, pobles, events, and newspaper;
- writes each screen as `.txt` and `.png`;
- writes `summary.md` with the run order.

The output folder is ignored by Git on purpose. Use it as visual proof before release work, launcher packaging, or UI refactors.

Recommended release gate:

```powershell
go test ./...
go build ./...
go run . --seed 420 --agent-play --agent-steps 14 --agent-out ..\agent-runs\release-check
```

Then open the PNGs and verify that no core player view is blank, stuck on loading, or showing a placeholder.
