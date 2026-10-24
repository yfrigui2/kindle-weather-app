#!/bin/sh
# Pocket Ink weather display: pauses the Kindle UI, draws the weather until a 3-second hold
# on the status bar (or stop.sh), then brings the Kindle UI back.
DIR=/mnt/us/extensions/pocketink
LOG=$DIR/pocketink.log
if [ -f "$LOG" ] && [ "$(wc -c < "$LOG")" -gt 262144 ]; then mv "$LOG" "$LOG.old"; fi
exec >> "$LOG" 2>&1

echo "$(date) run.sh: starting"
lipc-set-prop com.lab126.powerd preventScreenSaver 1
stop lab126_gui
sleep 2

"$DIR/bin/pocketink" run -config "$DIR/pocketink.json"
echo "$(date) run.sh: display exited ($?), restoring the Kindle UI"

# The goodbye screen stays up until the Kindle UI paints over it.
lipc-set-prop com.lab126.powerd preventScreenSaver 0
start lab126_gui
