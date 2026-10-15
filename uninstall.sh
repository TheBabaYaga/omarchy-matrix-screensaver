#!/bin/bash
# Remove the Matrix screensaver. Omarchy uses its stock screensaver again.
set -euo pipefail

config="$HOME/.config/hypr/hyprland.lua"

if [[ -f $config ]] && grep -q ' -- matrix-screensaver$' "$config"; then
  tmp=$(mktemp)
  grep -v ' -- matrix-screensaver$' "$config" >"$tmp" || true
  # install.sh adds a newline before its line if the file had none at the end. Remove it again.
  if [[ -n $(tail -c1 "$config") && -s $tmp ]]; then
    truncate -s -1 "$tmp"
  fi
  cat "$tmp" >"$config"
  rm -f "$tmp"
fi

rm -rf "$HOME/.local/share/matrix-screensaver"
rm -f "$HOME/.local/bin/matrix-rain"

if [[ -n ${HYPRLAND_INSTANCE_SIGNATURE:-} ]] && command -v hyprctl >/dev/null; then
  hyprctl reload >/dev/null
  echo "Done. Omarchy uses its stock screensaver again."
else
  echo "Done. Omarchy uses its stock screensaver after the next Hyprland login."
fi
