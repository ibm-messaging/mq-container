// +build linux

package main

import (
	log "github.com/sirupsen/logrus"
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

func checkFS(path string) {
	statfs := &unix.Statfs_t{}
	err := unix.Statfs(path, statfs)
	if err != nil {
		log.Println(err)
		return
	}
	t := fsTypes[statfs.Type]
	switch t {
	case "aufs", "overlayfs", "tmpfs":
		log.Fatalf("Error: %v uses unsupported filesystem type %v", path, t)
	default:
		log.Printf("Detected %v has filesystem type '%v'", path, t)
	}
}
