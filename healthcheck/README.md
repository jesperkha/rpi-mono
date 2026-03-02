# Healthcheck

A minimal, mobile-first system health dashboard built with Go. Designed to be installed as a PWA on your phone for quick server monitoring.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)

## Features

- **System uptime**
- **CPU load** — 1/5/15 min averages per core
- **Disk usage** — used / total with progress bar
- **RAM usage** — used / total with buffer & cache breakdown
- **Docker containers** — live status of all containers

## Quick Start

```sh
docker compose up -d
```

The dashboard is available at `http://localhost:8080`.

## Configuration

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `PORT`   | `:8080` | Server port |

## PWA

The app includes a web manifest and icons, so you can add it to your mobile home screen for a native app-like experience.
