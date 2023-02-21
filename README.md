# blobproxy  [![CI](https://github.com/logsquaredn/blobproxy/actions/workflows/ci.yml/badge.svg?branch=main&event=push)](https://github.com/logsquaredn/blobproxy/actions) [![godoc](https://pkg.go.dev/badge/github.com/logsquaredn/blobproxy.svg)](https://pkg.go.dev/github.com/logsquaredn/blobproxy) [![goreportcard](https://goreportcard.com/badge/github.com/logsquaredn/blobproxy)](https://goreportcard.com/report/github.com/logsquaredn/blobproxy) ![license](https://shields.io/github/license/logsquaredn/blobproxy)

Go `http.Handler` for serving the contents of a `gocloud.dev/blob.Bucket`, e.g. an s3 bucket.

## install

```sh
# application
go install github.com/logsquaredn/blobproxy/cmd/blobproxy
# module
go get github.com/logsquaredn/blobproxy
```

## use

```sh
$ blobproxy -h
Usage:
  blobproxy [--port|-p 8080] {s3|azblob|gs}://bucket [/prefix]

Flags:
  -h, --help                  help for blobproxy
  -p, --port int              port (default 8080)
  -V, --verbose count         verbose
  -v, --version               version for blobproxy
```

```sh
$ blobproxy s3://my-bucket /my-prefix
$ curl http://localhost:8080/my-prefix/my-bucket-object
```

> See https://gocloud.dev/concepts/urls/ for supported URL formats.

> Remember to escape `&` in the URL's query parameters. See [`docker-compose.yml`](docker-compose.yml) for an example.
