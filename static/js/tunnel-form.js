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


