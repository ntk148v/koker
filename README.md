<h1 align="center">Koker</h1>

<p align="center">Building a <b>Koker</b> - Kien's mini Docker.</p>
<p align="center"><i>What I cannot create, I do not understand — Richard Feynman</i></p>

<p align="center">
    <a href="https://github.com/ntk148v/koker/blob/master/LICENSE">
        <img alt="GitHub license" src="https://img.shields.io/github/license/ntk148v/koker?style=for-the-badge">
    </a>
    <a href="https://github.com/ntk148v/koker/stargazers"><img src="https://img.shields.io/github/stars/ntk148v/koker?colorA=192330&colorB=719cd6&style=for-the-badge"></a>
    <a href="https://github.com/ntk148v/koker/issues"><img src="https://img.shields.io/github/issues/ntk148v/koker?colorA=192330&colorB=dbc074&style=for-the-badge"></a>
    <a href="https://github.com/ntk148v/koker/contributors"><img src="https://img.shields.io/github/contributors/ntk148v/koker?colorA=192330&colorB=81b29a&style=for-the-badge"></a>
<a href="https://github.com/ntk148v/koker/network/members"><img src="https://img.shields.io/github/forks/ntk148v/koker?colorA=192330&colorB=9d79d6&style=for-the-badge"></a>
</p>

- [1. Introduction](#1-introduction)
- [2. Getting started](#2-getting-started)
- [3. Examples](#3-examples)
- [4. Contributing](#4-contributing)

## 1. Introduction

- What is **Koker**?
  - Koker is a tiny educational-purpose Docker-like tool, written in Golang.
  - Unlike Docker, Koker just uses a set of Linux's operating system primitives that provide the illusion of a container. It uses neither [containerd](https://containerd.io/) nor [runc](https://github.com/opencontainers/runc).
- Why **Koker**?
  - Have you ever wondered how Docker containers are constructed?
  - Koker provides an understanding of how extactly containers work at the Linux system call level by using logging (every steps!).
    - Control Groups for resource restriction (CPU, Memory, Swap, PIDs). _All CGroups modes (Legacy - v1, Hybrid - v1 & v2, Unified - v2) are handled_.
    - Namespace for global system resources isolation (Mount, UTS, Network, IPS, PID).
    - Union File System for branches to be overlaid in a single coherent file system. (OverlayFS)
    - Container networking using bridge and iptables.
- **Koker** is highly inspired by:
  - [Bocker](https://github.com/p8952/bocker).
  - [Containers-the-hard-way](https://github.com/shuveb/containers-the-hard-way)
  - [Vessel](https://github.com/0xc0d/vessel)
- Should I use **Koker** in production?
  - Nope, Koker isn't a production ready tool!
- Can **Koker** perform every Docker tasks?
  - Nope, ofc, Koker doesn't aim to recreate every Docker's tasks. There are just some simple tasks for educational-purpose.
- If you are looking for a tool similar to Docker in terms of functionality but do not require a daemon to operate, Podman is a suitable choice for you.

## 2. Getting started

- Install:

```shell
$ go get -u github.com/ntk148v/koker
# Or you can build yourself
$ git clone https://github.com/ntk148v/koker.git koker
$ cd koker
$ make build
$ sudo /tmp/koker --help
```

- You can also download binary file in [releases](https://github.com/ntk148v/koker/releases).

- Usage:
  - Note that you must have root permission to execute koker.

```shell
$ sudo koker --help
NAME:
   koker - Kien's mini Docker

USAGE:
   koker [global options] command [command options] [arguments...]

VERSION:
   v0.0.1

AUTHOR:
   Kien Nguyen-Tuan <kiennt2609@gmail.com>

COMMANDS:
   container, c  Manage container
   image, i      Manage images
   help, h       Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug, -D    Set log level to debug. You will see step-by-step what were executed (default: false)
   --help, -h     show help (default: false)
   --quiet, -q    Disable logging altogether (quiet mode) (default: false)
   --version, -v  print the version (default: false)
```

```shell
$ sudo koker container --help
NAME:
   koker container - Manage container

USAGE:
   koker container command [command options] [arguments...]

COMMANDS:
     run      Run a command in a new container
     child
     rm       Remove a container (WIP)
     ls       List running containers
     exec     Run a command inside a running container
     help, h  Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help (default: false)
```

```shell
$ sudo koker image --help
NAME:
   koker image - Manage images

USAGE:
   koker image command [command options] [arguments...]

COMMANDS:
     ls       List all available images
     pull     Pull an image or a repository from a registry (using image's name)
     rm       Remove a image (using image's name)
     help, h  Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help (default: false)
```

## 3. Examples

- Start container and execute command.

```shell
$ sudo koker -D container run --hostname test --mem 1024 alpine sh # Enable debugging

11:08AM INF Load image repository from file repository=/var/lib/koker/images/repositories.json
11:08AM DBG Load image repository
11:08AM DBG Check default bridge is up or not bridge=koker0
11:08AM INF Setup network for container container=ccjuq1p3l1hn8clpgib0
11:08AM DBG Setup virtual ethernet peer=veth1_ccjumf9 virt=veth0_ccjumf9
11:08AM DBG Set the master of the link device link=veth0_ccjumf9 master=koker0
11:08AM DBG Mount new network namespace netns=/var/lib/koker/netns/ccjuq1p3l1hn8clpgib0
11:08AM DBG Call syscall unshare CLONE_NEWNET netns=/var/lib/koker/netns/ccjuq1p3l1hn8clpgib0
11:08AM DBG Mount new network namespace netns=/var/lib/koker/netns/ccjuq1p3l1hn8clpgib0
11:08AM DBG Mount target source=/proc/self/ns/net target=/var/lib/koker/netns/ccjuq1p3l1hn8clpgib0
11:08AM DBG Set network namespace netns=/var/lib/koker/netns/ccjuq1p3l1hn8clpgib0
11:08AM DBG Put link device into a new network namespace link=veth1_ccjumf9 netns=/var/lib/koker/netns/ccjuq1p3l1hn8clpgib0
11:08AM DBG Set network namespace by file netns=/var/lib/koker/netns/ccjuq1p3l1hn8clpgib0
11:08AM DBG Change the name of the link device newname=eth0 oldname=veth1_ccjumf9
11:08AM DBG Add IP address to the ip device ip=172.69.216.38/16 link=eth0
11:08AM DBG Enable the link device link=eth0
11:08AM DBG Set gateway for the link device gateway=172.69.0.1 link=eth0
11:08AM DBG Enable the link device link=lo
11:08AM INF Construct new Image instance image=alpine
11:08AM INF Image exists, reuse image=index.docker.io/library/alpine:latest
11:08AM INF Mount filesystem for container from an image container=ccjuq1p3l1hn8clpgib0 image=index.docker.io/library/alpine:latest
11:08AM DBG Mount target source=none target=/var/lib/koker/containers/ccjuq1p3l1hn8clpgib0/mnt
11:08AM DBG Copy container config from image config container=ccjuq1p3l1hn8clpgib0 image=library/alpine
11:08AM INF Load image repository from file repository=/var/lib/koker/images/repositories.json
11:08AM DBG Load image repository
11:08AM DBG Load container config from file container=ccjuq1p3l1hn8clpgib0
11:08AM INF Set hostname container=ccjuq1p3l1hn8clpgib0
11:08AM INF Set container's limit using cgroup container=ccjuq1p3l1hn8clpgib0
11:08AM DBG Set container's memory limit container=ccjuq1p3l1hn8clpgib0
11:08AM DBG Set container's pids limit container=ccjuq1p3l1hn8clpgib0
11:08AM DBG Set container's cpus limit container=ccjuq1p3l1hn8clpgib0
11:08AM INF Copy nameserver config container=ccjuq1p3l1hn8clpgib0
11:08AM INF Execute command container=ccjuq1p3l1hn8clpgib0
11:08AM DBG Set network namespace container=ccjuq1p3l1hn8clpgib0
11:08AM DBG Set network namespace by file netns=/var/lib/koker/netns/ccjuq1p3l1hn8clpgib0
11:08AM DBG Mount target source=tmpfs target=dev
11:08AM DBG Mount target source=proc target=proc
11:08AM DBG Mount target source=sysfs target=sys
11:08AM DBG Mount target source=tmpfs target=tmp
11:08AM DBG Execute command command=sh container=ccjuq1p3l1hn8clpgib0
/ #
# Hit <Ctrl+c>
11:09AM DBG Unmount target source=tmpfs target=dev
11:09AM DBG Unmount target source=proc target=proc
11:09AM DBG Unmount target source=sysfs target=sys
11:09AM DBG Unmount target source=tmpfs target=tmp
11:09AM INF Save image repository to file repository=/var/lib/koker/images/repositories.json
11:09AM DBG Unmount target source=none target=/var/lib/koker/containers/ccjuq1p3l1hn8clpgib0/mnt
11:09AM DBG Unmount target source=/proc/self/ns/net target=/var/lib/koker/netns/ccjuq1p3l1hn8clpgib0
11:09AM INF Delete container container=ccjuq1p3l1hn8clpgib0
11:09AM DBG Remove container's directory container=ccjuq1p3l1hn8clpgib0
11:09AM DBG Remove container's network namespace container=ccjuq1p3l1hn8clpgib0
11:09AM DBG Remove container cgroups container=ccjuq1p3l1hn8clpgib0
11:09AM INF Save image repository to file repository=/var/lib/koker/images/repositories.json
```

- Pull and list image(s).

```shell
$ sudo koker -D image pull alpine
$ sudo koker -D image ls
11:13AM INF Load image repository from file repository=/var/lib/koker/images/repositories.json
11:13AM DBG Load image repository

REPOSITORY              TAG             IMAGE ID

jwilder/whoami          latest          4a4c1589a078

library/alpine          3.16.2          0261ca8a4a79

library/alpine          edge            9a2e669787f4

library/alpine          latest          0261ca8a4a79

11:13AM INF Save image repository to file repository=/var/lib/koker/images/repositories.json
```

- List all available containers (run a container then run the above command in the another session).

```shell
$ koker -D container ls
11:11AM INF Load image repository from file repository=/var/lib/koker/images/repositories.json
11:11AM DBG Load image repository
11:11AM DBG Load container config from file container=ccjuo013l1hkmh7sk540

CONTAINER ID            IMAGE                   COMMAND

ccjuq1p3l1hn8clpgib0    0261ca8a4a79            sh

11:11AM INF Save image repository to file repository=/var/lib/koker/images/repositories.json
```

- Run a command inside a running container.

```shell
$ sudo koker -D container exec ccjuq1p3l1hn8clpgib0 sh
11:17AM INF Load image repository from file repository=/var/lib/koker/images/repositories.json
11:17AM DBG Load image repository
11:17AM DBG Load container config from file container=ccjuq1p3l1hn8clpgib0
11:17AM INF Execute command container=ccjuq1p3l1hn8clpgib0
11:17AM DBG Execute command command=sh container=ccjuq1p3l1hn8clpgib0
/ #
```

- Run container with limited resource.
  - Run golang-memtest container to allocate 20MiB in 10MiB container.

```shell
$ sudo koker container run --mem 10 fabianlee/golang-memtest:1.0.0 20
9:20AM INF Load image repository from file repository=/var/lib/koker/images/repositories.json
9:20AM INF Setup network for container container=cdmr2vmfvq0sn7gs7r80
9:20AM INF Construct new Image instance image=fabianlee/golang-memtest:1.0.0
9:20AM INF Image exists, reuse image=index.docker.io/fabianlee/golang-memtest:1.0.0
9:20AM INF Mount filesystem for container from an image container=cdmr2vmfvq0sn7gs7r80 image=index.docker.io/fabianlee/golang-memtest:1.0.0
9:20AM INF Load image repository from file repository=/var/lib/koker/images/repositories.json
9:20AM INF Set hostname container=cdmr2vmfvq0sn7gs7r80
9:20AM INF Set container's limit using cgroup container=cdmr2vmfvq0sn7gs7r80
9:20AM INF Copy nameserver config container=cdmr2vmfvq0sn7gs7r80
9:20AM INF Execute command container=cdmr2vmfvq0sn7gs7r80
Alloc = 0 MiB   TotalAlloc = 0 MiB      Sys = 68 MiB    NumGC = 0
Asked to allocate 20Mb

Alloc = 1 MiB   TotalAlloc = 1 MiB      Sys = 68 MiB    NumGC = 0
Alloc = 2 MiB   TotalAlloc = 2 MiB      Sys = 68 MiB    NumGC = 0
Alloc = 3 MiB   TotalAlloc = 3 MiB      Sys = 68 MiB    NumGC = 0
Alloc = 4 MiB   TotalAlloc = 4 MiB      Sys = 68 MiB    NumGC = 1
Alloc = 5 MiB   TotalAlloc = 5 MiB      Sys = 68 MiB    NumGC = 1
Alloc = 6 MiB   TotalAlloc = 6 MiB      Sys = 68 MiB    NumGC = 1
Alloc = 7 MiB   TotalAlloc = 7 MiB      Sys = 68 MiB    NumGC = 1
Alloc = 8 MiB   TotalAlloc = 8 MiB      Sys = 68 MiB    NumGC = 2
9:20AM ERR Something went wrong error="error running child command: signal: killed"
9:20AM INF Save image repository to file repository=/var/lib/koker/images/repositories.json
9:20AM INF Delete container container=cdmr2vmfvq0sn7gs7r80
9:20AM INF Save image repository to file repository=/var/lib/koker/images/repositories.json
```

- Connect to outside the world from the container:

```shell
$ sudo koker -D container run alpine sh
11:25AM INF Load image repository from file repository=/var/lib/koker/images/repositories.json
11:25AM DBG Load image repository
11:25AM DBG Check default bridge is up or not bridge=koker0
11:25AM DBG Append POSTROUTING rule outgoing=koker0 source=172.69.0.0/16
11:25AM INF Setup network for container container=cui3je94h280e1c7nae0
11:25AM DBG Setup virtual ethernet peer=veth1_cui3je9 virt=veth0_cui3je9
11:25AM DBG Set the master of the link device link=veth0_cui3je9 master=koker0
11:25AM DBG Mount new network namespace netns=/var/lib/koker/netns/cui3je94h280e1c7nae0
11:25AM DBG Call syscall unshare CLONE_NEWNET netns=/var/lib/koker/netns/cui3je94h280e1c7nae0
11:25AM DBG Mount new network namespace netns=/var/lib/koker/netns/cui3je94h280e1c7nae0
11:25AM DBG Mount target source=/proc/self/ns/net target=/var/lib/koker/netns/cui3je94h280e1c7nae0
11:25AM DBG Set network namespace netns=/var/lib/koker/netns/cui3je94h280e1c7nae0
11:25AM DBG Put link device into a new network namespace link=veth1_cui3je9 netns=/var/lib/koker/netns/cui3je94h280e1c7nae0
11:25AM DBG Set network namespace by file netns=/var/lib/koker/netns/cui3je94h280e1c7nae0
11:25AM DBG Change the name of the link device newname=eth0 oldname=veth1_cui3je9
11:25AM DBG Add IP address to the ip device ip=172.69.98.139/16 link=eth0
11:25AM DBG Enable the link device link=eth0
11:25AM DBG Set gateway for the link device gateway=172.69.0.1 link=eth0
11:25AM DBG Enable the link device link=lo
11:25AM INF Construct new Image instance image=alpine
11:25AM INF Image exists, reuse image=index.docker.io/library/alpine:latest
11:25AM INF Mount filesystem for container from an image container=cui3je94h280e1c7nae0 image=index.docker.io/library/alpine:latest
11:25AM DBG Mount target source=none target=/var/lib/koker/containers/cui3je94h280e1c7nae0/mnt
11:25AM DBG Copy container config from image config container=cui3je94h280e1c7nae0 image=library/alpine
11:25AM INF Load image repository from file repository=/var/lib/koker/images/repositories.json
11:25AM DBG Load image repository
11:25AM DBG Load container config from file container=cui3je94h280e1c7nae0
11:25AM INF Set hostname container=cui3je94h280e1c7nae0
11:25AM INF Set container's limit using cgroup container=cui3je94h280e1c7nae0
11:25AM DBG Set container's memory limit container=cui3je94h280e1c7nae0
11:25AM DBG Set container's pids limit container=cui3je94h280e1c7nae0
11:25AM DBG Set container's cpus limit container=cui3je94h280e1c7nae0
11:25AM INF Copy nameserver config container=cui3je94h280e1c7nae0
11:25AM INF Execute command container=cui3je94h280e1c7nae0
11:25AM DBG Set network namespace container=cui3je94h280e1c7nae0
11:25AM DBG Set network namespace by file netns=/var/lib/koker/netns/cui3je94h280e1c7nae0
11:25AM DBG Mount target source=tmpfs target=dev
11:25AM DBG Mount target source=proc target=proc
11:25AM DBG Mount target source=sysfs target=sys
11:25AM DBG Mount target source=tmpfs target=tmp
11:25AM DBG Execute command command=sh container=cui3je94h280e1c7nae0
/ # ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
53: eth0@if54: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1500 qdisc noqueue state UP qlen 1000
    link/ether 72:eb:c0:fd:87:0c brd ff:ff:ff:ff:ff:ff
    inet 172.69.98.139/16 brd 172.69.255.255 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 fe80::70eb:c0ff:fefd:870c/64 scope link
       valid_lft forever preferred_lft forever
/ # ping 8.8.8.8
PING 8.8.8.8 (8.8.8.8): 56 data bytes
64 bytes from 8.8.8.8: seq=0 ttl=111 time=21.645 ms
^C
--- 8.8.8.8 ping statistics ---
1 packets transmitted, 1 packets received, 0% packet loss
round-trip min/avg/max = 21.645/21.645/21.645 ms
```

- If you find logging is annoying, ignore them with "--quiet" option.

```shell
$ sudo koker -q container ls
CONTAINER ID            IMAGE                   COMMAND

ccjuq1p3l1hn8clpgib0    0261ca8a4a79            sh

$ sudo koker -q container run --hostname test --mem 1024 alpine sh
/ #
```

- Note if you hit this kind of error, you should check the current max open files.

```bash
5:09PM ERR Something went wrong error="unable to pull image: unable to extract tarball's layer: open /var/lib/koker/images/2d389e545974d4a93ebdef09b650753a55f72d1ab4518d17a30c0e1b3e297444/31b3f1ad4ce1f369084d0f959813c51df0ca17d9877d5ee88c2db6ff88341430/usr/lib/x86_64-linux-gnu/perl-base/unicore/lib/Age/V80.pl: too many open files"
```

```shell
# Check current max open files
ulimit -n
# Change value to appropriate value, for example 4096
ulimit -n 4096
```

## 4. Contributing

Pull requests and issues are alway welcome!
