/*
© Copyright IBM Corporation 2026

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package probes

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"sync"
	"time"

	"github.com/ibm-messaging/mq-container/pkg/logger"
)

var (
	LivenessProbeSockPath = "/run/liveness-probe.sock"
	StartupProbeSockPath  = "/run/startup-probe.sock"
)

type LogLevel int

const (
	INFO LogLevel = iota
	ERROR
)

type ProbeType string

const (
	LivenessProbe ProbeType = "LIVENESS"
	StartupProbe  ProbeType = "STARTUP"
)

type ProbeLoggingSocket struct {
	wg                sync.WaitGroup
	lock              sync.Mutex
	logger            *logger.Logger
	lastLog           map[string]string
	sockets           []string
	probeLoggingState *ProbeLoggingState
}

// NewProbeLoggingSocket creates a ProbeLoggingSocket instance.
//   - initializes probe socket paths, logger, and probe logging state
//   - maintains last logged messages for probe log deduplication
func NewProbeLoggingSocket(name, logFormat string, probeState *ProbeLoggingState, log *logger.Logger) *ProbeLoggingSocket {

	return &ProbeLoggingSocket{
		logger:  log,
		lastLog: make(map[string]string),
		sockets: []string{
			LivenessProbeSockPath,
			StartupProbeSockPath,
		},
		probeLoggingState: probeState,
	}
}

// Start initializes Unix domain sockets and begins listening for probe events.
func (ps *ProbeLoggingSocket) Start(ctx context.Context) error {
	for _, path := range ps.sockets {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}

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

// initSocket creates and configures a Unix domain socket listener.
func (ps *ProbeLoggingSocket) initSocket(socketPath string) (*net.UnixListener, error) {
	unixAddress := &net.UnixAddr{Name: socketPath, Net: "unix"}
	listener, err := net.ListenUnix("unix", unixAddress)
	if err != nil {
		return nil, err
	}

	listener.SetUnlinkOnClose(true)

	if err := os.Chmod(socketPath, 0o600); err != nil {
		if listenerCloseError := listener.Close(); listenerCloseError != nil {
			return nil, listenerCloseError
		}
		return nil, err
	}

	return listener, nil
}

// listen accepts incoming connections on the socket.
func (ps *ProbeLoggingSocket) listen(ctx context.Context, listener *net.UnixListener, socketPath string) {
	defer listener.Close()

	go func() {
		<-ctx.Done()
		if err := listener.Close(); err != nil {
			return
		}
	}()

	for {
		connection, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}

		ps.handleConnection(connection, socketPath)
	}
}

// handleConnection processes incoming probe execution messages.
func (ps *ProbeLoggingSocket) handleConnection(connection net.Conn, socketPath string) {
	defer connection.Close()

	sc := bufio.NewScanner(connection)

	for sc.Scan() {
		logLine := sc.Text()

		var probeLoggingExecution *ProbeLoggingExecution
		if err := json.Unmarshal([]byte(logLine), &probeLoggingExecution); err != nil {
			return
		}

		if probeLoggingExecution.ProbeType == LivenessProbe {
			ps.handleLivenessProbe(probeLoggingExecution, socketPath)
		} else if probeLoggingExecution.ProbeType == StartupProbe {
			ps.handleStartupProbe(probeLoggingExecution, socketPath)
		}
	}
}

// handleLivenessProbe updates liveness probe state based on incoming events.
//   - INCOMPLETE events shift history and start a new current run
//   - PASSED/FAILED events complete the current run
func (ps *ProbeLoggingSocket) handleLivenessProbe(probeLoggingExecution *ProbeLoggingExecution, socketPath string) {
	if probeLoggingExecution == nil {
		return
	}

	state := ps.probeLoggingState.LivenessProbeLoggingState
	if state == nil {
		return
	}

	switch probeLoggingExecution.Status {
	case ProbeIncomplete:
		if state.CurrentRun != nil {

			// If the existing current run is still incomplete, capture its elapsed duration before moving it to history.
			if state.CurrentRun.Status == ProbeIncomplete && state.CurrentRun.StartTime != nil && state.CurrentRun.Duration == nil {
				duration := time.Since(*state.CurrentRun.StartTime).Milliseconds()
				state.CurrentRun.Duration = &duration
			}

			state.PreviousRuns[1] = state.PreviousRuns[0]
			state.PreviousRuns[0] = *state.CurrentRun
		}

		state.CurrentRun = probeLoggingExecution

	case ProbeFailed, ProbePassed:
		if state.CurrentRun != nil {
			state.CurrentRun.EndTime = probeLoggingExecution.EndTime
			state.CurrentRun.Status = probeLoggingExecution.Status
			state.CurrentRun.Duration = probeLoggingExecution.Duration
			state.CurrentRun.LogMessage = probeLoggingExecution.LogMessage
			state.CurrentRun.LogLevel = probeLoggingExecution.LogLevel
		} else {
			state.CurrentRun = probeLoggingExecution
		}

		ps.dedupLog(socketPath, state.CurrentRun.LogLevel.getLogLevel(), state.CurrentRun.LogMessage, state.CurrentRun.ProbeType.getProbeType())
	}
}

// handleStartupProbe updates startup probe state based on incoming events.
//   - INCOMPLETE events increment the startup probe attempt count
//   - PASSED/FAILED events finalize the current startup probe state
func (ps *ProbeLoggingSocket) handleStartupProbe(probeLoggingExecution *ProbeLoggingExecution, socketPath string) {
	if probeLoggingExecution == nil {
		return
	}

	state := ps.probeLoggingState.StartupProbeLoggingState
	if state == nil {
		return
	}

	switch probeLoggingExecution.Status {
	case ProbeIncomplete:
		attemptCount := 1
		if state.CurrentRun != nil {
			attemptCount = state.CurrentRun.AttemptCount + 1
		}
		probeLoggingExecution.AttemptCount = attemptCount
		state.CurrentRun = probeLoggingExecution
	case ProbeFailed, ProbePassed:
		if state.CurrentRun != nil {
			probeLoggingExecution.AttemptCount = state.CurrentRun.AttemptCount
		}
		state.CurrentRun = probeLoggingExecution

		ps.dedupLog(socketPath, state.CurrentRun.LogLevel.getLogLevel(), state.CurrentRun.LogMessage, state.CurrentRun.ProbeType.getProbeType())
	}
}

// dedupLog emits probe logs with probe-specific INFO-level deduplication.
//   - liveness probes suppress repeated INFO messages and always emit ERROR messages
//   - startup probes suppress repeated INFO messages and suppress all ERROR messages during runtime
func (ps *ProbeLoggingSocket) dedupLog(socketPath, logLevel, logMessage string, probeType string) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if probeType == StartupProbe.getProbeType() {
		ps.startupProbeDedupLogHelper(logLevel, socketPath, logMessage)
	} else if probeType == LivenessProbe.getProbeType() {
		ps.livenessProbeDedupLogHelper(logLevel, socketPath, logMessage)
	}
}

// startupProbeDedupLogHelper emits startup probe logs with INFO-level deduplication.
//   - suppresses repeated INFO messages
//   - suppresses startup probe ERROR logs during runtime
func (ps *ProbeLoggingSocket) startupProbeDedupLogHelper(logLevel, socketPath, logMessage string) {

	// we will not be logging startup probe errors
	if logLevel == ERROR.getLogLevel() {
		return
	}

	if ps.lastLog[socketPath] == logMessage {
		return
	}

	ps.lastLog[socketPath] = logMessage

	ps.logger.Printf("%s", logMessage)
}

// livenessProbeDedupLogHelper emits liveness probe logs with INFO-level deduplication.
//   - suppresses repeated INFO messages
//   - always emits ERROR messages
func (ps *ProbeLoggingSocket) livenessProbeDedupLogHelper(logLevel, socketPath, logMessage string) {

	if logLevel == INFO.getLogLevel() && ps.lastLog[socketPath] == logMessage {
		return
	}

	ps.lastLog[socketPath] = logMessage

	if logLevel == ERROR.getLogLevel() {
		ps.logger.Errorf("%s", logMessage)
	} else {
		ps.logger.Printf("%s", logMessage)
	}
}

// Wait blocks until all socket listener goroutines have completed
func (ps *ProbeLoggingSocket) Wait() {
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

func (p ProbeType) getProbeType() string {
	switch p {
	case LivenessProbe:
		return "LIVENESS"
	case StartupProbe:
		return "STARTUP"
	}
	return ""
}

// SendProbeLoggingExecution sends probe execution data over the Unix socket.
func SendProbeLoggingExecution(probeLoggingExecution *ProbeLoggingExecution, socketPath string) {
	connection, err := net.DialTimeout("unix", socketPath, 300*time.Millisecond)
	if err != nil {
		return
	}
	defer connection.Close()

	if err := connection.SetWriteDeadline(time.Now().Add(300 * time.Millisecond)); err != nil {
		return
	}

	writer := bufio.NewWriter(connection)

	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(probeLoggingExecution); err != nil {
		return
	}

	if err := writer.Flush(); err != nil {
		return
	}
}
