import os
import shutil
import subprocess
from config import setting
from utils.logger import RecordLog


class DNSManager:
    """Manages system DNS settings to route queries through Tor."""

    def __init__(self):
        """Initializes the DNSManager."""
        
        self._was_symlink = False
        self._symlink_target = None

        self.dns_service = setting.DNS_SERVICE_NAME
        self.resolv_conf_path = setting.RESOLV_CONF_PATH
        dns_module_dir = os.path.dirname(os.path.abspath(__file__))
        self.logger = RecordLog(self.__class__.__name__).get_logger()
        self.backup_path = os.path.join(dns_module_dir, setting.RESOLV_CONF_BACKUP_FILENAME)
        
    def _run_system_command(self, command: list, description: str):
        """Executes a system command and logs the outcome."""
        
        try:
            subprocess.run(command, capture_output=True, text=True, check=True, encoding='utf-8')
            self.logger.info(f"Successfully {description}.")
            return True
        except FileNotFoundError:
            self.logger.info(f"Command '{command[0]}' not found, skipping.")
            return True 
        except subprocess.CalledProcessError as e:
            self.logger.warning(f"Failed to {description}: {e.stderr.strip()}")
            return False

    def take_control(self) -> bool:
        """Takes control of the system's DNS settings to route through Tor."""
        
        self.logger.info("Taking control of system DNS...")
        self._run_system_command(["systemctl", "stop", self.dns_service], f"stopped {self.dns_service}")

        self.logger.info(f"Backing up '{self.resolv_conf_path}'...")
        if os.path.islink(self.resolv_conf_path):
            self._was_symlink = True
            self._symlink_target = os.readlink(self.resolv_conf_path)
            self.logger.info(f"Detected symlink, target is '{self._symlink_target}'.")
        
        try:
            if os.path.exists(self.resolv_conf_path):
                 shutil.copy(self.resolv_conf_path, self.backup_path, follow_symlinks=True)
                 self.logger.info(f"Content backed up to '{self.backup_path}'.")
        except (IOError, PermissionError) as e:
            self.logger.error(f"Could not back up DNS configuration: {e}")
            self._run_system_command(["systemctl", "start", self.dns_service], f"re-started {self.dns_service}")
            return False

        self.logger.info("Setting DNS to route through Tor...")
        try:
            if os.path.lexists(self.resolv_conf_path): os.remove(self.resolv_conf_path)
            with open(self.resolv_conf_path, "w", encoding='utf-8') as f:
                f.write(f"# Managed by Torsen\nnameserver {setting.TOR_DNS_IP}\n")
            return True
        except (IOError, PermissionError) as e:
            self.logger.error(f"Could not write to '{self.resolv_conf_path}': {e}")
            self.release_control()
            return False

    def release_control(self) -> bool:
        """Restores the original system DNS configuration."""
        
        self.logger.info("Releasing control of system DNS...")
        try:
            if os.path.lexists(self.resolv_conf_path):
                os.remove(self.resolv_conf_path)

            if self._was_symlink and self._symlink_target:
                os.symlink(self._symlink_target, self.resolv_conf_path)
                self.logger.info(f"Restored symlink -> '{self._symlink_target}'.")
            elif os.path.exists(self.backup_path):
                shutil.move(self.backup_path, self.resolv_conf_path)
                self.logger.info("Restored original file.")

        except (IOError, PermissionError) as e:
            self.logger.critical(f"Failed to restore DNS config: {e}. Manual intervention required!")

        self._run_system_command(["systemctl", "restart", self.dns_service], f"restarted {self.dns_service}")
        self._run_system_command(["nmcli", "general", "reload"], "reloaded NetworkManager")
        
        self.logger.info("DNS control released.")
        return True
