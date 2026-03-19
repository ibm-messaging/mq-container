// © Copyright IBM Corporation 2019, 2026
package main

import (
	"io/ioutil"
	"testing"
)

var getRemoveInstanaINIEntryTests = []struct {
	input    string
	output   string
	exitName string
	removed  bool
}{
	{"test1b_input.ini", "test1b_output.ini", "MQInstanaTracingExit", true},
	{"test2b_input.ini", "test2b_output.ini", "MQInstanaTracingExit", true},
	{"test3b_input.ini", "test3b_output.ini", "MQInstanaTracingExit", true},
	{"test4_input.ini", "test4b_output.ini", "MQInstanaTracingExit", false},
	{"test5_input.ini", "test5b_output.ini", "MQInstanaTracingExit", true},
}

// TestRemoveInstanaExitINIEntry runs a series of tests based on input/output
// INI files in the current directory
func TestRemoveInstanaExitINIEntry(t *testing.T) {
	for _, table := range getRemoveInstanaINIEntryTests {
		t.Run(table.input, func(t *testing.T) {
			b, err := ioutil.ReadFile("./test-files/opentracing/" + table.input)
			in := string(b)
			t.Logf("Input INI file: \n%v", in)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			removed, out, err := removeTracingExitINIEntry(table.exitName, in)
			if err != nil {
				t.Errorf("Error removing Instana ini entry: %v", err.Error())
			}
			t.Logf("Output INI file: \n%v", out)
			b2, err := ioutil.ReadFile("./test-files/opentracing/" + table.output)
			if string(b2) != out {
				t.Errorf("Expected contents of %v; got:\n%v", table.output, out)
			}
			if table.removed != removed {
				t.Errorf("Expected 'removed' to be %v; got %v", table.removed, removed)
			}
		})
	}
}
