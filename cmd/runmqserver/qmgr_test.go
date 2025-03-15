/*
© Copyright IBM Corporation 2023

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
	"strings"
	"testing"
)

func Test_validateLogFilePageSetting(t *testing.T) {
	type args struct {
		iniFilePath       string
		isValid           bool
		logFilePagesValue string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestLogFilePages1",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogFilePages_1.ini",
				isValid:           true,
				logFilePagesValue: "1235",
			},
		},
		{
			name: "TestLogFilePages2",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogFilePages_2.ini",
				isValid:           true,
				logFilePagesValue: "2224",
			},
		},
		{
			name: "TestLogFilePages3",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogFilePages_3.ini",
				isValid:           false,
				logFilePagesValue: "1235",
			},
		},
		{
			name: "TestLogFilePages4",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogFilePages_4.ini",
				isValid:           false,
				logFilePagesValue: "1235",
			},
		},
		{
			name: "TestLogFilePages5",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogFilePages_5.ini",
				isValid:           false,
				logFilePagesValue: "1235",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iniFileBytes, err := os.ReadFile(tt.args.iniFilePath)
			if err != nil {
				t.Fatal(err)
			}
			validate := validateLogFilePageSetting(iniFileBytes, tt.args.logFilePagesValue)
			if validate != tt.args.isValid {
				t.Fatalf("Expected ini file validation output to be %v got %v", tt.args.isValid, validate)
			}
		})
	}
}

func Test_validatePrimaryLogFileSetting(t *testing.T) {
	type args struct {
		iniFilePath            string
		isValid                bool
		primaryLogFilesValue   string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestPrimaryLogFile1",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogPrimaryFiles_1.ini",
				isValid:           true,
				primaryLogFilesValue: "3",
			},
		},
		{
			name: "TestPrimaryLogFile2",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogPrimaryFiles_2.ini",
				isValid:           true,
				primaryLogFilesValue: "3",
			},
		},
		{
			name: "TestPrimaryLogFile3",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogPrimaryFiles_3.ini",
				isValid:           false,
				primaryLogFilesValue: "10",
			},
		},
		{
			name: "TestPrimaryLogFile4",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogPrimaryFiles_4.ini",
				isValid:           false,
				primaryLogFilesValue: "3",
			},
		},
		{
			name: "TestPrimaryLogFile5",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogPrimaryFiles_5.ini",
				isValid:           false,
				primaryLogFilesValue: "1235",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iniFileBytes, err := os.ReadFile(tt.args.iniFilePath)
			if err != nil {
				t.Fatal(err)
			}
			validate := validatePrimaryLogFileSetting(iniFileBytes, tt.args.primaryLogFilesValue)
			if validate != tt.args.isValid {
				t.Fatalf("Expected ini file validation output to be %v got %v", tt.args.isValid, validate)
			}
		})
	}
}

func Test_validateSecondaryLogFileSetting(t *testing.T) {
	type args struct {
		iniFilePath            string
		isValid                bool
		secondaryLogFilesValue string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestSecondaryLogFile1",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogSecondaryFiles_1.ini",
				isValid:           true,
				secondaryLogFilesValue: "2",
			},
		},
		{
			name: "TestSecondaryLogFile2",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogSecondaryFiles_2.ini",
				isValid:           true,
				secondaryLogFilesValue: "2",
			},
		},
		{
			name: "TestSecondaryLogFile3",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogSecondaryFiles_3.ini",
				isValid:           false,
				secondaryLogFilesValue: "10",
			},
		},
		{
			name: "TestSecondaryLogFile4",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogSecondaryFiles_4.ini",
				isValid:           false,
				secondaryLogFilesValue: "2",
			},
		},
		{
			name: "TestSecondaryLogFile5",
			args: args{
				iniFilePath:       "./test-files/testvalidateLogSecondaryFiles_5.ini",
				isValid:           false,
				secondaryLogFilesValue: "1235",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iniFileBytes, err := os.ReadFile(tt.args.iniFilePath)
			if err != nil {
				t.Fatal(err)
			}
			validate := validatePrimaryLogFileSetting(iniFileBytes, tt.args.primaryLogFilesValue)
			if validate != tt.args.isValid {
				t.Fatalf("Expected ini file validation output to be %v got %v", tt.args.isValid, validate)
			}
		})
	}
}

// Unit test for special character in queue manager names
func Test_SpecialCharInQMNameReplacements(t *testing.T) {
	type qmNames struct {
		qmName         string
		replacedQMName string
	}

	tests := []qmNames{
		{
			qmName:         "QM.",
			replacedQMName: "QM!",
		}, {
			qmName:         "QM/",
			replacedQMName: "QM&",
		},
		{
			qmName:         "QM.GR.NAME",
			replacedQMName: "QM!GR!NAME",
		}, {
			qmName:         "QM/GR/NAME",
			replacedQMName: "QM&GR&NAME",
		}, {
			qmName:         "QMGRNAME",
			replacedQMName: "QMGRNAME",
		},
	}

	for _, test := range tests {
		replacedQMName := replaceCharsInQMName(test.qmName)
		if !strings.EqualFold(replacedQMName, test.replacedQMName) {
			t.Fatalf("QMName replacement failed. Expected %s but got %s\n", test.replacedQMName, replacedQMName)
		}
	}
}
