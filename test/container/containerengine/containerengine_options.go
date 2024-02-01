package containerengine

import (
	"os"
	"strings"
	"testing"
)

type ContainterClientOption func(*ContainerClient)

func WithTestCommandLogger(t *testing.T) ContainterClientOption {
	return func(cc *ContainerClient) {
		cc.logger = t
		cc.logOptions = logOptions{
			logCommands: strings.ToLower(os.Getenv("TEST_LOG_CONTAINER_COMMANDS")) == "true",
		}
	}
}
