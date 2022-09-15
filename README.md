# Koker

Building a Docker-like tool - `koker` - Kien's mini docker.

> What I cannot create, I do not understand — Richard Feynman

## 1. Introduction

- What is **Koker**?
  - Koker is a tiny educational-purpose Docker-like tool, written in Golang.
  - Unlike Docker, Koker just uses a set of Linux's operating system primitives that provide the illusion of a container. Tt uses neither [containerd](https://containerd.io/) nor [runc](https://github.com/opencontainers/runc).
- Why **Koker**?
  - Have you ever wondered how Docker containers are constructed?
  - Koker provides an understanding of how extactly containers work at the Linux system call level by using logging (every steps!).
    - Control Groups for resource restriction (CPU, Memory, Swap, PIDs).
    - Namespace for global system resources isolation (Mount, UTS, Network, IPS, PID).
    - Union File System for branches to be overlaid in a single coherent file system. (OverlayFS)
- Should I use **Koker** in production?
  - Nope, Koker isn't a production ready tool.
- **Koker** is highly inspired by:
  - [Bocker](https://github.com/p8952/bocker).
  - [Containers-the-hard-way](https://github.com/shuveb/containers-the-hard-way)
  - [Vessel](https://github.com/0xc0d/vessel)

## 2. Getting started

- Install:

```bash
$ go get -u github.com/ntk148v/koker
```

```
$ koker --help
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

```
$ koker container --help
NAME:
   koker container - Manage container

USAGE:
   koker container command [command options] [arguments...]

COMMANDS:
     run      Run a command in a new container
     child
     rm       Remove a container (WIP)
     ls       List running containers
     help, h  Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help (default: false)
```

```
$ koker image --help
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
