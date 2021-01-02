# Send

![Go](https://github.com/z0rr0/send/workflows/Go/badge.svg)
[![GoDoc](https://godoc.org/github.com/z0rr0/send?status.svg)](https://pkg.go.dev/github.com/z0rr0/send?tab=subdirectories)

Send is a service to share private text and/or file data.

It supports:

- expiration time
- number of requests before deleting
- AES encryption without secret saving

## Build

```shell
make build
```

### Test

```shell
make test
```

### Development

```shell
# start / stop / restart
make start
```

Custom config file can be used from environment variable `SENDCFG`.

## License

This source code is governed by a MIT license that can be found
in the [LICENSE](https://github.com/z0rr0/send/blob/main/LICENSE) file.
