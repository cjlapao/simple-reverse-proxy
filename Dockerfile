############################
# STEP 1 build executable binary
############################
FROM golang:alpine AS builder

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git

WORKDIR /go/src
COPY . .

RUN sed -i 's/localhost/host.docker.internal/g' /go/src/config.json

# Using go get.
RUN go get -d -v

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/proxy.service

############################
# STEP 2 build a small image
############################
FROM scratch

# Add ca-certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy our static executable.
COPY --from=builder /go/bin/proxy.service /go/bin/proxy.service

# Copy our configuration file and replace localhost with host.docker.internal.
COPY --from=builder /go/src/config.json /go/bin/config.json

ENV PORT=80

WORKDIR /go/bin

EXPOSE 80

ENTRYPOINT ["/go/bin/proxy.service"]