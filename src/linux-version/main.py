import os
import sys
import time
import signal
import subprocess
from config import setting
from utils.logger import RecordLog
from stem.control import Controller
from tor.controller import TorrcManager
from dns.dns_manager import DNSManager
from iptables.rules import IptablesManager

is_connected = False
dns_manager = DNSManager()
torrc_manager = TorrcManager()
connection_in_progress = False
iptables_manager = IptablesManager()
logger = RecordLog(__name__).get_logger()


def run_system_command(command: list, description: str, check_result: bool = True) -> bool:
    """Executes a system command and logs its description and outcome."""
    
    logger.info(description)
    try:
        result = subprocess.run(command, capture_output=True, text=True, check=check_result, encoding='utf-8')
        if result.stdout: logger.debug(f"Stdout: {result.stdout.strip()}")
        return True
    except subprocess.CalledProcessError as e:
        logger.critical(f"Failed to execute '{' '.join(command)}'. Stderr: {e.stderr.strip()}")
        return False
    except FileNotFoundError:
        logger.critical(f"Command '{command[0]}' not found. Is it in your PATH?")
        return False

def cleanup(signum=None, frame=None):
    """Restores all system configurations to their original state and exits."""
    
    global is_connected, connection_in_progress
    if not is_connected and not connection_in_progress:
        if signum is not None:
             sys.exit(0)
        return

    if is_connected: logger.warning("--- Cleanup Process Initiated ---")
    connection_in_progress = False

    script_path = os.path.join(os.path.dirname(__file__), 'config', 'bash.sh')

    logger.info("Step 1: Stopping Tor service via bash.sh...")
    run_system_command([script_path, "stop"], "Executing bash shutdown script for Tor service")

    logger.info("Step 2: Restoring firewall rules...")
    if not iptables_manager.restore_rules():
        logger.critical("Failed to restore iptables. Manual intervention may be required!")

    logger.info("Step 3: Restoring Tor configuration...")
    if not torrc_manager.restore_torrc(setting.SYSTEM_TORRC_PATH):
        logger.critical("Failed to restore original torrc. Manual intervention may be required!")
    
    logger.info("Step 4: Releasing control of DNS...")
    if not dns_manager.release_control():
        logger.critical("Failed to restore DNS. Manual intervention may be required!")

    logger.info("✅ System connectivity restored. Tor service is stopped.")
    is_connected = False
    sys.exit(0)

def wait_for_tor_bootstrap():
    """Waits for the Tor service to fully connect to the network."""
    
    logger.info("Connecting to Tor Control Port to check bootstrap status...")
    start_time = time.time()

    while time.time() - start_time < setting.TOR_BOOTSTRAP_TIMEOUT:
        try:
            with Controller.from_port(port=setting.TOR_CONTROL_PORT) as controller:
                controller.authenticate()
                bootstrap_status = controller.get_info("status/bootstrap-phase")
                
                if "PROGRESS=" in bootstrap_status:
                    progress = int(bootstrap_status.split("PROGRESS=")[1].split(" ")[0])
                    logger.info(f"Tor bootstrap progress: {progress}%")

                    if progress == 100:
                        logger.info("✅ Tor has successfully bootstrapped.")
                        return True
                else:
                    logger.debug(f"Tor status available, waiting for progress report. Status: {bootstrap_status}")

        except Exception as e:
            logger.debug(f"Could not connect to Tor Control Port yet, waiting... Error: {e}")

        time.sleep(2)

    logger.critical("Tor bootstrap timed out. Could not connect to the Tor network.")
    return False

def connect():
    """Orchestrates the entire process of establishing a secure, system-wide Tor connection."""
    
    global is_connected, connection_in_progress
    if os.geteuid() != 0:
        logger.critical("This script must be run as root.")
        sys.exit(1)

    connection_in_progress = True
    logger.info("--- Starting Secure Connection Process ---")

    if not dns_manager.take_control():
        logger.critical("Failed to take control of DNS. Aborting and cleaning up.")
        cleanup()

    if not torrc_manager.backup_torrc(setting.SYSTEM_TORRC_PATH) or \
       not torrc_manager.apply_template(setting.SYSTEM_TORRC_PATH):
        logger.critical("Failed to configure torrc. Aborting and cleaning up.")
        cleanup()

    script_path = os.path.join(os.path.dirname(__file__), 'config', 'bash.sh')
    run_system_command([script_path, "start"], "Executing bash startup script for Tor service", check_result=False)

    if not wait_for_tor_bootstrap():
        cleanup()

    if not iptables_manager.apply_tor_rules():
        logger.critical("Failed to apply firewall rules. Aborting and cleaning up.")
        cleanup()

    is_connected = True
    connection_in_progress = False
    logger.info("✅ System traffic is now securely routed through Tor.")
    print("\nConnection is active. Press Ctrl+C to disconnect and restore settings.")

    try:
        while True: time.sleep(1)
    except KeyboardInterrupt:
        cleanup()

def main():
    """The main entry point for the script."""
    
    signal.signal(signal.SIGINT, cleanup)
    signal.signal(signal.SIGTERM, cleanup)

    if len(sys.argv) != 2 or sys.argv[1] not in ["connect", "disconnect"]:
        print(f"Usage: sudo python3 {sys.argv[0]} [connect|disconnect]")
        sys.exit(1)

    command = sys.argv[1]

    if command == "connect":
        connect()
    elif command == "disconnect":
        logger.info("Disconnect command received.")
        global is_connected, connection_in_progress
        is_connected = True
        connection_in_progress = True
        cleanup()

if __name__ == "__main__":
    main()