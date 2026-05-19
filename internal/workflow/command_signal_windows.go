//go:build windows

package workflow

import "os"

func commandStageSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}
