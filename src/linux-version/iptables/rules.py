import os
import subprocess
from config import setting
from utils.logger import RecordLog


class IptablesManager:
    """Manages system firewall rules using iptables to enforce a strict Tor-only policy."""
    
    def __init__(self):
        """Initializes the IptablesManager."""

        iptables_dir = os.path.dirname(os.path.abspath(__file__))
        self.logger = RecordLog(self.__class__.__name__).get_logger()

        self.tor_user = setting.TOR_USER
        self.dns_port = setting.TOR_DNS_PORT
        self.trans_port = setting.TOR_TRANSPARENT_PORT
        self.non_tor_networks = setting.NON_TOR_NETWORKS
        self.tor_uid = self._get_user_uid(self.tor_user)
        self.backup_path_v4 = os.path.join(iptables_dir, "iptables.v4.bak")

    def _run_command(self, command: list) -> bool:
        """Executes a given shell command and logs its outcome."""
        
        self.logger.debug(f"Executing: {' '.join(command)}")    
        try:
            subprocess.run(command, capture_output=True, text=True, check=True, encoding='utf-8')
            return True
        except subprocess.CalledProcessError as e:
            self.logger.error(f"Failed: {' '.join(command)}. Stderr: {e.stderr.strip()}")
            return False
        except FileNotFoundError:
            self.logger.critical(f"Command not found: {command[0]}. Is it installed?")
            return False

    def _get_user_uid(self, username: str) -> str | None:
        """Retrieves the UID for a given system username."""
        
        self.logger.info(f"Fetching UID for user '{username}'...")
        try:
            return subprocess.check_output(['id', '-u', username]).strip().decode()
        except subprocess.CalledProcessError:
            self.logger.critical(f"User '{username}' not found.")
            return None

    def backup_rules(self) -> bool:
        """Saves the current IPv4 iptables rules to a backup file."""
        
        self.logger.info("Backing up IPv4 firewall rules...")
        try:
            with open(self.backup_path_v4, 'w', encoding='utf-8') as f_v4:
                subprocess.run(["iptables-save"], stdout=f_v4, check=True, text=True)
            self.logger.info("Backup successful.")
            return True
        except (IOError, subprocess.CalledProcessError) as e:
            self.logger.error(f"Failed to backup firewall rules: {e}")
            return False
    
    def restore_rules(self) -> bool:
        """Restores firewall rules from the backup file or resets to a default state."""
        
        self.logger.warning("Restoring firewall rules from backup...")

        self.logger.info("Flushing all existing IPv4 rules and chains...")
        self._run_command(["iptables", "-t", "nat", "-F"]) 
        self._run_command(["iptables", "-F"]) 

        self._run_command(["ip6tables", "-P", "INPUT", "ACCEPT"])
        self._run_command(["ip6tables", "-P", "FORWARD", "ACCEPT"])
        self._run_command(["ip6tables", "-P", "OUTPUT", "ACCEPT"])
        self._run_command(["ip6tables", "-F"])
        
        if os.path.exists(self.backup_path_v4):
            try:
                with open(self.backup_path_v4, 'r', encoding='utf-8') as f:
                    subprocess.run(["iptables-restore"], stdin=f, check=True, text=True)
                os.remove(self.backup_path_v4)
                self.logger.info("Successfully restored IPv4 rules from backup.")
            except (IOError, subprocess.CalledProcessError) as e:
                self.logger.error(f"Could not restore IPv4 rules: {e}")
        else:
            self.logger.info("No IPv4 backup file found. Resetting to default ACCEPT policies.")
            self._run_command(["iptables", "-P", "INPUT", "ACCEPT"])
            self._run_command(["iptables", "-P", "FORWARD", "ACCEPT"])
            self._run_command(["iptables", "-P", "OUTPUT", "ACCEPT"])
        return True

    def apply_tor_rules(self) -> bool:
        """Applies a strict set of firewall rules to force all traffic through Tor."""
        
        self.logger.info("Applying strict Tor-only firewall rules...")
        if not self.tor_uid:
            return False
        if not self.backup_rules():
            return False
    
        self._run_command(["iptables", "-F"])
        self._run_command(["iptables", "-t", "nat", "-F"])
        self._run_command(["iptables", "-t", "nat", "-A", "OUTPUT", "-m", "owner", "--uid-owner", self.tor_uid,"-j", "RETURN"])
        self._run_command(["iptables", "-t", "nat", "-A", "OUTPUT", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", str(self.dns_port)])
        self._run_command(["iptables", "-t", "nat", "-A", "PREROUTING", "-i", "lo", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", str(self.dns_port)])
    
        for net in self.non_tor_networks.split() + ["127.0.0.0/8"]:
            self._run_command(["iptables", "-t", "nat", "-A", "OUTPUT", "-d", net, "-j", "RETURN"])
    
        self._run_command(["iptables", "-t", "nat", "-A", "OUTPUT", "-p", "tcp", "--syn", "-j", "REDIRECT", "--to-ports", str(self.trans_port)])
        self._run_command(["iptables", "-F", "OUTPUT"])
        self._run_command(["iptables", "-A", "OUTPUT", "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT"])
    
        for net in self.non_tor_networks.split() + ["127.0.0.0/8"]:
            self._run_command(["iptables", "-A", "OUTPUT", "-d", net, "-j", "ACCEPT"])
    
        self._run_command(["iptables", "-A", "OUTPUT", "-m", "owner", "--uid-owner", self.tor_uid, "-j", "ACCEPT"])
        self._run_command(["iptables", "-A", "OUTPUT", "-j", "REJECT"])
        self._run_command(["ip6tables", "-P", "INPUT", "DROP"])
        self._run_command(["ip6tables", "-P", "FORWARD", "DROP"])
        self._run_command(["ip6tables", "-P", "OUTPUT", "DROP"])
        self._run_command(["ip6tables", "-F"])
    
        self.logger.info("✅ Strict Tor firewall rules applied. All traffic goes through Tor now.")
        return True
