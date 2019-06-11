// +build linux

/*
© Copyright IBM Corporation 2017, 2019

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
package containerruntime

import (
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
	0x58465342: "xfs",
	// less popular codes
	0xadf5:     "adfs",
	0xadff:     "affs",
	0x5346414F: "afs",
	0x0187:     "autofs",
	0x73757245: "coda",
	0x28cd3d45: "cramfs",
	0x453dcd28: "cramfs",
	0x64626720: "debugfs",
	0x73636673: "securityfs",
	0xf97cff8c: "selinux",
	0x43415d53: "smack",
	0x858458f6: "ramfs",
	0x958458f6: "hugetlbfs",
	0x73717368: "squashfs",
	0xf15f:     "ecryptfs",
	0x414A53:   "efs",
	0xabba1974: "xenfs",
	0x3434:     "nilfs",
	0xF2F52010: "f2fs",
	0xf995e849: "hpfs",
	0x9660:     "isofs",
	0x72b6:     "jffs2",
	0x6165676C: "pstorefs",
	0xde5e81e4: "efivarfs",
	0x00c0ffee: "hostfs",
	0x137F:     "minix_14",  // minix v1 fs, 14 char names
	0x138F:     "minix_30",  // minix v1 fs, 30 char names
	0x2468:     "minix2_14", // minix v2 fs, 14 char names
	0x2478:     "minix2_30", // minix v2 fs, 30 char names
	0x4d5a:     "minix3_60", // minix v3 fs, 60 char names
	0x4d44:     "msdos",
	0x564c:     "ncp",
	0x7461636f: "ocfs2",
	0x9fa1:     "openprom",
	0x002f:     "qnx4",
	0x68191122: "qnx6",
	0x6B414653: "afs_fs",
	0x52654973: "reiserfs",
	0x517B:     "smb",
	0x27e0eb:   "cgroup",
	0x63677270: "cgroup2",
	0x7655821:  "rdtgroup",
	0x57AC6E9D: "stack_end",
	0x74726163: "tracefs",
	0x01021997: "v9fs",
	0x62646576: "bdevfs",
	0x64646178: "daxfs",
	0x42494e4d: "binfmtfs",
	0x1cd1:     "devpts",
	0xBAD1DEA:  "futexfs",
	0x50495045: "pipefs",
	0x9fa0:     "proc",
	0x534F434B: "sockfs",
	0x62656572: "sysfs",
	0x9fa2:     "usbdevice",
	0x11307854: "mtd_inode",
	0x09041934: "anon_inode",
	0x73727279: "btrfs",
	0x6e736673: "nsfs",
	0xcafe4a11: "bpf",
	0x5a3c69f0: "aafs",
	0x15013346: "udf",
	0x13661366: "balloon_kvm",
	0x58295829: "zsmalloc",
}

// GetFilesystem returns the filesystem type for the specified path
func GetFilesystem(path string) (string, error) {
	statfs := &unix.Statfs_t{}
	err := unix.Statfs(path, statfs)
	if err != nil {
		return "", err
	}
	// Use a type conversion to make type an int64.  On s390x it's a uint32.
	t, ok := fsTypes[int64(statfs.Type)]
	if !ok {
		return "unknown", nil
	}
	return t, nil
}
