function loadStats() {
    fetch('/api/stats')
        .then(r => r.json())
        .then(stats => {
            document.getElementById('stats').innerHTML = 
                'Total: ' + stats.total_interactions + ' | ' +
                'Uptime: ' + Math.round(stats.uptime_seconds) + 's';
        })
        .catch(err => console.error('Stats error:', err));
}

function loadInteractions() {
    fetch('/api/interactions')
        .then(r => r.json())
        .then(data => {
            const container = document.getElementById('interactions');
            if (data.length === 0) {
                container.innerHTML = '<p>No interactions yet. Use MCP fact-check tools to see data here.</p>';
                return;
            }
            
            // Save open state of details elements
            const openDetails = new Set();
            container.querySelectorAll('details[open]').forEach(details => {
                openDetails.add(details.dataset.id);
            });
            
            container.innerHTML = data.slice(-10).reverse().map((interaction, index) => 
                '<div class="interaction">' +
                    '<div class="timestamp">' + new Date(interaction.timestamp).toLocaleTimeString() + '</div>' +
                    '<div><span class="tool-name">' + interaction.tool_name + '</span></div>' +
                    '<details data-id="' + interaction.id + '-args"' + (openDetails.has(interaction.id + '-args') ? ' open' : '') + '><summary>Arguments</summary><pre>' + JSON.stringify(interaction.arguments, null, 2) + '</pre></details>' +
                    '<details data-id="' + interaction.id + '-resp"' + (openDetails.has(interaction.id + '-resp') ? ' open' : '') + '><summary>Response</summary><pre>' + JSON.stringify(interaction.response, null, 2) + '</pre></details>' +
                    (interaction.error ? '<div style="color: red;">Error: ' + interaction.error + '</div>' : '') +
                '</div>'
            ).join('');
        })
        .catch(err => {
            document.getElementById('interactions').innerHTML = 'Error loading interactions: ' + err;
        });
}

// Initialize
loadStats();
loadInteractions();
setInterval(() => {
    loadStats();
    loadInteractions();
}, 3000);