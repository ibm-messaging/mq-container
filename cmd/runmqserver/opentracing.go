// © Copyright IBM Corporation 2019, 2026
package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/copy"
	"github.com/ibm-messaging/mq-container/internal/pathutils"
	"github.com/ibm-messaging/mq-container/pkg/mqini"
)

// enableInstanaTracingFiles copies all required Instana Tracing files to their target locations
func enableInstanaTracingFiles() error {

	// Copy userconfiginstana.ini file to ephemeral volume
	err := copy.CopyFile("/opt/MQInstanaTracing/userconfiginstana.ini", "/run/MQInstanaTracing/userconfiginstana.ini")
	if err != nil {
		return err
	}

	return nil
}

// disableInstanaTracing disables Instana Tracing by removing contents of userconfiginstana.ini file
func disableInstanaTracing() error {
	// #nosec G306
	err := ioutil.WriteFile("/run/MQInstanaTracing/userconfiginstana.ini", []byte(""), 0660)
	if err != nil {
		return err
	}

	return nil
}

// removeTracingExitINIEntry removes any stanza containing the specified exit e.g "MQInstanaTracingExit" from an INI string
// Returns a bool to indicate whether the source was changed in the output.  If false, the output is the
// same as the input.
func removeTracingExitINIEntry(exitName string, ini string) (bool, string, error) {
	scanner := bufio.NewScanner(strings.NewReader(ini))
	scanner.Split(bufio.ScanLines)
	var out strings.Builder
	var stanza strings.Builder
	insideTracingExit := false
	foundTracingExit := false
	// Read through the INI file
	for scanner.Scan() {
		t := scanner.Text()
		if strings.Contains(t, ":") {
			// Found a new stanza
			if !insideTracingExit {
				_, err := out.WriteString(stanza.String())
				if err != nil {
					return false, "", err
				}
			}
			insideTracingExit = false
			stanza.Reset()
		} else if strings.Contains(t, exitName) {
			// This stanza includes the tracing exit
			insideTracingExit = true
			foundTracingExit = true
		}
		// On a line inside a stanza.
		// Save it for later, in case this is the tracing exit.
		_, err := fmt.Fprintf(&stanza, "%v\n", t)
		if err != nil {
			return false, "", err
		}
	}
	// Write the last stanza
	if !insideTracingExit {
		_, err := out.WriteString(stanza.String())
		if err != nil {
			return false, "", err
		}
	}
	if foundTracingExit {
		return true, out.String(), nil
	} else {
		return false, ini, nil
	}
}

// removeTracingExit removes the API exit from the qm.ini file, if it's
// previously been enabled
func removeTracingExit(exitName string, name string) error {
	qm, err := mqini.GetQueueManager(name)
	if err != nil {
		// Return nil here, because the queue manager may never have been created
		return nil
	}
	qmData := mqini.GetDataDirectory(qm)
	qmIniFile := pathutils.CleanPath(qmData, "qm.ini")
	// #nosec G304 - qmData filepath is derived from dspmqinf
	b, err := ioutil.ReadFile(qmIniFile)
	if err != nil {
		return err
	}
	removed, newIni, err := removeTracingExitINIEntry(exitName, string(b))
	if err != nil {
		return err
	}
	if removed {
		// #nosec G306 - its a read by owner/s group, and pose no harm.
		err = ioutil.WriteFile(qmIniFile, []byte(newIni), 0660)
		if err != nil {
			return err
		}
	}
	return nil
}
