FROM golang:1.23-alpine as builder

RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o nad2go .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Europe/Helsinki
WORKDIR /root/
COPY --from=builder /app/nad2go .
EXPOSE 8080
CMD ["./nad2go"]
