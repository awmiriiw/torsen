# Torsen - Transparent Tor Network Manager

A lightweight, modular, and security-focused Linux network management tool built with **Go**, **Tor**, **iptables**, and **systemd**.

Torsen is designed to route system network traffic through the Tor anonymity network by managing transparent proxy rules, DNS configuration, firewall policies, and Tor services.

It provides automated connection management, Tor bootstrap monitoring, configuration backups, system recovery, and customizable Tor configuration templates through a simple command-line interface.

---

## Key Features

### 🔒 Transparent Tor Routing

- **System-Wide Tor Routing:** Redirects supported outgoing TCP traffic through the Tor network using transparent proxy rules.
- **Automatic Firewall Configuration:** Creates and manages dedicated iptables chains for Tor traffic redirection.
- **Traffic Filtering:** Restricts unauthorized outgoing connections while Tor routing is active.
- **IPv6 Leak Prevention:** Blocks IPv6 traffic during active Tor sessions to reduce potential traffic leaks.
- **Local Network Support:** Allows configurable private and local networks to bypass Tor routing.
- **Tor Process Exclusion:** Excludes the Tor service user from transparent redirection to prevent routing loops.

### 🌐 DNS Management

- **Automatic DNS Configuration:** Temporarily replaces system DNS settings to use Tor's DNS resolver.
- **DNS Redirection:** Redirects UDP DNS queries to the configured Tor DNSPort using iptables.
- **DNS Configuration Backup:** Preserves the original `resolv.conf` before applying changes.
- **Symbolic Link Support:** Detects and restores symbolic links used by system DNS managers.
- **DNS Service Management:** Integrates with systemd-resolved, dnsmasq, unbound, and NetworkManager.
- **Automatic DNS Restoration:** Restores the previous DNS configuration when disconnecting.

### 🛡️ Firewall & Network Protection

- **Dedicated iptables Chains:** Uses custom `TORSEN_OUTPUT`, `TORSEN_NAT`, and `TORSEN_PREROUTING` chains.
- **IPv4 Firewall Backup:** Saves existing IPv4 firewall rules before making changes.
- **IPv6 Firewall Backup:** Preserves IPv6 firewall rules for restoration after disconnecting.
- **Connection Filtering:** Uses iptables filtering rules to restrict outgoing network traffic.
- **Configurable Network Exceptions:** Supports custom networks that bypass Tor through environment configuration.
- **Firewall Restoration:** Attempts to restore previously saved firewall rules during disconnection and recovery.

### 🧅 Tor Service & Bootstrap Management

- **Automatic Tor Service Management:** Starts, restarts, and stops the Tor systemd service.
- **Bootstrap Monitoring:** Tracks Tor connection progress through the Tor ControlPort.
- **Connection Progress Logging:** Displays bootstrap percentage and connection status.
- **Configurable Bootstrap Timeout:** Allows customization of the maximum connection waiting time.
- **ControlPort Authentication:** Supports password-based authentication for communicating with Tor.
- **Automatic Password Hashing:** Generates hashed Tor ControlPort passwords using `tor --hash-password`.

### ⚙️ Tor Configuration & Templates

- **Embedded Tor Configuration:** Includes a default Tor configuration template embedded directly into the Go binary.
- **Custom Configuration Templates:** Supports persistent user-defined Tor configuration templates.
- **Interactive Template Editing:** Opens the configuration template using an available terminal text editor.
- **Runtime Configuration Rendering:** Automatically replaces configuration placeholders with environment values.
- **Configuration Validation:** Verifies generated Tor configurations before applying them.
- **Template Reset:** Restores the original embedded Tor configuration template.
- **Tor Configuration Backup:** Preserves the original system `torrc` before modification.

### 🔄 Connection Lifecycle & Recovery

- **Automated Connection Workflow:** Coordinates DNS configuration, Tor initialization, bootstrap monitoring, and firewall activation.
- **Automatic Cleanup:** Attempts to restore modified system settings when the connection is terminated.
- **Signal Handling:** Supports graceful disconnection through `SIGINT` and `SIGTERM`.
- **Persistent State Management:** Stores connection status and backup paths in a JSON state file.
- **Manual Recovery:** Provides a dedicated recovery command for restoring system settings.
- **Connection Status Monitoring:** Reports Tor service, DNS configuration, firewall chains, and saved connection state.

### 📝 Logging & Diagnostics

- **Structured Logging:** Provides timestamped logs with DEBUG, INFO, WARN, ERROR, and FATAL severity levels.
- **Colored Terminal Output:** Displays color-coded log messages for easier monitoring.
- **Persistent Log Files:** Stores application logs in a configurable system directory.
- **Dependency Verification:** Checks required system utilities, Tor user configuration, and essential directories.
- **Debug Logging:** Supports detailed command execution and diagnostic output.

---

## Requirements

Torsen is designed for Linux systems using **systemd** and **iptables**.

### Required Dependencies

- Go (compatible with the version specified in `go.mod`)
- Tor
- iptables
- iptables-save
- iptables-restore
- ip6tables
- ip6tables-save
- ip6tables-restore
- systemd
- bash

### Optional Dependencies

- obfs4proxy (Tor bridge support)
- NetworkManager / nmcli
- nano, vim, micro, or another terminal text editor

**Important:** Torsen requires root privileges because it modifies system firewall rules, DNS configuration, and Tor service settings.

---

## Get Start:

```
git clone https://github.com/awmiriiw/torsen.git

go mod download

go mod tidy

make build

make clean

make install

make uninstall


# The compiled binary will be available at:
# bin/torsen

# This installs Torsen to:
# /usr/local/bin/torsen

# The installation also creates the following directories:
# /var/lib/torsen
# /var/log/torsen
```

---

## Available Commands

| Command                      | Description                                                                       |
| :--------------------------- | :-------------------------------------------------------------------------------- |
| `sudo torsen connect`        | Starts Tor, configures DNS and firewall rules, and activates transparent routing. |
| `sudo torsen disconnect`     | Restores previous system settings using available backups.                        |
| `sudo torsen status`         | Displays the current Tor, DNS, firewall, and connection state.                    |
| `sudo torsen verify`         | Checks system dependencies and required configuration.                            |
| `sudo torsen recover`        | Attempts to recover the original network configuration.                           |
| `sudo torsen edit`           | Opens the custom Tor configuration template for editing.                          |
| `sudo torsen reset-template` | Restores the default Tor configuration template.                                  |

---

## License

This project is under the **GNU Affero General Public License v3.0 (AGPL-3.0)**.

See the [LICENSE](LICENSE) file for the complete license text.
