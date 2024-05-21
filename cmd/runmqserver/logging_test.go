/*
© Copyright IBM Corporation 2020, 2024

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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

var formatBasicTests = []struct {
	in          []byte
	outContains string
}{
	{
		[]byte("{\"ibm_datetime\":\"2020/06/24 00:00:00\",\"message\":\"Hello world\"}"),
		"Hello",
	},
	{
		[]byte("{\"ibm_datetime\":\"2020/06/24 00:00:00\",\"message\":\"Hello world\", \"ibm_commentInsert1\":\"foo\"}"),
		"CommentInsert1(foo)",
	},
	{
		[]byte("{\"ibm_datetime\":\"2020/06/24 00:00:00\",\"message\":\"Hello world\", \"ibm_arithInsert1\":1}"),
		"ArithInsert1(1)",
	},
}

func TestFormatBasic(t *testing.T) {
	for i, table := range formatBasicTests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			var inObj map[string]interface{}
			json.Unmarshal(table.in, &inObj)
			t.Logf("Unmarshalled: %+v", inObj)
			out := formatBasic(inObj)
			if !strings.Contains(out, table.outContains) {
				t.Errorf("formatBasic() with input=%v - expected output to contain %v, got %v", string(table.in), table.outContains, out)
			}
		})
	}
}

// This test covers for functions isLogConsoleSourceValid() & checkLogSourceForMirroring()
var mqLogSourcesTests = []struct {
	testNum     int
	logsrc      string
	exptValid   bool
	exptQmgrSrc bool
	exptWebSrc  bool
	exptMqscSrc bool
}{
	{1, "qmgr,web", true, true, true, false},
	{2, "qmgr", true, true, false, false},
	{3, "web,qmgr", true, true, true, false},
	{4, "web", true, false, true, false},
	{5, " ", true, true, true, false},
	{6, "QMGR,WEB", true, true, true, false},
	{7, "qmgr,     ", true, true, false, false},
	{8, "qmgr   ,    web", true, true, true, false},
	{9, "qmgr,dummy", false, true, false, false},
	{10, "fake,dummy", false, false, false, false},
	{11, "qmgr,fake,dummy", false, true, false, false},
	{12, "fake,dummy,web", false, false, true, false},
	{13, "true", false, false, false, false},
	{14, "false", false, false, false, false},
	{15, "", true, true, true, false},
	{16, "mqsc", true, false, false, true},
	{17, "MQSC", true, false, false, true},
	{18, "qmgr,mqsc", true, true, false, true},
	{19, "web,mqsc", true, false, true, true},
	{20, "qmgr,web,mqsc", true, true, true, true},
}

func TestLoggingConsoleSourceInputs(t *testing.T) {
	for _, mqlogsrctest := range mqLogSourcesTests {
		err := os.Setenv("MQ_LOGGING_CONSOLE_SOURCE", mqlogsrctest.logsrc)
		if err != nil {
			t.Error(err)
		}
		isValid := isLogConsoleSourceValid()
		if isValid != mqlogsrctest.exptValid {
			t.Errorf("Expected return value from isLogConsoleSourceValid() is %v for MQ_LOGGING_CONSOLE_SOURCE='%v', got %v\n", mqlogsrctest.exptValid, mqlogsrctest.logsrc, isValid)
		}
		isLogSrcQmgr := checkLogSourceForMirroring("qmgr")
		if isLogSrcQmgr != mqlogsrctest.exptQmgrSrc {
			t.Errorf("Expected return value from checkLogSourceForMirroring() is %v for MQ_LOGGING_CONSOLE_SOURCE='%v', got %v\n", mqlogsrctest.exptQmgrSrc, mqlogsrctest.logsrc, isLogSrcQmgr)
		}
		isLogSrcWeb := checkLogSourceForMirroring("web")
		if isLogSrcWeb != mqlogsrctest.exptWebSrc {
			t.Errorf("Expected return value from checkLogSourceForMirroring() is %v for MQ_LOGGING_CONSOLE_SOURCE='%v', got %v\n", mqlogsrctest.exptWebSrc, mqlogsrctest.logsrc, isLogSrcWeb)
		}
		isLogSrcMqsc := checkLogSourceForMirroring("mqsc")
		if isLogSrcMqsc != mqlogsrctest.exptMqscSrc {
			t.Errorf("Expected return value from checkLogSourceForMirroring() is %v for MQ_LOGGING_CONSOLE_SOURCE='%v', got %v\n", mqlogsrctest.exptMqscSrc, mqlogsrctest.logsrc, isLogSrcMqsc)
		}
	}
}

// This test covers for function isExcludedMsgIdPresent()
var mqExcludeIDTests = []struct {
	testNum        int
	exculdeIDsArr  []string
	expectedRetVal bool
	logEntry       string
}{
	{
		1,
		[]string{"AMQ5051I", "AMQ5037I", "AMQ5975I"},
		true,
		"{\"ibm_messageId\":\"AMQ5051I\",\"ibm_arithInsert1\":0,\"ibm_arithInsert2\":1,\"message\":\"AMQ5051I: The queue manager task 'AUTOCONFIG' has started.\"}",
	},
	{
		2,
		[]string{"AMQ5975I", "AMQ5037I"},
		false,
		"{\"ibm_messageId\":\"AMQ5051I\",\"ibm_arithInsert1\":0,\"ibm_arithInsert2\":1,\"message\":\"AMQ5051I: The queue manager task 'AUTOCONFIG' has started.\"}",
	},
	{
		3,
		[]string{""},
		false,
		"{\"ibm_messageId\":\"AMQ5051I\",\"ibm_arithInsert1\":0,\"ibm_arithInsert2\":1,\"message\":\"AMQ5051I: The queue manager task 'AUTOCONFIG' has started.\"}",
	},
}

func TestIsExcludedMsgIDPresent(t *testing.T) {
	for _, excludeIDTest := range mqExcludeIDTests {
		retVal := isExcludedMsgIdPresent(excludeIDTest.logEntry, excludeIDTest.exculdeIDsArr)
		if retVal != excludeIDTest.expectedRetVal {
			t.Errorf("%v. Expected return value from isExcludedMsgIdPresent() is %v for MQ_LOGGING_CONSOLE_EXCLUDE_ID='%v', got %v\n",
				excludeIDTest.testNum, excludeIDTest.expectedRetVal, excludeIDTest.exculdeIDsArr, retVal)
		}
	}
}
