FROM alpine:3.16
ENTRYPOINT ["/blobproxy"]
COPY blobproxy /blobproxy
