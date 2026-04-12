# rpi-mono

A monorepo of services I run on my Raspberry Pi. Each service is self-contained and managed via Docker Compose. The admin dashboard ties everything together.

## Custom apps

| App                          | Description                                                                                        |
| ---------------------------- | -------------------------------------------------------------------------------------------------- |
| [admin](./admin)             | Dashboard for system health monitoring, Docker container management, and triggering deploy actions. |
| [dagensbilde](./dagensbilde) | Mobile-first daily photo sharing app where users upload one photo per day and vote for a winner.   |
| [recipes](./recipes)         | Self-hosted recipe app for storing, browsing, and creating recipes.                                |

## Third-party services

| Service                      | Description                              |
| ---------------------------- | ---------------------------------------- |
| [convertx](./convertx)       | File format converter.                   |
| [flatnotes](./flatnotes)     | Flat-file, self-hosted note-taking app.  |
