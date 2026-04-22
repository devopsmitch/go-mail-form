FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /mailform .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl && \
    adduser -D -H mailform
COPY --from=build /mailform /mailform
USER mailform
EXPOSE 3000
ENTRYPOINT ["/mailform"]
