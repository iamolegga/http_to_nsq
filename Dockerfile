FROM scratch
COPY http_to_nsq /
ENTRYPOINT ["/http_to_nsq"]
