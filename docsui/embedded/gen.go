// Package embedded serves the docs UIs from assets embedded in the binary.
//
// The assets are pinned copies of the files referenced by the docsui URL
// constants. Run go generate in this directory to verify them against the
// docsui Integrity constants (network access required); to move a pin, edit
// the URL constant and run "go run ../internal/pinassets -update".
package embedded

//go:generate go run ../internal/pinassets
