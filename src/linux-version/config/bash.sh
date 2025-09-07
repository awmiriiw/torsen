#!/bin/bash
set -e


start_commands() {
    echo "[INFO] bash.sh: Running startup commands..."
    echo "[INFO] bash.sh: Restarting Tor service to apply new configuration..."

    systemctl start tor

    echo "[INFO] bash.sh: Startup commands finished successfully."
}
stop_commands() {
    echo "[INFO] bash.sh: Running shutdown commands..."
    echo "[INFO] bash.sh: Stopping Tor service..."

    systemctl stop tor

    echo "[INFO] bash.sh: Shutdown commands finished successfully."
}


case "$1" in
    start)
        start_commands
        ;;
    stop)
        stop_commands
        ;;
    *)
        echo "Error: Invalid argument. Usage: $0 {start|stop}" >&2
        exit 1
        ;;
esac

exit 0
