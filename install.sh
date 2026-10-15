#!/bin/bash
# Install or update the Matrix screensaver for Omarchy. Run it again to update.
set -euo pipefail

cd "$(dirname "$0")"

config="$HOME/.config/hypr/hyprland.lua"
share="$HOME/.local/share/matrix-screensaver"
line='do local f = os.getenv("HOME") .. "/.local/share/matrix-screensaver/hyprland.lua"; local h = io.open(f); if h then h:close(); dofile(f) end end -- matrix-screensaver'

if [[ ! -f $config ]]; then
  echo "Cannot find $config. This needs Omarchy with a Lua Hyprland config." >&2
  exit 1
fi

if ! command -v go >/dev/null; then
  echo "Go is not installed. Installing it now."
  omarchy pkg add go
fi

echo "Building matrix-rain..."
mkdir -p "$HOME/.local/bin" "$share/bin"
go build -trimpath -ldflags "-s -w" -o "$HOME/.local/bin/matrix-rain" .
install -m 755 bin/omarchy-screensaver "$share/bin/omarchy-screensaver"
install -m 644 hyprland.lua "$share/hyprland.lua"

if ! grep -qxF "$line" "$config"; then
  # Start on a new line if the file does not end with one.
  if [[ -s $config && -n $(tail -c1 "$config") ]]; then
    printf '\n%s' "$line" >>"$config"
  else
    printf '%s\n' "$line" >>"$config"
  fi
fi

if [[ -n ${HYPRLAND_INSTANCE_SIGNATURE:-} ]] && command -v hyprctl >/dev/null; then
  hyprctl reload >/dev/null
  echo "Done. Test it with: omarchy launch screensaver force"
else
  echo "Done. Log in to Hyprland, then test it with: omarchy launch screensaver force"
fi
