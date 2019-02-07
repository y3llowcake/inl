# inl: inotify loop
1. Run an arbitrary shell command. (e.g. build, test, scp, you-name-it)
2. Wait for a file-system event in the current working directory.
3. Goto 1.

I often have this running in one terminal pane, and my editor in another.

## Installation
`go get -u github.com/y3llowcake/inl`

## Usage
Simple:

`inl ls -la`

Don't run to completion (uses SIGKILL on event):

`inl -n echo zzz \&\& sleep 60`

Don't run to completion, wait 5 seconds before re-establishing watches. This is
may be useful when running a build step that touches the filesystem:

`inl -n -t 5s make \&\& ./long-lived-server`

Alternatively, only watch certain files:

`inl -i=".*\.go" go test ./...`

## A bit of history
The original version of this program was implemented as a shell script. This served me well for nearly a decade. Then I decided it was time for more features.

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
