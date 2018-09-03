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
	"fmt"
	"os/user"
	"strings"

	"github.com/ibm-messaging/mq-container/internal/command"
)

const groupName string = "supplgrp"

func verifyCurrentUser() error {
	log.Debug("Verifying current user information")
	curUser, err := user.Current()
	if err != nil {
		return err
	}
	log.Debugf("Detected current user as: %v+", curUser)
	if curUser.Username == "mqm" {
		// Not supported yet
		return fmt.Errorf("Container is running as mqm user which is not supported. Please run this container as root")
	} else if curUser.Username == "root" {
		// We're running as root so need to check for supplementary groups.
		// We can't use the golang User.GroupIDs as it doesn't seem to detect container supplementary groups..
		groups, err := getCurrentGroups(curUser)
		for _, e := range groups {
			_, _, testGroup := command.Run("getent", "group", e)
			if testGroup != nil {
				log.Printf("Group %s does not exist on the system... Adding to system and MQM user", e)
				_, _, err = command.Run("groupadd", "-g", e, groupName)
				if err != nil {
					log.Errorf("Failed to create group %s as %s", e, groupName)
					return err
				}
				_, _, err = command.Run("usermod", "-aG", groupName, "mqm")
				if err != nil {
					log.Errorf("Failed to add group %s(%s) to the mqm user.", groupName, e)
					return err
				}
			}
		}
	} else {
		// We're running as an unknown user...
		return fmt.Errorf("Container is running as %s user which is not supported. Please run this container as root", curUser.Username)
	}

	return nil
}

func logUser() {
	u, err := user.Current()
	if err == nil {
		g, err := getCurrentGroups(u)
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

	if err != nil && u.Username != "mqm" {
		mqm, err := user.Lookup("mqm")
		// Need to print out mqm user details as well.
		g, err := getCurrentGroups(mqm)
		if err != nil && len(g) == 0 {
			log.Printf("MQM user ID %v (%v) has primary group %v", mqm.Uid, mqm.Name, mqm.Gid)
		} else {
			// Look for the primary group in the list of group IDs
			for i, v := range g {
				if v == mqm.Gid {
					// Remove the element from the slice
					g = append(g[:i], g[i+1:]...)
				}
			}
			log.Printf("MQM user ID %v (%v) has primary group %v, and supplementary groups %v", mqm.Uid, mqm.Name, mqm.Gid, strings.Join(g, ","))
		}
	}
}

func getCurrentGroups(usr *user.User) ([]string, error) {
	var nilArray []string
	out, _, err := command.Run("id", "--groups", usr.Name)
	if err != nil {
		log.Debugf("Unable to get user %s groups", usr.Name)
		return nilArray, err
	}

	out = strings.TrimSpace(out)
	if out == "" {
		// we don't have any groups?
		return nilArray, fmt.Errorf("Unable to determine groups for user %s", usr.Name)
	}

	groups := strings.Split(out, " ")
	return groups, nil
}
