#!/bin/sh
# Stop the weather display; run.sh then brings the Kindle UI back.
if killall kindle-weather 2> /dev/null; then
  echo "Kindle Weather stopping"
else
  echo "Kindle Weather is not running"
fi
