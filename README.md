English | [فارسی](README.fa_IR.md)

# vWireguard - WireGuard VPN Management System

[![Go Report Card](https://goreportcard.com/badge/github.com/MmadF14/vwireguard)](https://goreportcard.com/report/github.com/MmadF14/vwireguard)
[![GoDoc](https://godoc.org/github.com/MmadF14/vwireguard?status.svg)](https://godoc.org/github.com/MmadF14/vwireguard)
[![License](https://img.shields.io/github/license/MmadF14/vwireguard)](LICENSE)

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

- Go 1.16 or higher
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
http://localhost:8080
```

3. Default credentials:
- Username: admin
- Password: admin

## 🔒 Security

- All passwords are hashed using bcrypt
- HTTPS support for secure communication
- Role-based access control
- Secure key storage and management
- Regular security updates

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
