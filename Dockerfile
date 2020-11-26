FROM golang:1.15-alpine3.12 as build
WORKDIR /go/src/github.com/orisano/bctx
RUN apk add --no-cache gcc musl-dev
COPY . .
RUN go build -o bin/bctx ./cmd/bctx

FROM alpine:3.12
RUN apk add --no-cache ca-certificates
COPY --from=build /go/src/github.com/orisano/bctx/bin/bctx /usr/bin
CMD ["bctx"]
