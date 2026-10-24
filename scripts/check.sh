#!/bin/sh
# Render the design system's sample data and compare it with the Chrome reference renders.
set -e
cd "$(dirname "$0")/.."
mkdir -p build/check
(cd src && go build -o ../build/pocketink-mac .)
for s in main:WeatherScreen days:DaysScreen; do
  n=${s%%:*}; r=${s##*:}
  build/pocketink-mac render -sample -screen $n -o build/check/$n.png
  magick reference/$r.png -colorspace gray build/check/$n.png +append build/check/cmp-$n.png
  printf "%s differing pixels: " $n
  magick compare -metric AE -fuzz 10% reference/$r.png build/check/$n.png build/check/diff-$n.png 2>&1 | sed 's/.*(//;s/)//'; echo
done
