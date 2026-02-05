English | [فارسی](README.fa_IR.md)

# vWireguard

vWireguard is a web-based management interface for WireGuard VPN, providing an easy-to-use dashboard for managing your WireGuard server and clients.

## Features

- Web-based management interface
- User authentication and authorization
- Client management (add, edit, delete)
- Server configuration management
- Real-time monitoring
- Wake-on-LAN support
- System utilities and monitoring
- Multi-language support (English and Persian)

## Installation

### One-Line Install (Recommended)

Install vWireguard with a single command. The installer automatically downloads the latest pre-built binary from GitHub releases - **no compilation required**.

```bash
bash <(curl -Ls https://raw.githubusercontent.com/MmadF14/vwireguard/master/install.sh)
```

**Requirements:**
- Linux (amd64 or arm64)
- Root access
- Systemd-based system
- Internet connection

The installer will:
- Detect your system architecture
- Download the latest release from GitHub
- Install to `/usr/local/vwireguard`
- Create and enable systemd service
- Install management CLI (`vwg` command)

### Manual Installation (Build from Source)

1. Install required packages:
```bash
sudo apt-get update
sudo apt-get install -y wireguard wireguard-tools golang-go git
```

2. Clone the repository:
```bash
sudo mkdir -p /opt/vwireguard
cd /opt/vwireguard
sudo git clone https://github.com/MmadF14/vwireguard.git .
```

3. Build the application:
```bash
sudo go build -o vwireguard
```

4. Create the systemd service:
```bash
sudo cat > /etc/systemd/system/vwireguard.service << EOL
[Unit]
Description=vWireguard Web Interface
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/vwireguard
ExecStart=/opt/vwireguard/vwireguard
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOL
```

5. Enable and start the service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable vwireguard
sudo systemctl start vwireguard
```

## Management CLI

After installation, you can use the `vwg` command to manage the service:

```bash
# Service management (requires root)
sudo vwg start      # Start the service
sudo vwg stop       # Stop the service
sudo vwg restart    # Restart the service
vwg status          # Check service status
vwg log             # View logs (last 50 lines)
vwg log 100         # View last 100 lines
vwg log -f          # Follow logs in real-time

# Update to latest release (requires root)
sudo vwg update     # Update to latest GitHub release

# Show help
vwg help
```

## Default Credentials

- **Username:** `admin`
- **Password:** `admin`

**Important**: Change the default password immediately after first login!

## Accessing the Web Interface

After installation, you can access the web interface at:
```
http://YOUR_SERVER_IP:8080
```

## Security Considerations

1. Change the default admin password immediately after installation
2. Configure your firewall to only allow access from trusted IP addresses
3. Use HTTPS in production environments
4. Regularly update the system and application

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## 📸 Screenshots

<div align="center">
  <h3>Dashboard</h3>
  <img src="assets/images/dashboard.png" alt="Dashboard" width="800"/>
  <p><em>Main dashboard showing system overview and statistics</em></p>
</div>

<div align="center">
  <h3>Client Management</h3>
  <img src="assets/images/client-management.png" alt="Client Management" width="800"/>
  <p><em>Client management interface with connection status and configuration options</em></p>
</div>

<div align="center">
  <h3>System Monitor</h3>
  <img src="assets/images/system-monitor.png" alt="System Monitor" width="800"/>
  <p><em>Real-time system monitoring with resource usage graphs</em></p>
</div>

## 🌟 Features

- 🔒 Secure WireGuard VPN management
- 👥 Multi-user support with role-based access control
- 🌐 Wake-on-LAN functionality for remote devices
- 📊 Real-time system monitoring
- 🔄 Automatic client configuration generation
- 📱 Responsive web interface
- 🌍 Bilingual support (English/Persian)
- 📝 Comprehensive logging system
- 🔧 System utilities and maintenance tools
- 🔐 Secure key management

## 🚀 Quick Start

1. Clone the repository:
```bash
git clone https://github.com/MmadF14/vwireguard.git
cd vwireguard
```

2. Install dependencies:
```bash
go mod download
```

3. Build the application:
```bash
go build
```

4. Run the application:
```bash
./vwireguard
```

## 📋 Prerequisites

- Go 1.21 or higher
- WireGuard installed on the server
- Linux-based system (Ubuntu recommended)
- Root privileges for system operations

## 🛠️ Configuration

1. Configure WireGuard:
```bash
wg-quick up wg0
```

2. Access the web interface:
```
http://localhost:5000
```

3. Default credentials:
- Username: admin
- Password: admin

4. To extend the "remember me" session duration, set the `SESSION_MAX_AGE` environment
   variable (in days). The default value is `7` days.

## 🔒 Security

- All passwords are hashed using bcrypt
- HTTPS support for secure communication
- Role-based access control
- Secure key storage and management
- Regular security updates

## 🆘 Troubleshooting

### V2Ray Tunnel Issues

If you encounter issues with V2Ray tunnels, especially the "incomplete tunnel configuration" error, please refer to our comprehensive troubleshooting guide:

📖 **[V2Ray Tunnel Troubleshooting Guide](docs/V2RAY_TROUBLESHOOTING.md)**

This guide covers:
- Common configuration errors and solutions
- Required fields for different V2Ray protocols
- Step-by-step configuration instructions
- Validation tips and best practices

### General Issues

For other issues, please check:
1. System logs: `journalctl -u vwireguard`
2. WireGuard status: `wg show`
3. Network connectivity
4. Firewall settings

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👥 Authors

- [MmadF14](https://github.com/MmadF14)

## 🙏 Acknowledgments

- WireGuard team for their excellent VPN solution
- Echo framework for the web framework
- All contributors and users of this project

---

<div align="center">
  <img src="https://img.shields.io/github/stars/MmadF14/vwireguard?style=social" alt="GitHub Stars">
  <img src="https://img.shields.io/github/forks/MmadF14/vwireguard?style=social" alt="GitHub Forks">
  <img src="https://img.shields.io/github/watchers/MmadF14/vwireguard?style=social" alt="GitHub Watchers">
</div>
