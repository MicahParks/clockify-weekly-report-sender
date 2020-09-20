FROM golang AS builder
COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/app -a -installsuffix cgo -ldflags "-s -w"

FROM alpine
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/app /app
