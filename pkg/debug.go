package fctl

import (
	"context"
	"fmt"
	"io"
)

func IfDebug(ctx context.Context, fn func()) {
	if IsDebugFromContext(ctx) {
		fn()
	}
}

func DebugLn(ctx context.Context, w io.Writer, a ...any) {
	IfDebug(ctx, func() {
		fmt.Fprintln(w, a...)
	})
}
