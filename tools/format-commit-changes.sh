#!/bin/sh
FILES=$(git diff --name-only --cached | grep .go)
[ -z "$FILES" ] && exit 0

echo "Running go formatter on commit..."

WRONG_FORMAT=$(go fmt ${FILES})

# if gofmt found no files with wrong formatting, exit ok
[ -z "$WRONG_FORMAT" ] && exit 0

# else print and fail
echo >&2 "Wrongly formatted files found and corrected!\n\n$WRONG_FORMAT\n\nPlease change files added to your commit now!"
exit 1
