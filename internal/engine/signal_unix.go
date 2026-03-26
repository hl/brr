//go:build !windows

package engine

import (
	"os"
	"os/signal"
	"syscall"
)

func notifySignals(ch chan<- os.Signal) {
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
}

const sigINT = syscall.SIGINT
const sigKILL = syscall.SIGKILL
const sigTERM = syscall.SIGTERM
