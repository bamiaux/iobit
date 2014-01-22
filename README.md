iobit [![Codebot](https://codebot.io/badge/github.com/bamiaux/iobit.png)](http://codebot.io/doc/pkg/github.com/bamiaux/iobit "Codebot")
=====

Package iobit provides primitives for reading & writing bits

The main purpose of this library is to remove the need to write
custom bit-masks when reading or writing bitstreams, and to ease
maintenance. This is true especially when you need to read/write
data which is not aligned on bytes.

*iobit is an open source library under the MIT license*

#### Documentation

Documentation is available at http://godoc.org/github.com/bamiaux/iobit

## Installation

#### Into the gopath

```
    go get github.com/bamiaux/iobit
```

#### Import it in your code

```go
    import (
        "github.com/bamiaux/iobit"
    )
```

## Usage

### Reading

```go
    var buffer []byte
    r := iobit.NewReader(buffer)
    base := r.Uint64(33)     // PCR base is 33-bits
    r.Skip(6)                // 6-bits are reserved
    extension := r.Uint64(9) // PCR extension is 9-bits
    if err := r.Check(); err != nil {
        return err
    }
```

### Writing

```go
    var buffer []byte
    w := iobit.NewWriter(buffer)
    w.PutUint64(33, base)
    w.PutUint32(6, 0)
    w.PutUint32(9, extension)
    if err := w.Flush(); err != nil {
        return err
    }
```
