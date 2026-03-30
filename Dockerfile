FROM golang:1.24-alpine AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o audioRec .

FROM alpine:latest
RUN apk add --no-cache ffmpeg
WORKDIR /app
COPY --from=build-stage /app/audioRec /app/storeAllSongs.sh ./

#CMD [ "./audioRec", "1", "./audioFiles/Arctic Monkeys - Do I Wanna Know？ (Official Video).wav"]
CMD ["tail", "-f", "/dev/null"]
