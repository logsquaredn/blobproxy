# blobproxy  [![CI](https://github.com/logsquaredn/blobproxy/actions/workflows/ci.yml/badge.svg?branch=main&event=push)](https://github.com/logsquaredn/blobproxy/actions) [![godoc](https://pkg.go.dev/badge/github.com/logsquaredn/blobproxy.svg)](https://pkg.go.dev/github.com/logsquaredn/blobproxy) ![license](https://shields.io/github/license/logsquaredn/blobproxy)

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
$ blobproxy s3://my-bucket /my-prefix
$ curl http://localhost:8080/my-prefix/my-bucket-object
```

> See https://gocloud.dev/concepts/urls/ for supported URL formats.
> Remember to escape `&` in the URL's query parameters.

### signed URLs

`use_signed_urls=true` answers each request with a `307` to a signed URL rather than
serving the object's bytes, so they travel from the store to the client directly.

```sh
$ blobproxy 's3://my-bucket?use_signed_urls=true&signed_url_expiry=1h' /my-prefix
```

`signed_url_expiry` sets how long a signed URL stays valid (default `1h`).

The redirect is served with a `Cache-Control` of nine tenths of that expiry, so clients
reuse one signed URL — and therefore their cached copy of the object — instead of being
handed a fresh URL, and a fresh cache key, on every request. The remaining tenth is
headroom, so a client following a cached redirect always has a live URL to fetch.
