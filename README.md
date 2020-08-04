# fkmd

Command-line utilities to work with serial-based cartridge dumpers, originally the krikzz.com Flashkit MD.

- [``sfmd``](#sfmd): Supports the krikzz.com Flashkit MD.
- [``sfgb``](#sfgb): (alpha) Will support the Game Boy Cart Flasher.

Tested on Linux, but should also work on Windows and MacOS (as supported by ``github.com/jacobsa/go-serial``).

## Installation

- Set up go: create a directory for $GOPATH, and set the environment variable $GOPATH to it, eg. on Linux:
  - ``mkdir ~/golang``
  - ``export GOPATH=~/golang``
- Fetch via go:
  - ``go get github.com/grantek/fkmd``
- Build and install into $GOPATH
  - ``go install github.com/grantek/fkmd/sfmd``
  - ``go install github.com/grantek/fkmd/sfgb``
  - Or the old version: ``go install github.com/grantek/fkmd``
- Run installed binary:
  - ``$GOPATH/bin/fkmd``

## Utilities

### fkmd (original utility)

"V1" utility supports reading cartridge ROM, writing a ROM image to a Flashcart MD device, and reading and writing cartridge RAM.

Use of the krikzz.com source code for the Windows C# utility is with permission.

### sfmd

"V2" utility does the same thing, but written in a different style.

### sfgb

WIP, currently supported flags: ``-rominfo`` ``-readram``

Game Boy cart flasher documented by [jrodrigo.net/cart-flasher](https://www.jrodrigo.net/es/project/gameboy-cart-flasher/) and [www.reinerziegler.de/readplus.htm](https://web.archive.org/web/20120403050446/http://www.reinerziegler.de/readplus.htm#GB_Flasher)
Original PC driver software from [sourceforge.net/projects/gbcf](https://sourceforge.net/projects/gbcf)

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
Usage of sfgb:
  -autoname
      Read ROM name and generate filenames to save ROM/RAM data
  -baud uint
      Baud rate (default 185000)
  -debug
      Output debug logs to stderr (implies verbose)
  -port string
      serial port to use (/dev/ttyUSB0, etc) (default "/dev/ttyUSB0")
  -ramfile string
      File to save or read RAM data (- for STDOUT/STDIN)
  -ramsize int
      Size of RAM (0 to autodetect)
  -readram
      Read and save RAM
  -readrom
      Read and save ROM
  -romfile string
      File to save or read ROM data (- for STDOUT/STDIN)
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

These should be automatically installed when you use "go get" to fetch this repository.

all:

- github.com/jacobsa/go-serial

gbcf:

- go tools (golang.org/x/tools/cmd/stringer)

