FROM golang:alpine AS builder
WORKDIR /src/app
COPY . .
RUN go build -o server
FROM alpine
WORKDIR /root/
RUN mkdir pictures
COPY --from=builder /src/app ./app
COPY --from=builder /src/app/templates ./templates
CMD ["./app/server"]
