# Anti-piracy launcher screen

POBLATION v1.0.0-beta.1 keeps this screen aesthetic only. It never deletes saves, locks the player out, or changes the game binary.

Implemented triggers:

- `AP_RANDOM`: exactly `rng.Intn(1000) == 0`, so 0.1% per launch.
- `AP_OFFLINE`: shown when the launcher uses a locally installed version because GitHub is unavailable.
- `AP_CLOCK`: shown after the friendly clock anomaly dialog continues.
- `AP_RAM_EATER`: shown only when multiple intentional clock-anomaly signals are detected after the first benefit-of-the-doubt occurrence.

All messages can include real save data from `~/.poblation/saves/`: world name, day, population, last played, and the selected save description.
