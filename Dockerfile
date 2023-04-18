FROM golang:1.20 as build
WORKDIR /go/src/app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -C . -o /go/bin/app

FROM scratch
COPY --from=build /go/bin/app /

ENTRYPOINT ["/app"]