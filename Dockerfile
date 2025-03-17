
# Step 1: Modules caching
FROM golang:1.23 AS base

# Move to working directory /build
WORKDIR /build
COPY go.mod go.sum ./
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY=https://goproxy.cn,direct
RUN go mod download

# Step 2: Builder
FROM golang:1.23 AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux
COPY --from=base /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN go build -o /bin/app .

# Step 3: Final
FROM alpine:latest
WORKDIR /home/works/program
COPY --from=builder /bin/app .
COPY conf ./conf
COPY --from=base /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
EXPOSE 8000
ENTRYPOINT ["./app"]
