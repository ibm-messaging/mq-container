/*
© Copyright IBM Corporation 2020

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
package htpasswd

import (
	"testing"
)

// TestCheckUser verifies Htpassword's use
func TestCheckUser(t *testing.T) {
	err := SetPassword("guest", "guestpw", true)
	if err != nil {
		t.Fatalf("htpassword test failed due to error:%s\n", err.Error())
	}
	found, ok, err := AuthenticateUser("guest", "guestpw", true)
	if err != nil {
		t.Fatalf("htpassword test1 failed as user could not be found:%s\n", err.Error())
	}
	if found == false || ok == false {
		t.Fatalf("htpassword test1 failed as user could not be found:%v, ok:%v\n", found, ok)
	}

	found, ok, err = AuthenticateUser("myguest", "guestpw", true)
	if err == nil {
		t.Fatalf("htpassword test2 failed as no error received for non-existing user\n")
	}
	if found == true || ok == true {
		t.Fatalf("htpassword test2 failed for non-existing user found :%v, ok:%v\n", found, ok)
	}

	found, ok, err = AuthenticateUser("guest", "guest", true)
	if err == nil {
		t.Fatalf("htpassword test3 failed as incorrect password of user did not return error\n")
	}

	if found == false || ok == true {
		t.Fatalf("htpassword test3 failed for existing user with incorrect passwored found :%v, ok:%v\n", found, ok)
	}

	found, err = ValidateUser("guest", true)
	if err != nil || found == false {
		t.Fatalf("htpassword test4 failed as user could not be found:%v, ok:%v\n", found, ok)
	}

	found, err = ValidateUser("myguest", true)
	if err != nil || found == true {
		t.Fatalf("htpassword test5 failed as non-existing user returned to be found:%v, ok:%v\n", found, ok)
	}
}
