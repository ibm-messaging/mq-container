/*
© Copyright IBM Corporation 2017, 2018

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

func signalHandler(qmgr string) chan int {
	control := make(chan int)
	// Use separate channels for the signals, to avoid SIGCHLD signals swamping
	// the buffer, and preventing other signals.
	stopSignals := make(chan os.Signal)
	reapSignals := make(chan os.Signal)
	signal.Notify(stopSignals, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for {
			select {
			case sig := <-stopSignals:
				log.Printf("Signal received: %v", sig)
				signal.Stop(reapSignals)
				signal.Stop(stopSignals)
				go metrics.StopMetricsGathering()
				stopQueueManager(qmgr)
				// One final reap
				reapZombies()
				close(control)
				// End the goroutine
				return
			case <-reapSignals:
				log.Debug("Received SIGCHLD signal")
				reapZombies()
			case job := <-control:
				switch {
				case job == startReaping:
					// Add SIGCHLD to the list of signals we're listening to
					log.Debug("Listening for SIGCHLD signals")
					signal.Notify(reapSignals, syscall.SIGCHLD)
				case job == reapNow:
					reapZombies()
				}
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
