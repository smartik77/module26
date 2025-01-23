FROM golang AS compiling_stage
RUN mkdir -p /go/src/module26
WORKDIR /go/src/module26
ADD pipeline.go .
ADD go.mod .
RUN go install .

FROM alpine:latest
LABEL version="1.0.0"
LABEL maintainer="anthony"
WORKDIR /root/
COPY --from=compiling_stage /go/bin/module26 .
ENTRYPOINT ./module26
EXPOSE 8080