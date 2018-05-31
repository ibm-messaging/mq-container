// +build linux

/*
© Copyright IBM Corporation 2017, 2018

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

	"golang.org/x/sys/unix"
)

// fsTypes contains file system identifier codes.
// This code will not compile on some operating systems - Linux only.
var fsTypes = map[int64]string{
	0x61756673: "aufs",
	0xef53:     "ext",
	0x6969:     "nfs",
	0x65735546: "fuse",
	0x9123683e: "btrfs",
	0x01021994: "tmpfs",
	0x794c7630: "overlayfs",
}

func checkFS(path string) error {
	statfs := &unix.Statfs_t{}
	err := unix.Statfs(path, statfs)
	if err != nil {
		log.Println(err)
		return nil
	}
	// Use a type conversion to make type an int64.  On s390x it's a uint32.
	t := fsTypes[int64(statfs.Type)]
	switch t {
	case "aufs", "overlayfs", "tmpfs":
		return fmt.Errorf("%v uses unsupported filesystem type: %v", path, t)
	default:
		log.Printf("Detected %v has filesystem type '%v'", path, t)
		return nil
	}
}
