/*
© Copyright IBM Corporation 2017, 2024

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
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ibm-messaging/mq-container/internal/metrics"
	"golang.org/x/sys/unix"
)

const (
	startReaping = iota
	reapNow      = iota
)

func signalHandler(qmgr string, startupCtx context.Context) chan int {
	control := make(chan int)
	// Use separate channels for the signals, to avoid SIGCHLD signals swamping
	// the buffer, and preventing other signals.
	stopSignals := make(chan os.Signal, 1)
	reapSignals := make(chan os.Signal, 1)
	signal.Notify(stopSignals, syscall.SIGTERM, syscall.SIGINT)

	// Pulling out as function as reused for shutdown and standard control flow
	processControlSignal := func(job int) {
		switch {
		case job == startReaping:
			// Add SIGCHLD to the list of signals we're listening to
			log.Debug("Listening for SIGCHLD signals")
			signal.Notify(reapSignals, syscall.SIGCHLD)
		case job == reapNow:
			reapZombies()
		}
	}

	// Start handling signals
	go func() {
		shutdownCtx, shutdownComplete := context.WithCancel(context.Background())
		defer func() {
			shutdownComplete()
		}()
		stopTriggered := false
		for {
			select {
			case sig := <-stopSignals:
				if stopTriggered {
					continue
				}
				log.Printf("Signal received: %v", sig)
				signal.Stop(stopSignals)
				stopTriggered = true

				// If a stop signal is received during the startup process continue processing control signals until the main thread marks startup as complete
				// Don't close the control channel until the main thread has been allowed to finish spawning processes and marks startup as complete
				// Continue to process job control signals to avoid a deadlock
				done := false
				for !done {
					select {
					// When the main thread has cancelled the startup context due to completion or an error stop processing control signals
					case <-startupCtx.Done():
						done = true
					// Keep processing control signals until the main thread has finished its startup
					case job := <-control:
						processControlSignal(job)
					}
				}
				metrics.StopMetricsGathering(log)

				// Shutdown queue manager in separate goroutine to allow reaping to continue in parallel
				go func() {
					_ = stopQueueManager(qmgr)
					shutdownComplete()
				}()
			case <-shutdownCtx.Done():
				signal.Stop(reapSignals)

				// One final reap
				// This occurs after all startup processes have been spawned
				reapZombies()

				close(control)
				// End the goroutine
				return
			case <-reapSignals:
				log.Debug("Received SIGCHLD signal")
				reapZombies()
			case job := <-control:
				processControlSignal(job)
			}
		}
	}()
	return control
}

// reapZombies reaps any zombie (terminated) processes now.
// This function should be called before exiting.
func reapZombies() {
	for {
		var ws unix.WaitStatus
		pid, err := unix.Wait4(-1, &ws, unix.WNOHANG, nil)
		// If err or pid indicate "no child processes"
		if pid == 0 || err == unix.ECHILD {
			return
		}
		log.Debugf("Reaped PID %v", pid)
	}
}
