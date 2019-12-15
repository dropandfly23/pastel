### Pastel

## Description

A tools for Indigo to lock env.

## Features

### Server

Run the pastel app that allows:
 - locking/unlocking of environment

### Tasks

Some tasks that can run with cron.

## How to...

### Build the project

```sh
GOOS=linux GOARCH=amd64 go build -o pastel .
```

### Build the image

```sh
docker build -t registry.forge.orange-labs.fr/indigo/tools/pastel .
```

### Run the image

```sh
docker run -d -e PASTEL_GITLAB_APP_ID=gitlab-app-id -e PASTEL_GITLAB_SECRET=gitlab-secret -p 8080:8080 registry.forge.orange-labs.fr/indigo/tools/pastel
```