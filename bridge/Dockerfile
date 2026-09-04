FROM golang:1.25-alpine AS build

WORKDIR /src
COPY main.go .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /ai-platform-bridge main.go

FROM alpine:3.22
RUN adduser -D -H -u 10001 app
COPY --from=build /ai-platform-bridge /usr/local/bin/ai-platform-bridge
USER app
EXPOSE 3001
ENTRYPOINT ["/usr/local/bin/ai-platform-bridge"]
