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

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ibm-messaging/mq-container/internal/metadata"
	"github.com/ibm-messaging/mq-container/pkg/logger"
	"github.com/ibm-messaging/mq-container/pkg/name"
)

func main() {
	var log *logger.Logger
	ctx, cancelFn := context.WithCancelCause(context.Background())

	go monitorSigterm(ctx, cancelFn)

	name, _ := name.GetQueueManagerName()
	var err error
	log, err = logger.NewLogger(os.Stdout, getDebug(), true, name)
	if err != nil {
		fmt.Printf("failed to create logger: %s\n", err.Error())
		os.Exit(1)
	}

	svr := NewServer(metadata.SocketPath(), log)
	err = svr.run(ctx)
	switch {
	case err == nil:
		// Do nothing
	case errors.Is(err, errSigtermReceived):
		log.Println("Server shutdown due to signal from host")
	default:
		log.Fatalf("Error running server: %s", err)
	}
}

func monitorSigterm(ctx context.Context, cancelFn context.CancelCauseFunc) {
	sigint := make(chan os.Signal, 1)

	signal.Notify(sigint, os.Interrupt)
	signal.Notify(sigint, syscall.SIGTERM)

	select {
	case <-sigint:
		cancelFn(errSigtermReceived)
	case <-ctx.Done():
	}
}

type serverLogger interface {
	Println(args ...interface{})
	Printf(format string, args ...interface{})
}

func getDebug() bool {
	debug := os.Getenv("DEBUG")
	if debug == "true" || debug == "1" {
		return true
	}
	return false
}

var errSigtermReceived = errors.New("sigterm received")
