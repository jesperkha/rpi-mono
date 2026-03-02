# Recipes

A simple self-hosted recipe app built with Go and server-side rendered HTML. Store, browse, and create recipes through a clean web interface.

## Screenshots

<p>
  <img src=".github/screenshot.png" width="100%" />
</p>

## Features

- Browse all recipes with filtering by type and cook time
- View recipe details with ingredients and step-by-step instructions
- Create new recipes through a web form (password protected)
- JSON file storage

## Setup

### Prerequisites

- [Go 1.25+](https://go.dev/dl/) (for local development)
- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/) (for containerized setup)
- [cenv](https://github.com/echo-webkom/cenv)

### Password

Recipe creation is protected by a password. The app expects a SHA-256 hash of the password in the environment.

Generate a password hash:

```sh
./hashpass.sh
```

### Environment variables

```sh
cenv fix
```

Set `PASSWORD_HASH` in `.env` using the output from `./hashpass.sh`.

## Run locally

```sh
go run ./cmd/main.go
```

The app will be available at [http://localhost:8080](http://localhost:8080).

## Run with Docker Compose

```sh
docker compose up --build -d
```

The app will be available at [http://localhost:8080](http://localhost:8080).

Recipe data is stored in a local folder (`data`) and persists across container restarts.

## License

[MIT](LICENSE)
