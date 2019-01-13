Published under MIT license with the permission of NX Studio.

# Memory mapping [![GoDoc](https://godoc.org/github.com/alexeymaximov/mmap?status.svg)](https://godoc.org/github.com/alexeymaximov/mmap) ![](https://img.shields.io/github/license/alexeymaximov/mmap.svg)

This is cross-platform Golang package for memory mapped file I/O.

Currently has been tested on following architectures:
* windows/amd64
* linux/amd64

## Installation

`$ go get github.com/alexeymaximov/mmap`

## TODO

* Support of all architectures which are available in Golang.
* Working set management on Windows.
* RLimit management on Linux.
* Memory segment interface (accessing typical integer values by offset).