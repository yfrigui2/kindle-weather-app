#!/bin/sh
# Stop the weather display; run.sh then brings the Kindle UI back.
if killall pocketink 2> /dev/null; then
  echo "Pocket Ink stopping"
else
  echo "Pocket Ink is not running"
fi
