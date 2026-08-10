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

> **Two things trip up almost every from-source build. Read this first.**
>
> 1. **You must run `./prepare_assets.sh` before `go build`.** The `assets/`
>    directory is git-ignored (only `.gitkeep` is committed) but `main.go`
>    embeds it with `//go:embed assets/*`. If you skip this step the build
>    either fails with `pattern assets/*: no matching files found`, or produces
>    a binary whose web UI has no CSS/JS at all. This is what the old
>    instructions were missing.
> 2. **Do not use `apt install golang-go`.** Debian/Ubuntu ship an old Go
>    (Ubuntu 22.04 ships 1.18); `go.mod` requires **Go 1.21+**, so the build
>    fails with `go.mod requires go >= 1.21`. Install Go from go.dev instead,
>    as shown below.

#### Step 1 — System packages

```bash
sudo apt-get update
sudo apt-get install -y git curl wireguard wireguard-tools
```

#### Step 2 — Go 1.21+ (from go.dev, not apt)

```bash
GO_VERSION=1.21.13
# use linux-arm64 instead of linux-amd64 on ARM servers (Oracle Ampere, etc.)
curl -fsSLO "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
rm "go${GO_VERSION}.linux-amd64.tar.gz"

# put Go on PATH permanently
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
export PATH=$PATH:/usr/local/go/bin

go version   # must print go1.21 or newer
```

#### Step 3 — Node.js + Yarn (needed only to build the web assets)

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs
sudo npm install -g yarn

node -v && yarn -v
```

#### Step 4 — Clone

```bash
sudo mkdir -p /opt/vwireguard
sudo chown "$USER":"$USER" /opt/vwireguard
git clone https://github.com/MmadF14/vwireguard.git /opt/vwireguard
cd /opt/vwireguard
```

#### Step 5 — Build the web assets  ← the step that is easy to forget

```bash
chmod +x prepare_assets.sh
./prepare_assets.sh
```

This runs `yarn install` and copies AdminLTE, jQuery, Bootstrap, FontAwesome,
toastr, select2 and the project's own CSS/JS into `assets/` and `static/`.

Verify it worked before moving on — `assets/` must contain more than `.gitkeep`:

```bash
ls assets/dist/js assets/dist/css assets/plugins
```

#### Step 6 — Build the binary

```bash
go mod download

go build -trimpath \
  -ldflags "-s -w \
    -X 'main.appVersion=$(git describe --tags --always)' \
    -X 'main.buildTime=$(date -u '+%Y-%m-%d %H:%M:%S UTC')' \
    -X 'main.gitCommit=$(git rev-parse --short HEAD)'" \
  -o vwireguard .
```

The `-ldflags` are optional; without them the About page just shows
`development` / `N/A`. A plain `go build -o vwireguard .` works too.

Check it:

```bash
./vwireguard --help
```

#### Step 7 — systemd service

```bash
sudo tee /etc/systemd/system/vwireguard.service > /dev/null << 'EOL'
[Unit]
Description=vWireguard Web Interface
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/vwireguard
ExecStart=/opt/vwireguard/vwireguard
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOL

sudo systemctl daemon-reload
sudo systemctl enable --now vwireguard
sudo systemctl status vwireguard --no-pager
```

The UI is now on `http://<server-ip>:5000` (default login `admin` / `admin` —
change it immediately).

> **Also enable your WireGuard interface at boot.** Enabling the panel does
> *not* enable the tunnel: `systemctl enable vwireguard` only starts the web
> app. Without the line below, after a reboot the panel comes up but
> `wg show` is empty and nobody can connect until you press *Apply Config*.
>
> ```bash
> sudo systemctl enable wg-quick@wg0    # use your real interface name
> ```

---

### Rebuilding after you change the code

Assets only need rebuilding when you touch `static/`, `custom/` or
`package.json`:

```bash
cd /opt/vwireguard
git pull

./prepare_assets.sh          # only if front-end files or deps changed
go build -trimpath -o vwireguard .

sudo systemctl restart vwireguard
```

### Cross-compiling from another machine

The project has **no cgo dependencies**, so you can build the Linux binary
anywhere — including on Windows or macOS — and copy it up:

```bash
# assets must still be prepared first, on the build machine
./prepare_assets.sh

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o vwireguard .
# GOARCH=arm64 for ARM servers

scp vwireguard root@<server>:/opt/vwireguard/vwireguard
ssh root@<server> 'systemctl restart vwireguard'
```

Windows users can run `build-linux.ps1` in the repo root, which does the same
thing and also offers `-Check` (runs `gofmt -l .`, `go build ./...` and
`go vet ./...` — the same checks CI runs) and `-Arch arm64`.

### Before you push (matching CI)

CI runs `golangci-lint` with `gofmt`, `goimports`, `govet`, `unused`,
`whitespace` and `misspell`, with default exclusions turned off. To avoid a red
build:

```bash
gofmt -l .        # any file listed here will fail CI
gofmt -w .        # auto-fix all of them
go vet ./...
go build ./...
```

### Build troubleshooting

| Error | Cause | Fix |
|---|---|---|
| `pattern assets/*: no matching files found` | `prepare_assets.sh` was never run | Run step 5 |
| UI loads but has no styling / broken layout | `assets/` had only `.gitkeep` at build time | Run step 5, then rebuild |
| `go.mod requires go >= 1.21` | Go from `apt` is too old | Install Go from go.dev (step 2) |
| `yarn: command not found` | Node/Yarn missing | Run step 3 |
| `prepare_assets.sh: Permission denied` | Script not executable | `chmod +x prepare_assets.sh` |
| Panel works but nobody can connect after a reboot | `wg-quick@wg0` not enabled | `sudo systemctl enable wg-quick@wg0` |

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
