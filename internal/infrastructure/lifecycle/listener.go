package lifecycle

import (
	"fmt"
	"net"
)

// PreBindListeners synchronously binds TCP listeners for both the public
// and management servers before any background workers or goroutines launch.
//
// If the management listener fails to bind, the public listener is rolled back
// immediately (closed) to prevent port or resource leaking.
func PreBindListeners(cfg *Config) (pubLn net.Listener, mgmtLn net.Listener, err error) {
	pubAddr := cfg.PublicAddress()
	pubLn, err = net.Listen("tcp", pubAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("bind public listener %s: %w", pubAddr, err)
	}

	mgmtAddr := cfg.ManagementAddress()
	mgmtLn, err = net.Listen("tcp", mgmtAddr)
	if err != nil {
		// Rollback public listener immediately
		_ = pubLn.Close()
		return nil, nil, fmt.Errorf("bind management listener %s: %w", mgmtAddr, err)
	}

	return pubLn, mgmtLn, nil
}
