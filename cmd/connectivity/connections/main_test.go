package connections

import (
	"os"
	"testing"

	"github.com/pterm/pterm"
)

// TestMain disables colours so assertions on plain output do not depend on
// the caller's NO_COLOR or terminal settings.
func TestMain(m *testing.M) {
	pterm.DisableColor()
	os.Exit(m.Run())
}
