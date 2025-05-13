package logrotation

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {

	// create a temporary directory
	rotatingLogger := createRotatingLogger(t)

	if err := rotatingLogger.Init(); err != nil {
		t.Fatalf("RotatingLogger Init() failed with error: %v", err)
	}

	// verify if logFilesCount number of files were created
	for i := 1; i <= rotatingLogger.logFilesCount; i++ {
		path := rotatingLogger.instanceFileName(i)
		_, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected %q to exist, got error %v", path, err)
		}
	}

}

func TestIfMessageShouldBeAppended(t *testing.T) {

	rotatingLogger := createRotatingLogger(t)

	if err := rotatingLogger.Init(); err != nil {
		t.Fatalf("RotatingLogger Init() failed with error: %v", err)
	}

	logFile := rotatingLogger.instanceFileName(1)

	messages := []struct {
		num                      int
		messageLine              string
		fileSeed                 string
		deduplicateLine          bool
		expectedShouldBeAppended bool
	}{
		{1, "Log Message", "", true, true},
		{2, "Log Message", "Log Message", true, false},
		{3, "Error Message", "Log Message", false, true},
		{4, "Error Message", "Error Message", false, true},
		{5, "Log Message", "Error Message", true, true},
	}

	for _, message := range messages {

		// write the fileSeed to the file
		if err := os.WriteFile(logFile, []byte(message.fileSeed), 0660); err != nil {
			t.Fatalf("error wrtitng %v to logfile: %v, received error: %v", message.fileSeed, logFile, err)
		}

		shouldBeAppended, _ := rotatingLogger.checkIfMessageShouldBeAppended(logFile, message.messageLine, message.deduplicateLine)

		if shouldBeAppended != message.expectedShouldBeAppended {
			t.Fatalf("test:%d failed, expected whether the line should be appended as: %v, but got: %v", message.num, message.expectedShouldBeAppended, shouldBeAppended)
		}
	}

}

func TestLogRotation(t *testing.T) {

	// create a temporary directory
	rotatingLogger := createRotatingLogger(t)

	if err := rotatingLogger.Init(); err != nil {
		t.Fatalf("RotatingLogger Init() failed with error: %v", err)
	}

	// write data in the files
	fileData := make([]string, 0, rotatingLogger.logFilesCount)

	for i := 1; i <= rotatingLogger.logFilesCount; i++ {
		content := []byte(fmt.Sprintf("data-%d", i))
		fileData = append(fileData, string(content))
		logFile := rotatingLogger.instanceFileName(i)
		if err := os.WriteFile(logFile, content, 0660); err != nil {
			t.Fatalf("error wrtitng %v to logfile: %v, received error: %v", content, logFile, err)
		}
	}

	// perform log-rtoation
	if err := rotatingLogger.performLogRotation(); err != nil {
		t.Fatalf("error performing log-rotation: %v", err)
	}

	// the first file should be empty, all other files should have content of one file previous to them
	for i := 1; i <= rotatingLogger.logFilesCount; i++ {
		logFile := rotatingLogger.instanceFileName(i)

		data, err := os.ReadFile(logFile)
		if err != nil {
			t.Errorf("expected %s to exist, got error: %v", logFile, err)
		}

		received := string(data)

		var expected string
		if i == 1 {
			expected = ""
		} else {
			expected = fileData[i-2]
		}

		if received != expected {
			t.Fatalf("expected %v, but receieved %v for file: %v", expected, received, logFile)
		}
	}

}

func createRotatingLogger(t *testing.T) *RotatingLogger {
	dir := t.TempDir()

	return NewRotatingLogger(filepath.Join(dir, "log"), 100, 3)
}
