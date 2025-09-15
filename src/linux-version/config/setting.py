import logging


# ==================================
#     LOGGING SETTINGS
# ==================================
LOG_BACKUP_COUNT = 5
LOG_DIRECTORY = "logs"
LOG_FILENAME = "logs.log"
LOG_MAX_BYTES = 10 * 1024 * 1024

FILE_LOG_LEVEL = logging.INFO
GLOBAL_LOG_LEVEL = logging.DEBUG
CONSOLE_LOG_LEVEL = logging.DEBUG

# ==================================
#     TOR SETTINGS
# ==================================
TOR_BOOTSTRAP_TIMEOUT = 120
SYSTEM_TORRC_PATH = "/etc/tor/torrc"
TOR_LOG_FILE_PATH = "/var/log/tor/notices.log"
TOR_CONTROL_PORT = 9051
TOR_USER = "tor" 

# ==================================
#  IPTABLES & FIREWALL SETTINGS
# ==================================
TOR_DNS_IP = "127.0.0.1" 
TOR_DNS_PORT = 5353
TOR_TRANSPARENT_PORT = 9040
NON_TOR_NETWORKS = "192.168.0.0/16 10.0.0.0/8 172.16.0.0/12"

# ==================================
#     DNS SETTINGS
# ==================================
RESOLV_CONF_PATH = "/etc/resolv.conf"
DNS_SERVICE_NAME = "systemd-resolved"
RESOLV_CONF_BACKUP_FILENAME = "resolv.conf.torsen.bak"

