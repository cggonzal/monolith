package server_management

import (
	"errors"
	"net"
	"os"
	"strconv"
)

// sdListeners returns the list of sockets passed by systemd via LISTEN_FDS.
//
// See <https://www.freedesktop.org/software/systemd/man/sd_listen_fds.html>.
func SdListeners() ([]net.Listener, error) {
	const fdStart = 3 // SD_LISTEN_FDS_START

	pidStr := os.Getenv("LISTEN_PID")
	fdStr := os.Getenv("LISTEN_FDS")
	if pidStr == "" || fdStr == "" {
		return nil, errors.New("no systemd sockets found")
	}
	if pid, _ := strconv.Atoi(pidStr); pid != os.Getpid() {
		return nil, errors.New("LISTEN_PID mismatch")
	}
	n, err := strconv.Atoi(fdStr)
	if err != nil || n == 0 {
		return nil, errors.New("LISTEN_FDS invalid or zero")
	}

	// Clear the env vars so they don't leak to child processes.
	_ = os.Unsetenv("LISTEN_PID")
	_ = os.Unsetenv("LISTEN_FDS")

	ls := make([]net.Listener, 0, n)
	for fd := fdStart; fd < fdStart+n; fd++ {
		file := os.NewFile(uintptr(fd), "listener")
		ln, err := net.FileListener(file)
		if err != nil {
			return nil, err
		}
		ls = append(ls, ln)
	}
	return ls, nil
}

// sdNotifyReady sends "READY=1" to systemd (no‑op if NOTIFY_SOCKET is unset).
func SdNotifyReady() {
	socket := os.Getenv("NOTIFY_SOCKET")
	if socket == "" {
		return
	}
	// Abstract‑namespace sockets start with '@' which translates to a
	// leading NUL byte when dialing.
	if socket[0] == '@' {
		socket = "\x00" + socket[1:]
	}
	addr := &net.UnixAddr{Name: socket, Net: "unixgram"}
	conn, err := net.DialUnix("unixgram", nil, addr)
	if err != nil {
		return // silently ignore; we’re just being polite to PID 1
	}
	_, _ = conn.Write([]byte("READY=1"))
	_ = conn.Close()
}
