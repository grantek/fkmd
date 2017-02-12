# fkmd

Command-line utility to work with serial-based cartridge dumpers, specifically the krikzz.com Flashkit MD

Supports reading cartridge ROM, and reading and writing cartridge RAM.

Writing of the krikzz.com Flashcart MD cartridge flash is implemented but currently untested.

Use of the krikzz.com source code for the Windows C# utility is with permission

## Installation

- Set up go: create a directory for $GOPATH, and set the environment variable $GOPATH to it, eg. on Linux:
  - ``mkdir ~/golang``
  - ``export GOPATH=~/golang``
- Fetch via go
  - ``go get github.com/grantek/fkmd``

## Usage

fkmd usage:
  -autoname
        Read ROM name and generate filenames to save ROM/RAM data
  -port string
        serial port to use (/dev/ttyUSB0, etc) (default "/dev/ttyUSB0")
  -ramfile string
        File to save or read RAM data
  -readram
        Read and output RAM
  -readrom
        Read and output ROM
  -romfile string
        File to save or read ROM data
  -rominfo
        Print ROM info
  -writeram
        Write supplied RAM data to cartridge
  -writerom
        (Flash cart only) Write ROM data to flash

