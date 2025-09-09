package syncwriter

import (
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	syncWriters = map[io.Writer]*SyncWriter{}

	loggerMutex            = sync.Mutex{}
	sharedStdoutStderrLock = sync.Mutex{}
)

type SyncWriter struct {
	writeLock *sync.Mutex
	output    io.Writer
}

func (s *SyncWriter) Write(p []byte) (n int, err error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	bytesWritten := 0
	for len(p) > 0 {
		n, err := s.output.Write(p)
		bytesWritten += n
		if err != nil {
			return bytesWritten, err
		}
		p = p[n:]
	}
	return bytesWritten, nil
}

func (s *SyncWriter) Print(a ...any)                 { fmt.Fprint(s, a...) }
func (s *SyncWriter) Println(a ...any)               { fmt.Fprintln(s, a...) }
func (s *SyncWriter) Printf(format string, a ...any) { fmt.Fprintf(s, format, a...) }

// For returns a SyncWriter for the given underlying writer.
//
// A separate SyncWriter will be created for each underlying writer but multiple calls supplying the same writer will return the same SyncWriter.
//
// Note: as a special case, stdout and stderr share a write lock to prevent race conditions where these streams are converged in container logs
func For(w io.Writer) *SyncWriter {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	if writer, found := syncWriters[w]; found {
		return writer
	}
	var lock *sync.Mutex
	if w == os.Stdout || w == os.Stderr {
		lock = &sharedStdoutStderrLock
	} else {
		lock = &sync.Mutex{}
	}
	writer := &SyncWriter{
		output:    w,
		writeLock: lock,
	}
	syncWriters[w] = writer
	return writer
}
