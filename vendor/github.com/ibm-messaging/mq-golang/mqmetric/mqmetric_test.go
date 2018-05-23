/*
© Copyright IBM Corporation 2018

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
package mqmetric

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ibm-messaging/mq-golang/ibmmq"
)

func TestNormalise(t *testing.T) {
	var expected float64
	test := MonElement{}
	value := int64(1000000)

	test.Datatype = ibmmq.MQIAMO_MONITOR_PERCENT
	expected = 10000
	returned := Normalise(&test, "", value)
	if returned != expected {
		t.Logf("Gave %s, %d. Expected: %f, Got: %f", "ibmmq.MQIAMO_MONITOR_PERCENT", value, expected, returned)
		t.Fail()
	}

	test.Datatype = ibmmq.MQIAMO_MONITOR_HUNDREDTHS
	expected = 10000
	returned = Normalise(&test, "", value)
	if returned != expected {
		t.Logf("Gave %s, %d. Expected: %f, Got: %f", "ibmmq.MQIAMO_MONITOR_HUNDREDTHS", value, expected, returned)
		t.Fail()
	}

	test.Datatype = ibmmq.MQIAMO_MONITOR_MB
	expected = 1048576000000
	returned = Normalise(&test, "", value)
	if returned != expected {
		t.Logf("Gave %s, %d. Expected: %f, Got: %f", "ibmmq.MQIAMO_MONITOR_MB", value, expected, returned)
		t.Fail()
	}

	test.Datatype = ibmmq.MQIAMO_MONITOR_GB
	expected = 1073741824000000
	returned = Normalise(&test, "", value)
	if returned != expected {
		t.Logf("Gave %s, %d. Expected: %f, Got: %f", "ibmmq.MQIAMO_MONITOR_GB", value, expected, returned)
		t.Fail()
	}

	test.Datatype = ibmmq.MQIAMO_MONITOR_MICROSEC
	expected = 1
	returned = Normalise(&test, "", value)
	if returned != expected {
		t.Logf("Gave %s, %d. Expected: %f, Got: %f", "ibmmq.MQIAMO_MONITOR_GB", value, expected, returned)
		t.Fail()
	}
}

func TestReadPatterns(t *testing.T) {
	const filename = "testFile"
	//Create dummy test file
	testData := []byte("test1=yes\ntest2=no\n")
	err := ioutil.WriteFile(filename, testData, 0644)
	if err != nil {
		t.Fatalf("Could not create test file - %v", err)
	}
	defer os.Remove(filename)

	expected := "test1=yes,test2=no"
	back, err := ReadPatterns(filename)
	if err != nil {
		t.Logf("Got error while running ReadPatterns - %v", err)
		t.Fail()
	} else if back != expected {
		t.Logf("File was not parsed correctly. Expected: %s. Got: %s", expected, back)
		t.Fail()
	}
}
func TestFormatDescription(t *testing.T) {
	give := [...]string{"hello", "no space", "no/slash", "no-dash", "single___underscore", "single__underscore", "ALLLOWER", "no_count", "this_bytes_written_switch", "this_bytes_max_switch", "this_bytes_in_use_switch", "this messages_expired_switch", "add_free_space", "suffix_byte", "suffix_message", "suffix_file", "this_percentage_move"}
	expected := [...]string{"hello", "no_space", "no_slash", "no_dash", "single_underscore", "single_underscore", "alllower", "no", "this_written_bytes_switch", "this_max_bytes_switch", "this_in_use_bytes_switch", "this_expired_messages_switch", "add_free_space_percentage", "suffix_bytes", "suffix_messages", "suffix_files", "this_move_percentage"}

	for i, e := range give {
		back := formatDescription(e)
		if back != expected[i] {
			t.Logf("Gave %s. Expected: %s, Got: %s", e, expected[i], back)
			t.Fail()
		}
	}
}
func TestFormatDescriptionElem(t *testing.T) {
	test := MonElement{}
	test.Description = "THIS-should__be/formatted_count"
	expected := "this_should_be_formatted"

	back := formatDescriptionElem(&test)
	if back != expected {
		t.Logf("Gave %s. Expected: %s, Got: %s", test.Description, expected, back)
		t.Fail()
	}

	test.Datatype = ibmmq.MQIAMO_MONITOR_MICROSEC
	expected = "this_should_be_formatted_seconds"
	back = formatDescriptionElem(&test)
	if back != expected {
		t.Logf("Gave %s. Expected: %s, Got: %s", test.Description, expected, back)
		t.Fail()
	}
}
func TestParsePCFResponse(t *testing.T) {
	cfh := ibmmq.NewMQCFH()
	cfh.Type = ibmmq.MQCFT_RESPONSE
	headerbytes := cfh.Bytes()

	params, last := parsePCFResponse(headerbytes)
	if len(params) != 0 && !last {
		t.Logf("Gave just a header. Expected: 0, false , Got: %d, %t", len(params), last)
		t.Fail()
	}

	cfh.ParameterCount = 1
	parm := ibmmq.PCFParameter{
		Type:           ibmmq.MQCFT_STRING, // String
		Parameter:      ibmmq.MQCACF_APPL_NAME,
		String:         []string{"HELLOTEST"},
		ParameterCount: 1,
	}
	headerbytes = cfh.Bytes()
	parmbytes := parm.Bytes()
	messagebytes := append(headerbytes, parmbytes...)

	params, last = parsePCFResponse(messagebytes)
	if len(params) != 1 && !last {
		t.Logf("Gave header and parameter. Expected: 1, false , Got: %d, %t", len(params), last)
		t.Fail()
	} else {
		elem := params[0]
		if elem.Type != parm.Type {
			t.Logf("Returned parameter 'Type' did not match. Expected: %d, Got: %d", parm.Type, elem.Type)
			t.Fail()
		}
		if elem.Parameter != parm.Parameter {
			t.Logf("Returned parameter 'Parameter' did not match. Expected: %d, Got: %d", parm.Parameter, elem.Parameter)
			t.Fail()
		}
		if len(elem.String) != len(parm.String) {
			t.Logf("Length of Returned parameter 'String' did not match. Expected: %d, Got: %d", len(parm.String), len(elem.String))
			t.Fail()
		} else if elem.String[0] != parm.String[0] {
			t.Logf("Returned parameter 'String' did not match. Expected: %s, Got: %s", parm.String[0], elem.String[0])
			t.Fail()
		}
		if len(elem.Int64Value) != 0 {
			t.Logf("Returned parameter 'Int64Value' was not empty, length=%d", len(elem.Int64Value))
			t.Fail()
		}
		if len(elem.GroupList) != 0 {
			t.Logf("Returned parameter 'GroupList' was not empty, length=%d", len(elem.GroupList))
			t.Fail()
		}
	}
}
