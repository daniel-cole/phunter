#!/bin/bash -e

# Inspiration from https://gist.github.com/brendangregg/1abcfeef9155ac526197f6f0abdd86bf

container_pid=$1
container_pid_ns=$(lsns -no NS -t pid -p "${container_pid}")
found=false
for UUID in $(docker ps -q); do
  pid=$(docker inspect -f '{{.State.Pid}}' "$UUID")
  name=$(docker inspect -f '{{.Name}}' "$UUID")
  name=${name#/}

  pidns=$(stat --format="%N" /proc/"${pid}"/ns/pid)
  pidns=${pidns#*[}
  pidns=${pidns%]*}

  if [[ "${container_pid_ns}" -eq "${pidns}" ]]; then
    echo "${name}"
    found=true
    break
  fi
done

if ! $found; then
  echo "unable to find container for pid: ${container_pid}"
fi
