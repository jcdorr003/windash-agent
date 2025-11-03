package metrics

import (
	"fmt"

	"github.com/denisbrodbeck/machineid"
)

// GetHostID returns a stable unique identifier for this machine
// Uses machine ID which persists across reboots
func GetHostID() (string, error) {
	id, err := machineid.ProtectedID("windash-agent")
	if err != nil {
		return "", fmt.Errorf("failed to get machine ID: %w", err)
	}
	return id, nil
}
