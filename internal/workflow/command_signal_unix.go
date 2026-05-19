//go:build !windows

package workflow

import (
	"os"
	"syscall"
)

func commandStageSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
