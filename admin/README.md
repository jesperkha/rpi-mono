# RPi Admin

A simple, mobile-friendly dashboard for managing Docker containers on a Raspberry Pi.

## Features

- View all Docker containers and their status
- Start/stop containers with one tap
- Password protected (SHA256)
- Dark themed, mobile-first UI

## Setup

1. Generate a password hash and put it in the env:

Generate a password hash:

```sh
./hashpass.sh
```

Create a `.env` file and

```sh
cenv fix
```

2. Run the server:

```sh
go run ./cmd
```

## Docker

```sh

```

