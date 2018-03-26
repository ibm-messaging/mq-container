# Troubleshooting

## AMQ7017: Log not available
If you see this message in the container logs, it means that the directory being used for the container's volume doesn't use a filesystem supported by IBM MQ.  To solve this, you need to make sure the container's `/mnt/mqm` volume is put on a supported filesystem.  The best way to do this is to use [Docker volumes](https://docs.docker.com/storage/volumes/), instead of bind-mounted directories.

## Container command not found or does not exist
This message also appears as "System error: no such file or directory" in some versions of Docker.  This can happen using a Docker client on Windows, and is related to line-ending characters.  When you clone the Git repository on Windows, Git is often configured to convert any UNIX-style LF line-endings to Windows-style CRLF line-endings.  Files with these line-endings end up in the built Docker image, and cause the container to fail at start-up.  One solution to this problem is to stop Git from converting the line-ending characters, with the following command:

```
git config --global core.autocrlf input
```

## Old Linux kernel versions
MQ works best if you have a Linux kernel version of V3.16 or higher (run `uname -r` to check).

If you have an older version, you might need to add the [`--ipc host`](https://docs.docker.com/engine/reference/run/#ipc-settings-ipc) option when you run an MQ container.  The reason for this is that IBM MQ uses shared memory, and on Linux kernels prior to V3.16, containers are usually limited to 32 MB of shared memory.  In a [change](https://git.kernel.org/cgit/linux/kernel/git/mhocko/mm.git/commit/include/uapi/linux/shm.h?id=060028bac94bf60a65415d1d55a359c3a17d5c31
) to Linux kernel V3.16, the hard-coded limit is greatly increased.  This kernel version is available in Ubuntu 14.04.2 onwards, Fedora V20 onwards, and boot2docker V1.2 onwards.  Some Linux distributions, like Red Hat Enterprise Linux, patch older kernel versions, so you might find that the patch has been applied already, even if you see a lower kernel version number.  If you are using a host with an older kernel version, then you can still run MQ, but you have to give it access to the host's IPC namespace using the [`--ipc host`](https://docs.docker.com/engine/reference/run/#ipc-settings-ipc) option on `docker run`.  Note that this reduces the security isolation of your container.
