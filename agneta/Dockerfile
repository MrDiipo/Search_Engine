
FROM golang:1.18 AS BUILDER

# Pull build dependencies
WORKDIR $GOPATH/src/Search_Engine
COPY . .
RUN make deps
#
## Build static image.
#RUN GIT_SHA=$(git rev-parse --short HEAD) && \
#    CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
#    go build -a \
#    -ldflags "-extldflags '-static' -w -s -X main.appSha=$GIT_SHA" \
#    -o /go/bin/agneta-monolith \
#    Search_Engine

########################################################
# STEP 2 create alpine image with trusted certs
########################################################

FROM alpine:3.10
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /go/bin/agneta-monolith /go/bin/agneta-monolith

ENTRYPOINT ["/go/bin/agneta-monolith"]