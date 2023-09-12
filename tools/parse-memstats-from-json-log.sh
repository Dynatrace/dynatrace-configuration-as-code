#!/bin/sh

# This util assumes that you have created a structured JSON log containing entries created by memory.LogMemStats calls.
# Call the script as ./parse-memstats-from-json-log.sh <path to your logfile>.
# The script will copy the logfile over to a folder named based on it's filename (which if you didn't rename the logfile is it's timestamp).
# It will then parse the structured log, creating an individual file for the memstats and sampling that down to minute and 10 sec resolutions.
# For each memstat file it creates a JSON as well as CSV file - which can be useful to create visual representations of your findings.


if [ $# -eq 0 ];
then
  echo "$0: Missing argument - call as ./parse-memstats-from-json-log.sh <path to your logfile>"
  exit 1
elif [ $# -gt 1 ];
then
  echo "$0: Too many arguments: $@ - call as ./parse-memstats-from-json-log.sh <path to your logfile>"
  exit 1
fi

path="$1"
file=$(echo "$path" | sed 's/.logs\///g')
folder=$(echo "$file" | sed 's/.log//g')

echo "Copying $file to $folder"
mkdir "$folder"
cp "$path" "$folder/log.json"

echo "Extracting mem stats into $folder/memstats.json..."
cat "$folder/log.json" | jq 'select( .msg | contains("MEMSTATS"))' > "$folder/memstats.json"

echo "Sampling mem stats by timestamp into $folder/memstats.sampled.json..."
cat "$folder/memstats.json" | jq -r -s 'group_by(.ts) | map(max_by(.heapAllocByte))' > "$folder/memstats.sampled.json"

echo "Sampling mem stats by minute..."
cat "$folder/memstats.json" | jq -r -s '.[] | .["ts"] = (.ts | sub(":[0-9][0-9]\\+[0-9][0-9]:[0-9][0-9]"; ""; "g"))' > "$folder/memstats.by-min.json"
cat "$folder/memstats.by-min.json" | jq -r -s 'group_by(.ts) | map(max_by(.heapAllocByte))' > "$folder/memstats.by-min.sampled.json"

echo "Sampling mem stats by 10 sec..."
cat "$folder/memstats.json" | jq -r -s '.[] | .["ts"] = (.ts | sub("[0-9]\\+[0-9][0-9]:[0-9][0-9]"; ""; "g"))' > "$folder/memstats.by-ten-sec.json"
cat "$folder/memstats.by-ten-sec.json" | jq -r -s 'group_by(.ts) | map(max_by(.heapAllocByte))' > "$folder/memstats.by-ten-sec.sampled.json"

echo "Creating mem stats CSV for sampled JSONs..."
cat "$folder/memstats.sampled.json" | jq -r '(map(keys) | add | unique) as $cols | map(. as $row | $cols | map($row[.])) as $rows | $cols, $rows[] | @csv' > "$folder/memstats.sampled.csv"
cat "$folder/memstats.by-min.sampled.json" | jq -r '(map(keys) | add | unique) as $cols | map(. as $row | $cols | map($row[.])) as $rows | $cols, $rows[] | @csv' > "$folder/memstats.by-min.sampled.csv"
cat "$folder/memstats.by-ten-sec.sampled.json" | jq -r '(map(keys) | add | unique) as $cols | map(. as $row | $cols | map($row[.])) as $rows | $cols, $rows[] | @csv' > "$folder/memstats.by-ten-sec.sampled.csv"


maxHeap=$(cat "$folder/memstats.json" | jq -r 'select(.heapAlloc | contains("GB")) | .heapAlloc | sub("(?<x>.*)\\s.*"; "\(.x)"; "g") | tonumber' | jq -s 'max')
echo "Max Heap Memory used $maxHeap GB"
echo "Max Heap Memory used $maxHeap GB" > "$folder/maxheap.txt"
