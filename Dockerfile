FROM golang:1.26-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION}" -o /mailform .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl && \
    adduser -D -H mailform
COPY --from=build /mailform /mailform
USER mailform
EXPOSE 3000
ENTRYPOINT ["/mailform"]
