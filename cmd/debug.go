package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/viper"
)

func ifDebug(fn func()) {
	if viper.GetBool(debugFlag) {
		fn()
	}
}

func debugln(w io.Writer, a ...any) {
	ifDebug(func() {
		fmt.Fprintln(w, a...)
	})
}
