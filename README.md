# omarchy-matrix-screensaver

Matrix digital rain as the Omarchy screensaver.

## Install

You need Omarchy with `~/.config/hypr/hyprland.lua`, and Alacritty, Foot, Ghostty or Kitty as the default terminal.

```bash
git clone https://github.com/TheBabaYaga/omarchy-matrix-screensaver
cd omarchy-matrix-screensaver
./install.sh
```

If Go is missing, the script installs it. That step asks for your password. To update, run `git pull` and then `./install.sh`.

## Try it

Start the screensaver now. Press a key to stop it.

```bash
omarchy launch screensaver force
```

## Options

| Flag | Default | Meaning |
| --- | --- | --- |
| `--charset` | `katakana` | Glyphs: `katakana`, `ascii` or `binary` |
| `--color` | `#00ff41` | Main color as `#rrggbb` |
| `--speed` | `1.0` | Speed multiplier, more than 0 |
| `--density` | `0.7` | Part of each column with rain, more than 0, up to 1 |
| `--fps` | `30` | Frames per second, 1 to 240 |

To change a flag, edit the `"$matrix_rain" &` line in `bin/omarchy-screensaver`, then run `./install.sh` again.

```bash
"$matrix_rain" --color '#ff0000' --speed 1.5 &
```

## Uninstall

```bash
cd omarchy-matrix-screensaver
./uninstall.sh
```

You can then delete the `omarchy-matrix-screensaver` folder.

## How it works

- `install.sh` builds `~/.local/bin/matrix-rain` and copies the files to `~/.local/share/matrix-screensaver`.
- It adds one line to `~/.config/hypr/hyprland.lua`. That line loads `~/.local/share/matrix-screensaver/hyprland.lua`, which puts the wrapper folder first in the Hyprland PATH.
- Omarchy runs `omarchy-screensaver` by name, so it finds the wrapper in that folder first.
- The wrapper starts `matrix-rain`. A key press or a focus change stops it.
- `uninstall.sh` removes the line and the files. Omarchy then uses its stock screensaver.

## License

MIT. See [LICENSE](LICENSE).
