FROM golang:latest AS compiling_stage
RUN mkdir -p /go/src/pipeline
WORKDIR /go/src/pipeline
ADD pipeline.go .
ADD go.mod .

RUN go install

FROM alpine:latest
LABEL version="1.0.0"
LABEL maintainer="anthony"
WORKDIR /root/
COPY --from=compiling_stage /go/bin/pipeline .
RUN chmod +x /root/pipeline
ENTRYPOINT ./pipeline