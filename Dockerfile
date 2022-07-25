FROM golang:1.18

WORKDIR /app

COPY . .
RUN go build ./bin/main.go

# postgres
# CMD ["./main", "-ps"] 

# local
CMD ["./main"] 