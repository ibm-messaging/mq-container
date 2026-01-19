package probesocket

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ibm-messaging/mq-container/pkg/logger"
)

const (
	LivenessProbeSockPath  = "/run/liveness-probe.sock"
	ReadinessProbeSockPath = "/run/readiness-probe.sock"
	StartupProbeSockPath   = "/run/startup-probe.sock"
)

type LogLevel int

const (
	INFO LogLevel = iota
	ERROR
)

type ProbeSocket struct {
	wg      sync.WaitGroup
	lock    sync.Mutex
	logger  *logger.Logger
	lastLog map[string]string
	sockets []string
}

func NewProbeSocket(name, logFormat string) (*ProbeSocket, error) {

	logger, err := logger.NewLogger(os.Stdout, false, (logFormat == "json"), name)
	if err != nil {
		return &ProbeSocket{}, err
	}

	return &ProbeSocket{
		logger:  logger,
		lastLog: make(map[string]string),
		sockets: []string{
			LivenessProbeSockPath,
			ReadinessProbeSockPath,
			StartupProbeSockPath,
		},
	}, nil

}

func (ps *ProbeSocket) Start(ctx context.Context) error {

	for _, path := range ps.sockets {

		// Remove socket file if it already exists
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}

		// initialize the socket conection
		listener, err := ps.initSocket(path)
		if err != nil {
			return err
		}

		ps.wg.Add(1)
		go func(listener *net.UnixListener, socketPath string) {
			defer ps.wg.Done()

			ps.listen(ctx, listener, socketPath)

		}(listener, path)

	}

	return nil

}

func (ps *ProbeSocket) initSocket(socketPath string) (*net.UnixListener, error) {

	unixAddress := &net.UnixAddr{Name: socketPath, Net: "unix"}
	listener, err := net.ListenUnix("unix", unixAddress)
	if err != nil {
		return nil, err
	}

	// remove socket files on connection close
	listener.SetUnlinkOnClose(true)

	// modify socket path permissions
	if err := os.Chmod(socketPath, 0o600); err != nil {
		if listenerCloseError := listener.Close(); listenerCloseError != nil {
			return nil, listenerCloseError
		}
		return nil, err
	}

	return listener, nil

}

func (ps *ProbeSocket) listen(ctx context.Context, listener *net.UnixListener, socketPath string) {

	defer listener.Close()

	// go routine to handle context
	go func() {
		<-ctx.Done()
		if err := listener.Close(); err != nil {
			return
		}
	}()

	// start accepting connections
	for {
		connection, err := listener.Accept()
		if err != nil {
			// if the err is due to context timeout
			if ctx.Err() != nil {
				// exit gracefully
				return
			}
			continue
		}

		ps.handleConnection(connection, socketPath)

	}

}

func (ps *ProbeSocket) handleConnection(connection net.Conn, socketPath string) {

	defer connection.Close()

	sc := bufio.NewScanner(connection)

	for sc.Scan() {

		logLine := sc.Text()

		// default
		logLevel := INFO.getLogLevel()
		logMessage := logLine

		if i := strings.IndexByte(logLine, '\t'); i >= 0 {
			logLevel = strings.ToUpper(strings.TrimSpace(logLine[:i]))
			logMessage = strings.TrimSpace(logLine[i+1:])
		}

		ps.dedupLog(socketPath, logLevel, logMessage)

	}

}

func (ps *ProbeSocket) dedupLog(socketPath, logLevel, logMessage string) {

	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.lastLog[socketPath] == logMessage {
		return
	}

	ps.lastLog[socketPath] = logMessage

	if logLevel == ERROR.getLogLevel() {
		ps.logger.Errorf("%s", logMessage)
	} else {
		ps.logger.Printf("%s", logMessage)
	}

}

func (ps *ProbeSocket) Wait() {
	ps.wg.Wait()
}

func (l LogLevel) getLogLevel() string {
	switch l {
	case ERROR:
		return "ERROR"
	case INFO:
		return "INFO"
	default:
		return "INFO"
	}
}

func SendProbeLogs(logLevel LogLevel, socketPath, logMessage string) {

	if skipLogEnabled() {
		return
	}

	connection, err := net.DialTimeout("unix", socketPath, 300*time.Millisecond)
	if err != nil {
		return
	}
	defer connection.Close()

	if err := connection.SetWriteDeadline(time.Now().Add(300 * time.Millisecond)); err != nil {
		return
	}

	writer := bufio.NewWriter(connection)

	if _, err := writer.WriteString(fmt.Sprintf("%s\t%s", logLevel.getLogLevel(), logMessage)); err != nil {
		return
	}

	if err := writer.Flush(); err != nil {
		return
	}

}

func skipLogEnabled() bool {

	args := os.Args

	if len(args) >= 2 && (slices.Contains(args, "-skip-log") || slices.Contains(args, "--skip-log")) {
		return true
	}

	return false

}
