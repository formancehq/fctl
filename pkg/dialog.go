package fctl

import (
	"fmt"

	"github.com/pterm/pterm"
)

type Dialog interface {
	Info(msg string, args ...any)
}

type ptermDialog struct{}

func (p ptermDialog) Info(msg string, args ...any) {
	pterm.DefaultLogger.Info(fmt.Sprintf(msg, args...))
}

func NewPTermDialog() Dialog {
	return &ptermDialog{}
}
