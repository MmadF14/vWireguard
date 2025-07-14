/**
 * Optimized WireGuard Configuration Management
 * Provides enhanced feedback and status monitoring for WireGuard operations
 */

class WireGuardOptimized {
    constructor() {
        this.basePath = window.basePath || '';
        this.statusCheckInterval = null;
    }

    /**
     * Apply configuration with enhanced feedback
     */
    async applyConfig() {
        const confirmMessage = 'Apply WireGuard configuration?\n\n' +
            '✅ New optimized method will preserve active connections\n' +
            '⚠️ Only new peers will be added without disruption\n' +
            '🔄 Existing connections will remain stable';

        if (!confirm(confirmMessage)) {
            return;
        }

        // Show loading state
        this.showLoadingState();

        try {
            const response = await fetch(`${this.basePath}/api/apply-wg-config`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
            });

            const data = await response.json();

            if (data.success) {
                this.showSuccess('Configuration applied successfully! ✅\n\n' +
                    '• Active connections preserved\n' +
                    '• New peers added seamlessly\n' +
                    '• No service interruption');
                
                // Refresh the page after a short delay
                setTimeout(() => {
                    window.location.reload();
                }, 2000);
            } else {
                this.showError('Configuration failed: ' + (data.message || 'Unknown error'));
            }
        } catch (error) {
            console.error('Apply config error:', error);
            this.showError('Network error: ' + error.message);
        } finally {
            this.hideLoadingState();
        }
    }

    /**
     * Add a single peer without disrupting others
     */
    async addPeer(peerData) {
        try {
            const response = await fetch(`${this.basePath}/api/wg/add-peer`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(peerData),
            });

            const data = await response.json();

            if (data.success) {
                this.showSuccess('Peer added successfully! ✅\n\n' +
                    '• Added without disrupting existing connections\n' +
                    '• Interface remains stable');
                return true;
            } else {
                this.showError('Failed to add peer: ' + (data.message || 'Unknown error'));
                return false;
            }
        } catch (error) {
            console.error('Add peer error:', error);
            this.showError('Network error: ' + error.message);
            return false;
        }
    }

    /**
     * Remove a single peer
     */
    async removePeer(publicKey) {
        const confirmMessage = 'Remove peer?\n\n' +
            '⚠️ This will disconnect the client immediately\n' +
            '✅ Other clients will remain unaffected';

        if (!confirm(confirmMessage)) {
            return false;
        }

        try {
            const response = await fetch(`${this.basePath}/api/wg/remove-peer`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ public_key: publicKey }),
            });

            const data = await response.json();

            if (data.success) {
                this.showSuccess('Peer removed successfully! ✅\n\n' +
                    '• Client disconnected\n' +
                    '• Other clients unaffected');
                return true;
            } else {
                this.showError('Failed to remove peer: ' + (data.message || 'Unknown error'));
                return false;
            }
        } catch (error) {
            console.error('Remove peer error:', error);
            this.showError('Network error: ' + error.message);
            return false;
        }
    }

    /**
     * Get interface status
     */
    async getInterfaceStatus() {
        try {
            const response = await fetch(`${this.basePath}/api/wg/status`);
            const data = await response.json();

            if (data.success) {
                return {
                    interface: data.interface,
                    status: data.status,
                    active: data.active,
                };
            } else {
                throw new Error(data.message || 'Failed to get status');
            }
        } catch (error) {
            console.error('Get status error:', error);
            throw error;
        }
    }

    /**
     * Monitor interface status with periodic updates
     */
    startStatusMonitoring() {
        if (this.statusCheckInterval) {
            clearInterval(this.statusCheckInterval);
        }

        this.statusCheckInterval = setInterval(async () => {
            try {
                const status = await this.getInterfaceStatus();
                this.updateStatusDisplay(status);
            } catch (error) {
                console.error('Status monitoring error:', error);
            }
        }, 5000); // Check every 5 seconds
    }

    /**
     * Stop status monitoring
     */
    stopStatusMonitoring() {
        if (this.statusCheckInterval) {
            clearInterval(this.statusCheckInterval);
            this.statusCheckInterval = null;
        }
    }

    /**
     * Update status display in the UI
     */
    updateStatusDisplay(status) {
        const statusElement = document.getElementById('wg-status');
        if (statusElement) {
            const statusClass = status.active ? 'text-green-600' : 'text-red-600';
            const statusIcon = status.active ? '🟢' : '🔴';
            
            statusElement.innerHTML = `
                <span class="${statusClass}">
                    ${statusIcon} ${status.interface}: ${status.status}
                </span>
            `;
        }
    }

    /**
     * Show loading state
     */
    showLoadingState() {
        const applyButton = document.getElementById('apply-config-button');
        if (applyButton) {
            applyButton.disabled = true;
            applyButton.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>Applying...';
        }

        // Show loading overlay
        this.showOverlay('Applying configuration...', 'info');
    }

    /**
     * Hide loading state
     */
    hideLoadingState() {
        const applyButton = document.getElementById('apply-config-button');
        if (applyButton) {
            applyButton.disabled = false;
            applyButton.innerHTML = '<i class="fas fa-check mr-2"></i>Apply Config';
        }

        // Hide loading overlay
        this.hideOverlay();
    }

    /**
     * Show success message
     */
    showSuccess(message) {
        this.showNotification(message, 'success');
    }

    /**
     * Show error message
     */
    showError(message) {
        this.showNotification(message, 'error');
    }

    /**
     * Show notification
     */
    showNotification(message, type) {
        // Try to use existing notification system
        if (typeof showNotification === 'function') {
            showNotification(message, type);
        } else if (typeof toastr !== 'undefined') {
            if (type === 'success') {
                toastr.success(message);
            } else {
                toastr.error(message);
            }
        } else {
            // Fallback to alert
            alert(message);
        }
    }

    /**
     * Show overlay
     */
    showOverlay(message, type = 'info') {
        let overlay = document.getElementById('wg-overlay');
        if (!overlay) {
            overlay = document.createElement('div');
            overlay.id = 'wg-overlay';
            overlay.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50';
            overlay.innerHTML = `
                <div class="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-sm mx-4">
                    <div class="flex items-center">
                        <div class="flex-shrink-0">
                            <i class="fas fa-spinner fa-spin text-blue-600 text-xl"></i>
                        </div>
                        <div class="ml-3">
                            <p class="text-sm font-medium text-gray-900 dark:text-white">${message}</p>
                        </div>
                    </div>
                </div>
            `;
            document.body.appendChild(overlay);
        }
    }

    /**
     * Hide overlay
     */
    hideOverlay() {
        const overlay = document.getElementById('wg-overlay');
        if (overlay) {
            overlay.remove();
        }
    }

    /**
     * Initialize the optimized WireGuard interface
     */
    init() {
        // Replace the global applyConfig function
        window.applyConfig = () => this.applyConfig();

        // Start status monitoring
        this.startStatusMonitoring();

        // Add status display to the page if it doesn't exist
        this.addStatusDisplay();

        console.log('WireGuard Optimized initialized');
    }

    /**
     * Add status display to the page
     */
    addStatusDisplay() {
        const header = document.querySelector('header');
        if (header && !document.getElementById('wg-status')) {
            const statusDiv = document.createElement('div');
            statusDiv.id = 'wg-status';
            statusDiv.className = 'text-sm font-medium';
            statusDiv.innerHTML = '<span class="text-gray-600">🔄 Checking status...</span>';
            
            // Insert after the apply config button
            const applyButton = document.getElementById('apply-config-button');
            if (applyButton && applyButton.parentNode) {
                applyButton.parentNode.insertBefore(statusDiv, applyButton.nextSibling);
            }
        }
    }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.wgOptimized = new WireGuardOptimized();
    window.wgOptimized.init();
}); 