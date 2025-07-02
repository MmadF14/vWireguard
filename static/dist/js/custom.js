// Dark mode functionality
document.addEventListener('DOMContentLoaded', () => {
    const themeToggle = document.getElementById('theme-toggle');
    const body = document.body;
    const icon = themeToggle.querySelector('i');
    
    // Check for saved theme preference
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme) {
        body.setAttribute('data-theme', savedTheme);
        updateThemeIcon(icon, savedTheme);
    }
    
    // Theme toggle click handler
    themeToggle.addEventListener('click', () => {
        const currentTheme = body.getAttribute('data-theme');
        const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
        
        body.setAttribute('data-theme', newTheme);
        localStorage.setItem('theme', newTheme);
        updateThemeIcon(icon, newTheme);
    });
});

function updateThemeIcon(icon, theme) {
    if (theme === 'dark') {
        icon.classList.remove('fa-moon');
        icon.classList.add('fa-sun');
    } else {
        icon.classList.remove('fa-sun');
        icon.classList.add('fa-moon');
    }
}

// Card hover effects
document.addEventListener('DOMContentLoaded', () => {
    const cards = document.querySelectorAll('.card');
    cards.forEach(card => {
        card.addEventListener('mouseenter', () => {
            card.style.transform = 'translateY(-5px)';
            card.style.boxShadow = '0 8px 16px rgba(0,0,0,0.1)';
        });
        
        card.addEventListener('mouseleave', () => {
            card.style.transform = 'translateY(0)';
            card.style.boxShadow = '0 4px 6px rgba(0,0,0,0.1)';
        });
    });
});

// Smooth scrolling
document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        if (anchor.getAttribute('href') === '#') return; // Skip empty anchors
        
        anchor.addEventListener('click', function (e) {
            e.preventDefault();
            const targetId = this.getAttribute('href').substring(1);
            if (!targetId) return; // Skip if no target ID
            
            const target = document.getElementById(targetId);
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });
});

// Loading state for buttons
document.addEventListener('DOMContentLoaded', () => {
    const buttons = document.querySelectorAll('.btn');
    buttons.forEach(button => {
        button.addEventListener('click', function(e) {
            if (this.getAttribute('data-loading-text')) {
                const originalText = this.innerHTML;
                this.innerHTML = this.getAttribute('data-loading-text');
                this.classList.add('disabled');
                
                // Reset button after action completes
                setTimeout(() => {
                    this.innerHTML = originalText;
                    this.classList.remove('disabled');
                }, 2000);
            }
        });
    });
});

// Original functionality
function addGlobalStyle(css, id) {
    if (!document.querySelector('#' + id)) {
        let head = document.head;
        if (!head) { return; }
        let style = document.createElement('style');
        style.type = 'text/css';
        style.id = id;
        style.innerHTML = css;
        head.appendChild(style);
    }
}

function updateApplyConfigVisibility() {
    // Only show apply config button when there are actual changes pending
    // This should be called when clients are modified
    const hasChanges = $('.client-row').length > 0; // Check if there are clients
    if (hasChanges) {
        $("#apply-config-button").show();
    } else {
        $("#apply-config-button").hide();
    }
}

function updateQuotaInput() {
    const quotaPreset = document.getElementById('quota_preset');
    const quotaInput = document.getElementById('client_quota');
    
    if (quotaPreset.value === 'custom') {
        quotaInput.value = '';
        quotaInput.disabled = false;
    } else {
        quotaInput.value = quotaPreset.value;
        quotaInput.disabled = true;
    }
}

// Enhanced notifications
function showNotification(message, type = 'info') {
    toastr.options.closeDuration = 100;
    toastr.options.positionClass = 'toast-top-right-fix';
    
    switch(type) {
        case 'error':
            toastr.error(message);
            break;
        case 'success':
            toastr.success(message);
            break;
        case 'warning':
            toastr.warning(message);
            break;
        default:
            toastr.info(message);
    }
}

// Status indicator updates
function updateStatusIndicators() {
    const indicators = document.querySelectorAll('.status-indicator');
    indicators.forEach(indicator => {
        const status = indicator.getAttribute('data-status');
        if (status === 'online') {
            indicator.classList.add('status-online');
            indicator.classList.remove('status-offline');
        } else {
            indicator.classList.add('status-offline');
            indicator.classList.remove('status-online');
        }
    });
}

// Initialize all functionality
document.addEventListener('DOMContentLoaded', () => {
    // Add custom toast style
    addGlobalStyle(`
        .toast-top-right-fix {
            top: 67px;
            right: 12px;
        }
    `, 'toastrToastStyleFix');

    // Initialize tooltips
    $('[data-toggle="tooltip"]').tooltip();
    
    // Initialize popovers
    $('[data-toggle="popover"]').popover();
    
    // Update status indicators periodically
    updateStatusIndicators();
    setInterval(updateStatusIndicators, 30000);
    
    // Initialize quota input
    updateQuotaInput();
    
    // Update apply config visibility
    if (window.location.pathname.includes('/') && !window.location.pathname.includes('/tunnels')) {
        // Only check for apply config visibility on client pages
        updateApplyConfigVisibility();
    } else {
        // Hide apply config button on non-client pages
        $("#apply-config-button").hide();
    }
    
    // Initialize AllowedIPs tag inputs
    $("#client_allowed_ips").tagsInput({
        'width': '100%',
        'height': '75%',
        'interactive': true,
        'defaultText': 'Add More',
        'removeWithBackspace': true,
        'minChars': 0,
        'minInputWidth': '100%',
        'placeholderColor': '#666666'
    });

    $("#client_extra_allowed_ips").tagsInput({
        'width': '100%',
        'height': '75%',
        'interactive': true,
        'defaultText': 'Add More',
        'removeWithBackspace': true,
        'minChars': 0,
        'minInputWidth': '100%',
        'placeholderColor': '#666666'
    });
    
    // New Client modal event
    $("#modal_new_client").on('shown.bs.modal', function (e) {
        $("#client_name").val("");
        $("#client_email").val("");
        $("#client_public_key").val("");
        $("#client_preshared_key").val("");
        $("#client_allocated_ips").importTags('');
        $("#client_extra_allowed_ips").importTags('');
        $("#client_endpoint").val('');
        $("#client_telegram_userid").val('');
        $("#additional_notes").val('');
        updateSubnetRangesList("#subnet_ranges");
    });
    
    // Handle subnet range select
    $('#subnet_ranges').on('select2:select', function (e) {
        updateIPAllocationSuggestion();
    });
    
    // Apply config confirm button event
    $("#apply_config_confirm").click(function () {
        $.ajax({
            cache: false,
            method: 'POST',
            url: basePath + '/api/apply-wg-config',
            dataType: 'json',
            contentType: "application/json",
            success: function(data) {
                updateApplyConfigVisibility();
                $("#modal_apply_config").modal('hide');
                showNotification('Applied config successfully', 'success');
            },
            error: function(jqXHR, exception) {
                const responseJson = jQuery.parseJSON(jqXHR.responseText);
                showNotification(responseJson['message'], 'error');
            }
        });
    });

    // Handle modal accessibility
    $('#modal_apply_config').on('show.bs.modal', function () {
        $(this).removeAttr('aria-hidden');
    }).on('hidden.bs.modal', function () {
        $(this).attr('aria-hidden', 'true');
    });
});

// Client population function
function populateClient(client_id) {
    $.ajax({
        cache: false,
        method: 'GET',
        url: basePath + '/api/client/' + client_id,
        dataType: 'json',
        contentType: "application/json",
        success: function (resp) {
            renderClientList([resp]);
        },
        error: function (jqXHR, exception) {
            const responseJson = jQuery.parseJSON(jqXHR.responseText);
            showNotification(responseJson['message'], 'error');
        }
    });
}

// Date formatting function
function prettyDateTime(timestamp) {
    if (!timestamp) return '';
    
    // Handle different timestamp formats
    let date;
    if (typeof timestamp === 'string') {
        // Try parsing ISO format
        date = new Date(timestamp);
    } else {
        // Handle Unix timestamp (seconds or milliseconds)
        date = new Date(timestamp * (timestamp > 9999999999 ? 1 : 1000));
    }
    
    // Check if date is valid
    if (isNaN(date.getTime())) {
        return 'Invalid Date';
    }
    
    const now = new Date();
    const diff = Math.floor((now - date) / 1000);
    
    // If less than a minute ago
    if (diff < 60) {
        return 'Just now';
    }
    
    // If less than an hour ago
    if (diff < 3600) {
        const minutes = Math.floor(diff / 60);
        return `${minutes} minute${minutes > 1 ? 's' : ''} ago`;
    }
    
    // If less than a day ago
    if (diff < 86400) {
        const hours = Math.floor(diff / 3600);
        return `${hours} hour${hours > 1 ? 's' : ''} ago`;
    }
    
    // If less than a week ago
    if (diff < 604800) {
        const days = Math.floor(diff / 86400);
        return `${days} day${days > 1 ? 's' : ''} ago`;
    }
    
    // Format date for older timestamps
    const options = {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false
    };
    
    return date.toLocaleString(undefined, options);
}

// Format bytes to human readable format
function formatBytes(bytes, decimals = 2) {
    if (!bytes) return '0 B';
    
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
} 