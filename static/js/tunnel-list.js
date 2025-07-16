window.tunnelList = function(tunnels) {
    return {
        tunnels: tunnels,
        init() {
            const ws = new WebSocket(`${window.basePath}/api/tunnels/health/stream`);
            ws.onmessage = (ev) => {
                const data = JSON.parse(ev.data);
                if(this.tunnels[data.id]) {
                    this.tunnels[data.id].status = data.color;
                }
            };
        },
        async toggle(id) {
            const t = this.tunnels[id];
            const action = t.status === 'inactive' ? 'start' : 'stop';
            t.status = action === 'start' ? 'green' : 'inactive';
            await fetch(`${window.basePath}/api/tunnels/${id}/${action}`, {method:'PUT'});
        }
    };
}
