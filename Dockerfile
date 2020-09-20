FROM golang AS builder
COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/app -a -installsuffix cgo -ldflags "-s -w"

FROM alpine
COPY --from=builder /app/app /app
