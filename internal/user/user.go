/*
© Copyright IBM Corporation 2018, 2020

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
package user

import (
	"golang.org/x/sys/unix"
)

// User holds information on primary and supplemental OS groups
type User struct {
	UID             int
	PrimaryGID      int
	SupplementalGID []int
}

// GetUser returns the current user and group information
func GetUser() (User, error) {
	u := User{
		UID:        unix.Geteuid(),
		PrimaryGID: unix.Getgid(),
	}
	groups, err := unix.Getgroups()
	if err != nil {
		return u, err
	}
	u.SupplementalGID = groups
	return u, nil
}
