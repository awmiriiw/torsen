#!/bin/bash
set -e

start_commands() {
  echo "[INFO] bash.sh: Restarting Tor service..."
  systemctl restart tor
  echo "[INFO] bash.sh: Tor service started."
}

stop_commands() {
  echo "[INFO] bash.sh: Stopping Tor service..."
  systemctl stop tor
  echo "[INFO] bash.sh: Tor service stopped."
}

case "$1" in
start)
  start_commands
  ;;
stop)
  stop_commands
  ;;
*)
  echo "Error: Usage: $0 {start|stop}" >&2
  exit 1
  ;;
esac

exit 0
