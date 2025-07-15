window.alpData = window.alpData || {};
window.alpData.v2ray = window.alpData.v2ray || {};

function parseV2Link(link) {
    if (!link) return null;
    try {
        if (link.startsWith('vmess://')) {
            const jsonStr = atob(link.slice(8));
            const v = JSON.parse(jsonStr);
            return {
                protocol: 'vmess',
                remote_address: v.add || '',
                remote_port: parseInt(v.port, 10) || 0,
                uuid: v.id || '',
                flow: v.flow || '',
                security: v.tls || 'none',
                server_name: v.host || '',
                fingerprint: v.fp || '',
                alpn: v.alpn ? v.alpn.split(',') : [],
                network: v.net || '',
                path: v.path || '',
                sni: v.sni || ''
            };
        }
        if (link.startsWith('vless://') || link.startsWith('trojan://')) {
            const urlObj = new URL(link);
            const protocol = urlObj.protocol.replace(':', '');
            const p = urlObj.searchParams;
            const cfg = {
                protocol: protocol,
                remote_address: urlObj.hostname,
                remote_port: parseInt(urlObj.port, 10) || 0,
                flow: p.get('flow') || '',
                security: p.get('security') || 'none',
                server_name: p.get('host') || '',
                fingerprint: p.get('fp') || '',
                alpn: p.get('alpn') ? p.get('alpn').split(',') : [],
                network: p.get('type') || '',
                path: p.get('path') || p.get('serviceName') || '',
                sni: p.get('sni') || ''
            };
            if (protocol === 'vless') cfg.uuid = urlObj.username;
            if (protocol === 'trojan') cfg.password = urlObj.username;
            return cfg;
        }
    } catch (e) {
        console.error('parseV2Link error', e);
    }
    return null;
}

function validateV2RayConfig() {
    const protocol = document.getElementById('v2ray_protocol').value;
    const remoteAddress = document.getElementById('v2ray_remote_address').value;
    const remotePort = document.getElementById('v2ray_remote_port').value;
    const security = document.getElementById('v2ray_security').value;
    const network = document.getElementById('v2ray_network').value;
    const uuid = document.getElementById('v2ray_uuid').value;
    const password = document.getElementById('v2ray_password').value;
    
    const errors = [];
    
    // Basic validation
    if (!remoteAddress) errors.push('Remote Address is required');
    if (!remotePort || remotePort <= 0) errors.push('Remote Port must be a positive number');
    if (!security) errors.push('Security setting is required');
    if (!network) errors.push('Network type is required');
    
    // Protocol-specific validation
    if (protocol === 'vmess' || protocol === 'vless') {
        if (!uuid) errors.push('UUID is required for ' + protocol + ' protocol');
    } else if (protocol === 'trojan') {
        if (!password) errors.push('Password is required for Trojan protocol');
    } else if (!protocol) {
        errors.push('Protocol selection is required');
    }
    
    return {
        isValid: errors.length === 0,
        errors: errors
    };
}

function showV2RayValidationResult() {
    const validation = validateV2RayConfig();
    const resultDiv = document.getElementById('v2ray_validation_result');
    
    if (!resultDiv) {
        const div = document.createElement('div');
        div.id = 'v2ray_validation_result';
        div.className = 'mt-3 p-3 rounded-lg';
        document.getElementById('v2ray_config').appendChild(div);
    }
    
    const div = document.getElementById('v2ray_validation_result');
    
    if (validation.isValid) {
        div.className = 'mt-3 p-3 bg-green-50 dark:bg-green-900/20 rounded-lg';
        div.innerHTML = '<p class="text-sm text-green-800 dark:text-green-200"><i class="fas fa-check-circle mr-2"></i>V2Ray configuration is valid!</p>';
    } else {
        div.className = 'mt-3 p-3 bg-red-50 dark:bg-red-900/20 rounded-lg';
        div.innerHTML = '<p class="text-sm text-red-800 dark:text-red-200"><i class="fas fa-exclamation-triangle mr-2"></i><strong>Configuration errors:</strong></p><ul class="text-sm text-red-700 dark:text-red-300 mt-1 ml-4">' + 
                       validation.errors.map(error => '<li>• ' + error + '</li>').join('') + '</ul>';
    }
}

