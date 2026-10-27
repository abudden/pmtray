// Package single ensures only one instance of the application runs. Later
// instances ask the running one to show its popup, then exit.
package single

import (
	"bufio"
	"errors"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Listen claims the instance socket in dir. If another instance already
// holds it, that instance is asked to show itself and ok is false.
// Otherwise onShow is called (from another goroutine) whenever a later
// instance starts.
func Listen(dir string, onShow func()) (ok bool, err error) {
	path := filepath.Join(dir, "pmtray.sock")
	l, err := net.Listen("unix", path)
	if err != nil {
		conn, derr := net.DialTimeout("unix", path, time.Second)
		if derr == nil {
			conn.Write([]byte("show\n"))
			conn.Close()
			return false, nil
		}
		// Stale socket left by a previous instance that didn't exit cleanly.
		if rerr := os.Remove(path); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
			return true, err
		}
		if l, err = net.Listen("unix", path); err != nil {
			return true, err // run anyway, without single instance support
		}
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				c.SetReadDeadline(time.Now().Add(2 * time.Second))
				line, _ := bufio.NewReader(c).ReadString('\n')
				if line == "show\n" {
					onShow()
				}
			}(conn)
		}
	}()
	return true, nil
}
