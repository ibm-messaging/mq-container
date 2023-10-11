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
