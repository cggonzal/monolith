/*
Package server_management provides helpers for systemd socket activation and
deployment utilities used by the monolith.
*/
package server_management

import (
	"context"
	"embed"
	"errors"
	"log"
	"log/slog"
	"monolith/config"
	"monolith/routes"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func RunServer(staticFiles embed.FS) {
	// Grab the listener from systemd (fall back to a normal port if run
	// without socket activation — handy for local dev).
	listeners, err := sdListeners()
	var ln net.Listener
	if err == nil && len(listeners) > 0 {
		ln = listeners[0]
		log.Printf("using systemd listener on %s", ln.Addr())
	} else {
		ln, err = net.Listen("tcp", "127.0.0.1:"+config.PORT)
		if err != nil {
			log.Fatalf("listen: %v", err)
		}
		log.Printf("socket activation unavailable, listening on %s", ln.Addr())
	}

	slog.Info("Starting server", "address", ":"+config.PORT)

	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		Handler:           routes.InitServerHandler(staticFiles),
	}

	// Tell systemd we’re ready **before** we start accepting traffic.
	go sdNotifyReady()

	// Graceful shutdown on SIGTERM/SIGINT.
	idleConnsClosed := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("HTTP shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("serving HTTP")
	if err := server.Serve(ln); err != http.ErrServerClosed {
		log.Fatalf("Serve: %v", err)
	}

	<-idleConnsClosed
	log.Printf("goodbye")
}

// sdListeners returns the list of sockets passed by systemd via LISTEN_FDS.
//
// See <https://www.freedesktop.org/software/systemd/man/sd_listen_fds.html>.
func sdListeners() ([]net.Listener, error) {
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
func sdNotifyReady() {
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
