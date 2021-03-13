FROM golang:1.16 as build
WORKDIR /go/src/github.com/orisano/bctx
COPY . .
RUN go build -o bin/bctx ./cmd/bctx

FROM gcr.io/distroless/static
COPY --from=build /go/src/github.com/orisano/bctx/bin/bctx /usr/bin
CMD ["bctx"]
