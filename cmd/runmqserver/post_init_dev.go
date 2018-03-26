// +build mqdev

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
package main

import (
	"os"
)

// postInit is run after /var/mqm is set up
// This version of postInit is only included as part of the MQ Advanced for Developers build
func postInit(name string) error {
	disable := os.Getenv("MQ_DISABLE_WEB_CONSOLE")
	if disable != "true" && disable != "1" {
		// Configure and start the web server, in the background (if installed)
		// WARNING: No error handling or health checking available for the web server,
		// which is why it's limited to use with MQ Advanced for Developers only
		go func() {
			configureWebServer()
			startWebServer()
		}()
	}
	return nil
}
