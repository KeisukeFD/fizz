package tictactoe

import (
	"testing"

	"github.com/wI2L/fizz/internal/conformance/repro"
)

// TestReproTicTacToe rebuilds learn.openapis.org examples/v3.1/tictactoe.json.
func TestReproTicTacToe(t *testing.T) {
	repro.Run(t, New(), "v3.1/tictactoe.json", "allow.json")
}
