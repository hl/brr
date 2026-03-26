//go:build windows

package engine

import (
	"os"
	"os/signal"
	"syscall"
)

func notifySignals(ch chan<- os.Signal) {
	signal.Notify(ch, os.Interrupt)
}

const sigINT = syscall.SIGINT
const sigKILL = syscall.SIGKILL

// sigTERM is not delivered on Windows but is needed for compilation.
const sigTERM = syscall.Signal(0xf)
