#!/bin/sh
# Start the weather display in the background (from KUAL or SSH).
DIR=/mnt/us/extensions/kindle-weather
if [ ! -f "$DIR/kindle-weather.json" ]; then
  echo "No kindle-weather.json: copy kindle-weather.example.json to kindle-weather.json and set your location"
  exit 1
fi
if pidof kindle-weather > /dev/null; then
  echo "Kindle Weather is already running"
  exit 0
fi
setsid sh "$DIR/bin/run.sh" < /dev/null > /dev/null 2>&1 &
echo "Kindle Weather starting"
