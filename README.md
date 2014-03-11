#iobit [![GoDoc](https://godoc.org/github.com/bamiaux/iobit/web?status.png)](https://godoc.org/github.com/bamiaux/iobit) [![Build Status](https://travis-ci.org/bamiaux/iobit.png)](https://travis-ci.org/bamiaux/iobit)
Package iobit provides primitives for reading & writing bits The main purpose of this library is to remove the need to write custom bit-masks when reading or writing bitstreams, and to ease maintenance.

Download:
```shell
go get github.com/bamiaux/iobit
```


Full documentation at http://godoc.org/github.com/bamiaux/iobit

* * *
Package iobit provides primitives for reading & writing bits

The main purpose of this library is to remove the need to write
custom bit-masks when reading or writing bitstreams, and to ease
maintenance. This is true especially when you need to read/write
data which is not aligned on bytes.

For example, with iobit you can read an MPEG-TS PCR like this:

```
r := iobit.NewReader(buffer)
base := r.Uint64(33)     // PCR base is 33-bits
r.Skip(6)                // 6-bits are reserved
extension := r.Uint64(9) // PCR extension is 9-bits
```

instead of:

```
base  = uint64(buffer[0]) << 25
base |= uint64(buffer[1]) << 17
base |= uint64(buffer[2]) << 9
base |= uint64(buffer[3]) << 1
base |= buffer[4] >> 7
extension := uint16(buffer[4] & 0x1) << 8
extension |= buffer[5]
```

and write it like this:

```
w := iobit.NewWriter(buffer)
w.PutUint64(33, base)
w.PutUint32(6, 0)
w.PutUint32(9, extension)
```



