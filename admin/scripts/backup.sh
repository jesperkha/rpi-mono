#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Configuration
SOURCE_DIR="../recipes/data"
BACKUP_DIR="../recipes/backups"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
ARCHIVE_NAME="data_backup_${TIMESTAMP}.tar.gz"

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Create compressed archive
tar -czf "${BACKUP_DIR}/${ARCHIVE_NAME}" "$SOURCE_DIR"

echo "Backup created: ${BACKUP_DIR}/${ARCHIVE_NAME}"