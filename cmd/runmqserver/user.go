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
	"fmt"
	"os/user"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
)

const groupName string = "supplgrp"

func logUser() {
	u, usererr := user.Current()
	if usererr == nil {
		g, err := getCurrentUserGroups()
		if err != nil && len(g) == 0 {
			log.Printf("Running as user ID %v (%v) with primary group %v", u.Uid, u.Name, u.Gid)
		} else {
			// Look for the primary group in the list of group IDs
			for i, v := range g {
				if v == u.Gid {
					// Remove the element from the slice
					g = append(g[:i], g[i+1:]...)
				}
			}
			log.Printf("Running as user ID %v (%v) with primary group %v, and supplementary groups %v", u.Uid, u.Name, u.Gid, strings.Join(g, ","))
		}
	}

	if usererr == nil && u.Username != "mqm" {
		mqm, err := user.Lookup("mqm")
		// Need to print out mqm user details as well.
		g, err := getUserGroups(mqm)
		if err != nil && len(g) == 0 {
			log.Printf("MQM user ID %v (%v) has primary group %v", mqm.Uid, "mqm", mqm.Gid)
		} else {
			// Look for the primary group in the list of group IDs
			for i, v := range g {
				if v == mqm.Gid {
					// Remove the element from the slice
					g = append(g[:i], g[i+1:]...)
				}
			}
			log.Printf("MQM user ID %v (%v) has primary group %v, and supplementary groups %v", mqm.Uid, "mqm", mqm.Gid, strings.Join(g, ","))
		}
	}
}

func getCurrentUserGroups() ([]string, error) {
	var nilArray []string
	out, _, err := command.Run("id", "--groups")
	if err != nil {
		log.Debug("Unable to get current user groups")
		return nilArray, err
	}

	out = strings.TrimSpace(out)
	if out == "" {
		// we don't have any groups?
		return nilArray, fmt.Errorf("Unable to determine groups for current user")
	}

	groups := strings.Split(out, " ")
	return groups, nil
}

func getUserGroups(usr *user.User) ([]string, error) {
	var nilArray []string
	out, _, err := command.Run("id", "--groups", usr.Uid)
	if err != nil {
		log.Debugf("Unable to get user %s groups", usr.Uid)
		return nilArray, err
	}

	out = strings.TrimSpace(out)
	if out == "" {
		// we don't have any groups?
		return nilArray, fmt.Errorf("Unable to determine groups for user %s", usr.Uid)
	}

	groups := strings.Split(out, " ")
	return groups, nil
}
