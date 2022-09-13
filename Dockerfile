FROM golang:1.19-alpine as builder
WORKDIR /app
COPY go.mod go.sum ./
COPY submodules/ ./submodules
RUN go mod download
COPY . .
RUN go build -o phantomias .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/phantomias ./
CMD ["/app/phantomias"]
EXPOSE 3000