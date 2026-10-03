# Vinyl

Vinyl is a Shazam-like song recognition app. It records a short audio sample in
the browser, creates an audio fingerprint, and compares that fingerprint with
songs stored in a PostgreSQL database.

The project has a Go backend and a React frontend. The browser runs the Go
fingerprinting code through WebAssembly.

## Requirements

- [Go](https://go.dev/) 1.24.1 or later
- [Node.js](https://nodejs.org/) and npm
- [PostgreSQL](https://www.postgresql.org/)
- [FFmpeg](https://ffmpeg.org/)
- A browser with microphone access

## Set up the database

Create a PostgreSQL database, then create the required tables:

```sh
createdb vinyl
psql -d vinyl -f backend/db/db.sql
```

Copy the example environment file:

```sh
cp .env.example .env
```

Update `.env` with your PostgreSQL connection details. For the database created
above, set `DB_NAME=vinyl`.

```dotenv
DB_USER=postgres
DB_PASSWORD=password
DB_HOST=localhost
DB_PORT=5432
DB_NAME=vinyl
DB_SSLMODE=disable
```

The backend uses port `3000` by default. You can override it with `PORT` and set
the allowed frontend origin with `FRONTEND_ORIGIN`.

## Add songs

The backend's `store` mode fingerprints a WAV file and saves it to the database.
Name each file with this exact format:

```text
Song Name - Artist.wav
```

Place the file in `backend/audioFiles`, then run:

```sh
cd backend
go run . store "audioFiles/Song Name - Artist.wav"
```

Song names must be unique. To import every WAV file in `backend/audioFiles`, run
the included script from the `backend` directory:

```sh
./storeAllSongs.sh
```

## Run the app locally

Start the backend in `record` mode. This starts the API that receives
fingerprints from the browser:

```sh
cd backend
go run . record
```

In another terminal, configure and start the frontend:

```sh
cd frontend
cp .env.example .env
npm install
npm run dev
```

Set the frontend API URL to:

```dotenv
VITE_API_URL=http://localhost:3000/api
```

Open the local URL printed by Vite, allow microphone access, and play a song
that has already been added to the database.

## Backend modes

- `record` starts the HTTP API. The browser handles the microphone recording.
- `store <path>` fingerprints one WAV file and stores it in PostgreSQL.

Running `go run .` without a mode also starts the API.

## Rebuild the WebAssembly module

The repository includes a compiled WebAssembly module. Rebuild it if you change
the Go fingerprinting code:

```sh
cd backend
GOOS=js GOARCH=wasm go build -o ../frontend/public/wasm/main.wasm ./wasm
```

When you rebuild the WebAssembly module with a different Go version, also
replace `frontend/src/wasm/wasm_exec.js` with the copy supplied by that Go
version. Run this from the `backend` directory:

```sh
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ../frontend/src/wasm/wasm_exec.js
```

Fingerprint settings live in `backend/internal/audioconfig/config.go`. If you
change the sample rate, window size, or related settings, rebuild the WebAssembly
module and regenerate every stored fingerprint.
