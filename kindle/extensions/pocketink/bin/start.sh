#!/bin/sh
# Start the weather display in the background (from KUAL or SSH).
DIR=/mnt/us/extensions/pocketink
if [ ! -f "$DIR/pocketink.json" ]; then
  echo "No pocketink.json: copy pocketink.example.json to pocketink.json and set your location"
  exit 1
fi
if pidof pocketink > /dev/null; then
  echo "Pocket Ink is already running"
  exit 0
fi
setsid sh "$DIR/bin/run.sh" < /dev/null > /dev/null 2>&1 &
echo "Pocket Ink starting"
