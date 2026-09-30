FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /comeandfind .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /comeandfind /app/comeandfind
COPY schema.sql /app/schema.sql
COPY web /app/web
EXPOSE 8080
CMD ["/app/comeandfind"]
