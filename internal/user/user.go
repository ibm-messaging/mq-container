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
package user

import (
	"fmt"
	"os/user"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
)

// User holds information on primary and supplemental OS groups
type User struct {
	UID             string
	Name            string
	PrimaryGID      string
	SupplementalGID []string
}

// GetUser returns the current user and group information
func GetUser() (User, error) {
	u, err := user.Current()
	if err != nil {
		return User{}, err
	}
	g, err := getCurrentUserGroups()
	if err != nil {
		return User{}, err
	}
	if err != nil && len(g) == 0 {
		return User{
			UID:             u.Uid,
			Name:            u.Name,
			PrimaryGID:      u.Gid,
			SupplementalGID: []string{},
		}, nil
	}
	// Look for the primary group in the list of group IDs
	for i, v := range g {
		if v == u.Gid {
			// Remove the element from the slice
			g = append(g[:i], g[i+1:]...)
		}
	}
	return User{
		UID:             u.Uid,
		Name:            u.Name,
		PrimaryGID:      u.Gid,
		SupplementalGID: g,
	}, nil
}

func getCurrentUserGroups() ([]string, error) {
	var nilArray []string
	out, _, err := command.Run("id", "--groups")
	if err != nil {
		return nilArray, err
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return nilArray, fmt.Errorf("Unable to determine groups for current user")
	}

	groups := strings.Split(out, " ")
	return groups, nil
}
