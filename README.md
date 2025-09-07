# Torsen

Torsen is a powerful command-line tool for Linux designed to route all your system's internet traffic through the Tor network. It provides a full-system transparent proxy, ensuring your applications are automatically anonymized without needing individual configuration.

## 🔎 About

In today's digital world, privacy is paramount. Torsen was created to simplify the complex process of achieving system-wide anonymity. Manually configuring `iptables`, DNS settings, and the Tor service can be error-prone and can lead to dangerous leaks.

Torsen automates this entire process with two simple commands: `connect` and `disconnect`. When you connect, it intelligently backs up your existing network settings, modifies system files (`/etc/tor/torrc`, `/etc/resolv.conf`), and applies a strict set of firewall rules to prevent any non-Tor traffic from leaving your machine. When you disconnect, it safely restores your system to its original state, leaving no trace behind.

This tool is perfect for privacy-conscious users, security researchers, and anyone who wants to ensure their digital footprint is protected.

## 🛠️ Features

- **System-Wide Anonymity**: Transparently forces all internet traffic from every application through the Tor network.
- **DNS Leak Protection**: Hijacks system DNS requests and routes them through Tor's secure DNS resolver, preventing your ISP from logging your browsing activity.
- **Strict Firewall Rules**: Implements robust `iptables` rules to block any direct connection, ensuring that if the Tor connection fails, your real IP address is not exposed.
- **Fully Automated**: Automatically manages the backup, modification, and restoration of critical network configuration files.
- **Safe & Reversible**: A built-in cleanup function ensures that your original system settings are perfectly restored upon disconnection (using `Ctrl+C` or the `disconnect` command).
- **Connection Monitoring**: Actively checks the Tor bootstrap process to confirm a successful connection before enforcing firewall rules.
- **Colored Logging**: Provides clear, color-coded console output and detailed log files for easy status tracking and debugging.

## 📂 Project Structure

The project is organized into modular components for clarity and maintainability.

```
.
├── default-torrc.txt
├── README.md
├── requirements.txt
└── src
    ├── linux-version
    │   ├── config
    │   │   ├── bash.sh
    │   │   ├── __init__.py
    │   │   ├── setting.py
    │   │   └── tor-config.template
    │   ├── dns
    │   │   ├── dns_manager.py
    │   │   └── __init__.py
    │   ├── iptables
    │   │   ├── __init__.py
    │   │   └── rules.py
    │   ├── logs
    │   │   └── logs.log
    │   ├── main.py
    │   ├── tor
    │   │   ├── controller.py
    │   │   └── __init__.py
    │   └── utils
    │       └── logger.py
    └── mobile-version
```

## 🚀 Getting Started

Follow these instructions to get Torsen up and running on your system.

### Prerequisites

Before you begin, ensure you have the following installed:

- **Root Access**: The script must be run with `sudo` privileges.
- **Python 3**: Version 3.6 or higher.
- **Tor**: The Tor service and obfs4proxy must be installed on your system.

  ```bash
  # For ArchLinux/Arch-based systems
  sudo pacman -Syu
  sudo pacman -S tor
  sudo pacman -S obfs4proxy
  ```

  ```bash
  # For ubuntu/debian-based systems
  sudo apt install update
  sudo apt install tor
  sudo apt install obfs4proxy
  ```

- **iptables**: Standard on most Linux distributions.
- **Python Libraries**: The script depends on `stem` and `coloredlogs`.

### Installation & Usage

1.  **Clone the repository (or download the source code):**

    ```bash
    git clone "https://github.com/awmiriiw/torsen.git"
    cd torsen
    ```

2.  **Install the required Python packages:**

    run:

    ```bash
    sudo pip3 install -r requirements.txt --break-system-packages
    ```

3.  **Run the script:**
    Navigate to the `src/linux-version` directory to run the main script.
    - **To start the secure connection:**

      ```bash
      sudo python3 main.py connect
      ```

      The script will now configure your system. Once connected, it will say "Connection is active."

    - **To stop the connection and restore settings:**
      Simply press `Ctrl+C` in the terminal where the script is running.
      Alternatively, if you need to disconnect from another terminal, you can run:
      ```bash
      sudo python3 main.py disconnect
      ```

## 📝 License

This project is open-source and available under the [GNU AGPLv3 License](https://opensource.org/license/agpl-v3).

**Thanks for visiting! ☕**
