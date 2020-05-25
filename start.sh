#!/bin/bash

# Start the db-to-ynab sync service with Docker.
# User may specify a version tag as the first argument, otherwise defaults to "latest"
VERSION=:${1:-latest}

echo "Starting db-to-ynab sync service on port 3000, version${VERSION}"
$(which docker) run -p 3000:3000 --env-file .env ohthehugemanatee/db-ynab-sync${VERSION}
