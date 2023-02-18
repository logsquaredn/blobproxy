# blobproxy

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
  blobproxy {s3|azblob|gs}://bucket [/prefix] [--s3-endpoint=http://minio:9000/] [--s3-force-path-style] [--s3-disable-ssl] [flags]

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

> Remember to escape `&` in the URL's query parameters. See [`docker.compose.yml`](docker.compose.yml) for an example.
