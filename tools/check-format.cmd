@ECHO off
REM  NOTE: This is meant to be run inside the repo root as ./tools/check-format.sh

echo Checking files for correct go formatting...

for /f %%i in ('gofmt -l .') do set WRONG_FORMAT=%%i

REM if gofmt found no files with wrong formatting, exit ok
if not defined WRONG_FORMAT (
    exit 0
) else (
    REM else print and fail
    echo Unformatted files found!
    echo %WRONG_FORMAT%
    echo Please format them using gofmt!
    exit 1
)
