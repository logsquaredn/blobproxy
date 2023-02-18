FROM scratch
ENTRYPOINT ["/blobproxy"]
COPY blobproxy /blobproxy
