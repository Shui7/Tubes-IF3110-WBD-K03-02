#!/bin/sh
set -e

echo "Initializing sample databases"

for f in /sample/*.sql; do
    [ -e "$f" ] || continue
    fname=$(basename "$f")
    dbname=$(echo "$fname" | sed 's/\.sql$//' | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_')
    echo "Processing sample file: $fname (target db: $dbname)..."

    psql -v ON_ERROR_STOP=0 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -c "CREATE DATABASE \"$dbname\";" || true

    echo "Importing $fname into database $dbname..."
    psql -v ON_ERROR_STOP=0 --quiet --username "$POSTGRES_USER" --dbname "$dbname" -f "$f" || true
    echo "Finished processing $dbname"
done

echo "Sample databases initialized"
