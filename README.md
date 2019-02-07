# inl: inotify loop

Premise:
1. Run an arbitrary shell command.
2. Wait for a file-system event in the current working directory.
3. Goto 1.

## Installation
`go get -u github.com/y3llowcake/inl`

## Usage
Simple:

`inl echo hi`

Don't run to completion, use SIGKILL instead:

`inl -n echo zzz \&\& sleep 60`

Don't run to completion, wait 5 seconds before re-establishing watches:

`inl -n -t 5s go build \&\& ./binary`

Only watch certain files:

`inl -i=".*\.go" go test ./...`

## A bit of history
The original version of this program was implemented as a shell script. This served me well for over 5 years. Then I decided it was time for more features.

```
while true; do
  clear
  echo $@
  /bin/bash -c "$@"
  inotifywait -r -e CREATE -e ATTRIB ./ --exclude '(^.+\.sw[px]$)|(^4913$)'
  sleep 0.1
  if [[ $? != 0 ]]; then
    exit $?
  fi
done
```
