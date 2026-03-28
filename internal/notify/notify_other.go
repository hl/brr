//go:build !darwin && !linux

package notify

func send(_, _ string) error {
	return nil // no-op on platforms without a built-in notification mechanism
}
