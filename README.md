# fkmd

Command-line utilities to work with serial-based cartridge dumpers, originally the krikzz.com Flashkit MD.

Tested on Linux, but should also work on Windows and MacOS (as supported by ``github.com/jacobsa/go-serial``).

## fkmd

"V1" utility supports reading cartridge ROM, writing a ROM image to a Flashcart MD device, and reading and writing cartridge RAM.

Use of the krikzz.com source code for the Windows C# utility is with permission

### sfmd

"V2" utility does the same thing, but written in a different style.

## Installation

- Set up go: create a directory for $GOPATH, and set the environment variable $GOPATH to it, eg. on Linux:
  - ``mkdir ~/golang``
  - ``export GOPATH=~/golang``
- Fetch via go:
  - ``go get github.com/grantek/fkmd``
- Build and install into $GOPATH
  - ``go install github.com/grantek/fkmd/sfmd``
  - Or the old version: ``go install github.com/grantek/fkmd``
- Run installed binary:
  - ``$GOPATH/bin/fkmd``

## Usage
```
Usage of sfmd:
  -autoname
      Read ROM name and generate filenames to save ROM/RAM data
  -debug
      Output debug logs to stderr (implies verbose)
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
  -verbose
      Output info logs to stderr
  -writeram
      Write supplied RAM data to cartridge
  -writerom
      (Flash cart only) Write ROM data to flash
```

```
Usage of fkmd:
  -autoname
      Read ROM name and generate filenames to save ROM/RAM data
  -port string
      serial port to use (/dev/ttyUSB0, etc) (default "/dev/ttyUSB0")
  -ramfile string
      File to save or read RAM data
  -rangeend int
      Do not probe size, end at this byte
  -rangestart int
      Do not probe size, start at this byte (requires end)
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
```

## Dependencies

- github.com/jacobsa/go-serial

gbcf:
- github.com/howeyc/crc16
- go tools (golang.org/x/tools/cmd/stringer)
