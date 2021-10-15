# Build the manager binary
FROM alpine:3.7
WORKDIR /
COPY ./bin/manager .
COPY ./provider .
ENTRYPOINT ["./manager"]