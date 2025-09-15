import os
import shutil
from utils.logger import RecordLog


class TorrcManager:
    """Manages the system's torrc configuration file."""

    def __init__(self):
        """Initializes the TorrcManager."""

        self.logger = RecordLog(self.__class__.__name__).get_logger()
        self.backup_path = os.path.join(os.path.dirname(__file__), '..', 'tor', 'torrc.bak')
        self.template_path = os.path.join(os.path.dirname(__file__), '..', 'config', 'tor-config.template')

    def backup_torrc(self, system_torrc_path: str) -> bool:
        """Backs up the system's current torrc file."""
        
        self.logger.info(f"Backing up '{system_torrc_path}' to '{self.backup_path}'")
        try:
            if not os.path.exists(system_torrc_path):
                self.logger.warning("Original torrc not found. Creating a blank backup.")
                open(self.backup_path, 'w').close()
            else:
                shutil.copy2(system_torrc_path, self.backup_path)
            self.logger.info("Backup successful.")
            return True
        except (PermissionError, IOError) as e:
            self.logger.error(f"Error during backup: {e}", exc_info=True)
            return False

    def apply_template(self, system_torrc_path: str) -> bool:
        """Replaces the system torrc file with the custom template."""
        
        self.logger.info(f"Replacing '{system_torrc_path}' with template '{self.template_path}'")
        try:
            if not os.path.exists(self.template_path):
                self.logger.error(f"Template file not found at '{self.template_path}'.")
                return False
            
            shutil.copy(self.template_path, system_torrc_path)
            self.logger.info("Template successfully replaced the main torrc file.")
            return True
        except (PermissionError, IOError) as e:
            self.logger.error(f"Error applying template: {e}", exc_info=True)
            return False

    def restore_torrc(self, system_torrc_path: str) -> bool:
        """Restores the original torrc file from the backup."""
        
        self.logger.info(f"Restoring original torrc from '{self.backup_path}'...")
        try:
            if not os.path.exists(self.backup_path):
                self.logger.error("Backup file not found for restore.")
                return False
            
            shutil.move(self.backup_path, system_torrc_path)
            self.logger.info("Successfully restored original torrc file.")
            return True
        except (PermissionError, IOError) as e:
            self.logger.error(f"Error restoring torrc: {e}", exc_info=True)
            return False