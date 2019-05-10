/*
© Copyright IBM Corporation 2018, 2019

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
func postInit(name string) error {
	enableWebServer := os.Getenv("MQ_ENABLE_EMBEDDED_WEB_SERVER")
	if enableWebServer == "true" || enableWebServer == "1" {

		// Configure Single-Sign-On for the web server (if enabled)
		enableSSO := os.Getenv("MQ_BETA_ENABLE_SSO")
		if enableSSO == "true" || enableSSO == "1" {
			err := configureSSO()
			if err != nil {
				return err
			}
		}

		// Configure the web server (if enabled)
		err := configureWebServer()
		if err != nil {
			return err
		}
		// Start the web server, in the background (if installed)
		// WARNING: No error handling or health checking available for the web server
		go func() {
			startWebServer()
		}()
	}
	return nil
}
