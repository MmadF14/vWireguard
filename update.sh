#!/bin/bash

# vWireguard Panel - Update Script
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${BLUE}"
cat << "EOF"
██╗   ██╗██╗    ██╗██╗██████╗ ███████╗ ██████╗ ██╗   ██╗ █████╗ ██████╗ ██████╗ 
██║   ██║██║    ██║██║██╔══██╗██╔════╝██╔════╝ ██║   ██║██╔══██╗██╔══██╗██╔══██╗
██║   ██║██║ █╗ ██║██║██████╔╝█████╗  ██║  ███╗██║   ██║███████║██████╔╝██║  ██║
╚██╗ ██╔╝██║███╗██║██║██╔══██╗██╔══╝  ██║   ██║██║   ██║██╔══██║██╔══██╗██║  ██║
 ╚████╔╝ ╚███╔███╔╝██║██║  ██║███████╗╚██████╔╝╚██████╔╝██║  ██║██║  ██║██████╔╝
  ╚═══╝   ╚══╝╚══╝ ╚═╝╚═╝  ╚═╝╚══════╝ ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝ 
EOF
echo -e "${NC}"
echo -e "${CYAN}=== vWireguard Panel - Update ===${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}❌ Please run with root privileges:${NC}"
    echo -e "${GREEN}sudo bash update.sh${NC}"
    exit 1
fi

VWIREGUARD_DIR="/opt/vwireguard"
BACKUP_DIR="/opt/vwireguard-backup-$(date +%Y%m%d_%H%M%S)"

log() { echo -e "${GREEN}[$(date '+%H:%M:%S')] $1${NC}"; }
error() { echo -e "${RED}[$(date '+%H:%M:%S')] ❌ $1${NC}"; }
warn() { echo -e "${YELLOW}[$(date '+%H:%M:%S')] ⚠️  $1${NC}"; }

# Check if vWireguard is installed
check_installation() {
    if [ ! -d "$VWIREGUARD_DIR" ] || [ ! -f "$VWIREGUARD_DIR/vwireguard" ]; then
        error "vWireguard is not installed. Please install first with install.sh"
        exit 1
    fi
    log "vWireguard installation found"
}

# Get current version
get_current_version() {
    cd "$VWIREGUARD_DIR"
    CURRENT_VERSION=$(./vwireguard --version 2>/dev/null | grep -oP 'App Version\s*:\s*\K[^\s]+' || echo "unknown")
    log "Current version: $CURRENT_VERSION"
}

# Create backup
create_backup() {
    log "Creating backup..."
    
    # Create backup directory
    mkdir -p "$BACKUP_DIR"
    
    # Backup database
    cp -r "$VWIREGUARD_DIR/db" "$BACKUP_DIR/" 2>/dev/null
    
    # Backup static files if customized
    if [ -d "$VWIREGUARD_DIR/static" ]; then
        cp -r "$VWIREGUARD_DIR/static" "$BACKUP_DIR/" 2>/dev/null
    fi
    
    # Backup config files
    cp "$VWIREGUARD_DIR"/*.conf "$BACKUP_DIR/" 2>/dev/null || true
    
    # Backup current binary
    cp "$VWIREGUARD_DIR/vwireguard" "$BACKUP_DIR/vwireguard.old" 2>/dev/null
    
    log "Backup saved to $BACKUP_DIR"
}

# Stop services
stop_services() {
    log "Stopping services..."
    systemctl stop vwireguard
    sleep 2
}

# Download and install new version
update_binary() {
    log "Downloading new version..."
    
    cd "$VWIREGUARD_DIR"
    local arch=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
    local release_url="https://github.com/MmadF14/vwireguard/releases/latest/download/vwireguard-linux-${arch}.tar.gz"
    
    # Try to download prebuilt binary
    if wget -q --spider "$release_url" 2>/dev/null; then
        log "Downloading prebuilt version..."
        wget -q "$release_url" -O vwireguard-new.tar.gz
        
        # Extract new version
        mkdir -p temp_extract
        tar -xzf vwireguard-new.tar.gz -C temp_extract
        
        # Replace binary
        mv temp_extract/vwireguard ./vwireguard-new
        chmod +x vwireguard-new
        
        # Update templates and static files if they exist
        if [ -d "temp_extract/templates" ]; then
            cp -r temp_extract/templates ./
        fi
        
        if [ -d "temp_extract/static" ]; then
            cp -r temp_extract/static ./
        fi
        
        # Replace old binary with new one
        mv vwireguard vwireguard.old
        mv vwireguard-new vwireguard
        
        # Cleanup
        rm -rf temp_extract vwireguard-new.tar.gz
        
    else
        log "Building from source..."
        
        # Install Go if not present
        if ! command -v go >/dev/null 2>&1; then
            log "Installing Go..."
            local go_version="go1.21.5"
            wget -q "https://go.dev/dl/${go_version}.linux-${arch}.tar.gz" -O /tmp/go.tar.gz
            rm -rf /usr/local/go
            tar -C /usr/local -xzf /tmp/go.tar.gz
            export PATH=$PATH:/usr/local/go/bin
        fi
        
        # Clone and build
        git clone https://github.com/MmadF14/vwireguard.git temp_build
        cd temp_build
        
        export PATH=$PATH:/usr/local/go/bin
        go mod tidy
        go build -ldflags="-s -w" -o ../vwireguard-new
        
        cd ..
        rm -rf temp_build
        
        # Replace binary
        mv vwireguard vwireguard.old
        mv vwireguard-new vwireguard
        chmod +x vwireguard
    fi
    
    log "Binary updated successfully"
}

# Start services
start_services() {
    log "Starting services..."
    systemctl start vwireguard
    sleep 3
    
    if systemctl is-active --quiet vwireguard; then
        log "Service started successfully"
    else
        error "Failed to start service"
        log "Restoring from backup..."
        restore_backup
        exit 1
    fi
}

# Restore backup if update fails
restore_backup() {
    log "Restoring from backup..."
    
    systemctl stop vwireguard
    
    if [ -f "$BACKUP_DIR/vwireguard.old" ]; then
        cp "$BACKUP_DIR/vwireguard.old" "$VWIREGUARD_DIR/vwireguard"
        chmod +x "$VWIREGUARD_DIR/vwireguard"
    fi
    
    systemctl start vwireguard
    
    if systemctl is-active --quiet vwireguard; then
        log "Restore successful"
    else
        error "Failed to restore - please check manually"
    fi
}

# Get new version
get_new_version() {
    cd "$VWIREGUARD_DIR"
    NEW_VERSION=$(./vwireguard --version 2>/dev/null | grep -oP 'App Version\s*:\s*\K[^\s]+' || echo "unknown")
    log "New version: $NEW_VERSION"
}

# Show update summary
show_summary() {
    echo ""
    echo -e "${GREEN}=====================================${NC}"
    echo -e "${GREEN}✅ Update completed successfully!${NC}"
    echo -e "${GREEN}=====================================${NC}"
    echo ""
    echo -e "${CYAN}📋 Update Information:${NC}"
    echo -e "  ${YELLOW}Previous version:${NC} $CURRENT_VERSION"
    echo -e "  ${YELLOW}New version:${NC} $NEW_VERSION"
    echo -e "  ${YELLOW}Backup location:${NC} $BACKUP_DIR"
    echo ""
    echo -e "${CYAN}⚙️ Useful commands:${NC}"
    echo -e "  ${YELLOW}Status:${NC} systemctl status vwireguard"
    echo -e "  ${YELLOW}Logs:${NC} journalctl -u vwireguard -f"
    echo -e "  ${YELLOW}Restart:${NC} systemctl restart vwireguard"
    echo ""
    echo -e "${GREEN}🎉 Panel updated successfully!${NC}"
    echo -e "${GREEN}=====================================${NC}"
}

# Cleanup old backups (keep last 5)
cleanup_backups() {
    log "Cleaning up old backups..."
    ls -dt /opt/vwireguard-backup-* 2>/dev/null | tail -n +6 | xargs -r rm -rf
}

# Main function
main() {
    log "Starting vWireguard update..."
    
    check_installation
    get_current_version
    
    echo ""
    echo -e "${YELLOW}Are you sure you want to update? (y/n):${NC}"
    read -p "" -n 1 -r
    echo ""
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log "Update cancelled"
        exit 0
    fi
    
    create_backup
    stop_services
    update_binary
    start_services
    get_new_version
    cleanup_backups
    show_summary
}

# Run main function
main "$@" 