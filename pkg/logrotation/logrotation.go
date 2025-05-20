/*
© Copyright IBM Corporation 2025

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

// Package logrotation contains code to manage the logrotation and append-log logic for logs
package logrotation

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type RotatingLogger struct {
	baseDirectory  string
	filenameFormat string
	maxFileBytes   int
	logFilesCount  int
	lock           sync.Mutex
}

// NewRotatingLogger create a new RotatingLogger, it expects three input parameters,
// basePath is the log-file path prefix,
// filenameFormat is the format string used for each instance of the log file - it should include a `%d` (or variant such as `%02d`) to indicate the log file instance
// maxFileBytes is the max allowed log-file size in bytes,
// logFilesCount is the number of log files required to be created.
func NewRotatingLogger(baseDirectory string, filenameFormat string, maxFileBytes int, logFilesCount int) *RotatingLogger {
	return &RotatingLogger{
		baseDirectory:  baseDirectory,
		filenameFormat: filenameFormat,
		maxFileBytes:   maxFileBytes,
		logFilesCount:  logFilesCount,
	}
}

// instanceFileName returns a log instance filename
func (r *RotatingLogger) instanceFileName(instance int) string {
	filename := fmt.Sprintf(r.filenameFormat, instance)
	return filepath.Join(r.baseDirectory, filename)
}

// Init creates log files
func (r *RotatingLogger) Init() error {
	for i := 1; i <= r.logFilesCount; i++ {
		err := os.WriteFile(r.instanceFileName(i), []byte(""), 0660)
		if err != nil {
			return err
		}
	}

	return nil

}

// Append appends the message line to the logFile and if the logFile size exceeds the maxFileSize then perform log-rotation.
// messageLine is the log message that we need to append to the log-file,
// if deduplicateLine is false the messageLine will always be appended,
// if deduplicateLine is true messageLine will only be appended if it is different from the last line in the logfile.
func (r *RotatingLogger) Append(messageLine string, deduplicateLine bool) {
	r.lock.Lock()
	defer r.lock.Unlock()

	// Ensure message is terminated with a single line feed
	messageLine = strings.TrimSpace(messageLine) + "\n"

	// we will always log in the first instance of the log files
	logFilePath := r.instanceFileName(1)

	// open the log file in append mode
	// for the gosec rule Id: G302 - Expect file permissions to be 0600 or less
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		return
	}

	defer func(f *os.File) {
		if err := logFile.Close(); err != nil {
			fmt.Printf("Error: %v, Failed to close log file: %s\n", err, logFilePath)
		}
	}(logFile)

	// check if the message should be appended to the file
	shouldBeAppended, err := r.checkIfMessageShouldBeAppended(logFilePath, messageLine, deduplicateLine)
	if err != nil {
		fmt.Printf("Failed to validate the currentLog and the lastLog line %v\n", err)
	}

	if !shouldBeAppended {
		return
	}

	// check if the logFileSize has exceeded the maxFileSize then perform the logrotation
	logFileSizeExceeded, err := r.checkIfLogFileSizeExceeded(len(messageLine), logFile)
	if err != nil {
		fmt.Printf("Failed to validate log file size: %v\n", err)
		return
	}

	if logFileSizeExceeded {

		// close the current log file
		err = logFile.Close()
		if err != nil {
			fmt.Printf("Error: %v, Failed to close log file: %v\n", err, logFile.Name())
		}

		// perform log rotation
		err = r.performLogRotation()
		if err != nil {
			fmt.Printf("Failed to perform log-rotation: %v\n", err)
		}

		// open the newly created logFile
		logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Printf("Failed to open log file: %v\n", err)
			return
		}

		defer func(f *os.File) {
			if err := logFile.Close(); err != nil {
				fmt.Printf("Error: %v, Failed to close log file: %s\n", err, logFilePath)
			}
		}(logFile)
	}

	// append the message to the file
	_, err = logFile.WriteString(messageLine)
	if err != nil {
		fmt.Printf("Failed to write to log file: %v\n", err)
	}

}

func (r *RotatingLogger) performLogRotation() error {

	// delete the last log file
	lastLogFile := r.instanceFileName(r.logFilesCount)
	if err := os.Remove(lastLogFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error deleting '%s': %w", lastLogFile, err)
	}

	// rename the remaining instances of the log-files
	for i := r.logFilesCount; i >= 2; i-- {
		oldLogFileInstance := r.instanceFileName(i - 1)
		newLogFileInstance := r.instanceFileName(i)

		if err := os.Rename(oldLogFileInstance, newLogFileInstance); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error moving '%s' to '%s': %w", oldLogFileInstance, newLogFileInstance, err)
		}
	}

	// create the first log-file
	firstInstancePath := r.instanceFileName(1)
	if err := os.WriteFile(firstInstancePath, []byte(""), 0660); err != nil {
		return fmt.Errorf("error creating log file ('%s'): %w", firstInstancePath, err)
	}

	return nil

}

func (r *RotatingLogger) checkIfLogFileSizeExceeded(messageLineLength int, logFile *os.File) (bool, error) {

	fileStat, err := logFile.Stat()
	if err != nil {
		return false, err
	}

	return ((fileStat.Size() + int64(messageLineLength)) >= int64(r.maxFileBytes)), nil

}

func (r *RotatingLogger) checkIfMessageShouldBeAppended(logFilePath, currentLogLine string, deduplicateLine bool) (bool, error) {

	if !deduplicateLine {
		return true, nil
	}

	lastLogLine, err := r.getLogLastLine(logFilePath)

	if err != nil {
		return false, err
	}

	cleanedCurrentLogLine := strings.ReplaceAll(strings.TrimSpace(currentLogLine), "\n", " ")
	cleanedLastLogLine := strings.ReplaceAll(strings.TrimSpace(lastLogLine), "\n", " ")

	if cleanedCurrentLogLine != cleanedLastLogLine {
		return true, nil
	} else {
		return false, nil
	}

}

func (r *RotatingLogger) getLogLastLine(logFilePath string) (string, error) {

	logFile, err := os.Open(logFilePath)

	if err != nil {
		return "", err
	}

	defer func() {
		if err := logFile.Close(); err != nil {
			fmt.Printf("error closing logfile: %s", logFilePath)
		}
	}()

	lineCharsBackwards := []byte{}
	char := make([]byte, 1)

	for pos, err := logFile.Seek(0, io.SeekEnd); pos > 0; pos, err = logFile.Seek(-1, io.SeekCurrent) {

		if err != nil {
			return "", fmt.Errorf("seek failed: %w", err)
		}

		n, err := logFile.ReadAt(char, pos-1)
		if err != nil {
			return "", fmt.Errorf("read failed: %w", err)
		}

		if n < 1 {
			return "", fmt.Errorf("unexpectedly read 0 bytes")
		}

		if char[0] == '\n' || char[0] == '\r' {
			if len(lineCharsBackwards) > 0 {
				break
			}
			continue
		}

		lineCharsBackwards = append(lineCharsBackwards, char...)

	}

	// reverse the slice in place
	slices.Reverse(lineCharsBackwards)

	return string(lineCharsBackwards), nil

}
