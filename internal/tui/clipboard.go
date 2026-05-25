package tui

import "github.com/atotto/clipboard"

// copyToClipboard writes s to the system clipboard.
// Indirected through a package variable so tests can swap it out
// without depending on the host clipboard backend (pbcopy / xclip / ...).
var copyToClipboard = func(s string) error {
	return clipboard.WriteAll(s)
}
