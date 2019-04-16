FROM alpine:latest as builder

# Chosing a working directory
WORKDIR /root

RUN apk add --no-cache ca-certificates

RUN apk update && \
    apk add libc6-compat git && \
    apk add --virtual .build-deps wget && \
    wget https://storage.googleapis.com/golang/go1.12.3.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.12.3.linux-amd64.tar.gz && \
    mkdir go && mkdir go/src && mkdir go/bin && mkdir go/pkg && \
    mkdir -p my/src/go-synology-dropbox-ds-adapter && \
    rm go1.12.3.linux-amd64.tar.gz && \
    apk --purge del .build-deps

# Setting environment variables for Go
ENV PATH=${PATH}:/usr/local/go/bin GOROOT=/usr/local/go GOPATH=/root/go CGO_ENABLED=0

# copying only go.mod/go.lock
COPY go.* /root/my/src/go-synology-dropbox-ds-adapter/
WORKDIR /root/my/src/go-synology-dropbox-ds-adapter
RUN go mod download

COPY . /root/my/src/go-synology-dropbox-ds-adapter/
RUN go build


FROM alpine:latest

# Chosing a working directory
WORKDIR /root
# Setting environment variables for Go
ENV PATH=${PATH}:/usr/local/go/bin GOROOT=/usr/local/go GOPATH=/root/go CGO_ENABLED=0

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /root/my/src/go-synology-dropbox-ds-adapter/go-synology-dropbox-ds-adapter .

EXPOSE 80

# Launching our server
CMD /root/go-synology-dropbox-ds-adapter
