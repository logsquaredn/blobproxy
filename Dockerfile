ARG build_image=golang:1.20-alpine3.16

FROM ${build_image} as build
ENV CGO_ENABLED 0
WORKDIR $GOPATH/src/github.com/logsquaredn/blobproxy
ARG semver=0.0.0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags "-s -w -X github.com/logsquaredn/blobproxy.Semver=${semver}" -o /assets/blobproxy ./cmd/blobproxy

FROM scratch AS blobproxy
ENTRYPOINT ["/blobproxy"]
COPY --from=build /assets /
