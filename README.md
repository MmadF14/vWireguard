English | [فارسی](README.fa_IR.md)

# vWireguard

vWireguard is a powerful, web-based management interface for WireGuard VPN, designed to provide an easy-to-use dashboard for managing your WireGuard server and clients. The dashboard has been completely redesigned with a modern, responsive TailwindCSS interface.

## 🌟 Key Features

- **Modern Dashboard**: A fully redesigned interface using TailwindCSS, inspired by modern design principles.
- **Bilingual Support**: Full support for English (LTR) and Persian (RTL) languages.
- **Dark/Light Mode**: Seamlessly switch between dark and light themes.
- **Responsive Design**: Optimized for all devices, from desktops to mobile phones.
- **User Management**: Role-based access control (Admin, Manager, User).
- **Client Management**: Easily add, edit, and delete WireGuard clients. Generate QR codes and configuration files.
- **Server Configuration**: Manage WireGuard interface settings directly from the dashboard.
- **Real-time Monitoring**: Monitor system resources (CPU, RAM, Disk) and network traffic in real-time.
- **Wake-on-LAN**: Remotely wake up devices on your network.
- **System Utilities**: Built-in tools for system maintenance and troubleshooting.

## 🚀 Installation

### One-Line Install (Recommended)

Install vWireguard with a single command. The installer automatically downloads the latest pre-built binary, sets up the systemd service, and configures the environment.

```bash
bash <(curl -Ls https://raw.githubusercontent.com/MmadF14/vwireguard/master/install.sh)
```

**Requirements:**
- Linux (amd64 or arm64)
- Root access
- Systemd-based system
- Internet connection

### Manual Installation (Build from Source)

If you prefer to build from source or have specific requirements:

1.  **Install Prerequisites**:
    -   Go 1.21 or higher
    -   WireGuard (`wireguard`, `wireguard-tools`)
    -   Git

2.  **Clone the Repository**:
    ```bash
    git clone https://github.com/MmadF14/vwireguard.git
    cd vwireguard
    ```

3.  **Build the Application**:
    ```bash
    go build -o vwireguard
    ```

4.  **Run**:
    ```bash
    ./vwireguard
    ```
    (Note: For production, it's recommended to set up a systemd service.)

## 🛠️ Configuration

### Default Credentials
- **Username:** `admin`
- **Password:** `admin`

**Important**: Change the default password immediately after your first login!

### Accessing the Web Interface
By default, the dashboard is accessible at:
```
http://YOUR_SERVER_IP:5000
```

## 💻 Management CLI (`vwg`)

vWireguard includes a command-line utility `vwg` for managing the service. This is installed automatically with the one-line installer.

```bash
# Start the service
sudo vwg start

# Stop the service
sudo vwg stop

# Restart the service
sudo vwg restart

# Check service status
vwg status

# View logs (add -f to follow in real-time)
vwg log
vwg log -f

# Update to the latest version
sudo vwg update
```

## 🔒 Security

- **Password Hashing**: All user passwords are securely hashed using bcrypt.
- **Role-Based Access**: Granular control over user permissions.
- **Secure Key Management**: WireGuard keys are generated and stored securely.
- **HTTPS Support**: Recommended for production environments (can be configured via reverse proxy like Nginx or Caddy).

## 🆘 Troubleshooting

If you encounter issues, please refer to our [Troubleshooting Guide](docs/TROUBLESHOOT.md).

Common checks:
1.  Ensure WireGuard is installed and the kernel module is loaded.
2.  Check system logs using `vwg log`.
3.  Verify firewall settings allow traffic on the WireGuard port (default UDP 51820) and the dashboard port (TCP 5000).

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👥 Authors

- [MmadF14](https://github.com/MmadF14)

---

<div align="center">
  <img src="https://img.shields.io/github/stars/MmadF14/vwireguard?style=social" alt="GitHub Stars">
  <img src="https://img.shields.io/github/forks/MmadF14/vwireguard?style=social" alt="GitHub Forks">
  <img src="https://img.shields.io/github/watchers/MmadF14/vwireguard?style=social" alt="GitHub Watchers">
</div>
