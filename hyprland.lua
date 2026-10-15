-- matrix-screensaver: install.sh copies this file to ~/.local/share/matrix-screensaver/hyprland.lua.
-- Omarchy puts its own bin folder first in PATH. Put the wrapper folder before it.
-- Remove both folders first, so a reload does not add copies.
local matrix_dir = os.getenv("HOME") .. "/.local/share/matrix-screensaver/bin"
local omarchy_bin = (os.getenv("OMARCHY_PATH") or "/usr/share/omarchy") .. "/bin"
local matrix_path = { matrix_dir, omarchy_bin }
for entry in (os.getenv("PATH") or "/usr/local/bin:/usr/bin"):gmatch("[^:]+") do
  if entry ~= matrix_dir and entry ~= omarchy_bin then table.insert(matrix_path, entry) end
end
hl.env("PATH", table.concat(matrix_path, ":"))
