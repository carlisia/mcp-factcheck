function loadStats() {
    fetch('/api/stats')
        .then(r => r.json())
        .then(stats => {
            const uptimeMinutes = Math.floor(stats.uptime_seconds / 60);
            const uptimeSeconds = Math.round(stats.uptime_seconds % 60);
            const uptimeDisplay = uptimeMinutes > 0 ? 
                uptimeMinutes + 'm ' + uptimeSeconds + 's' : 
                uptimeSeconds + 's';
                
            document.getElementById('stats').innerHTML = 
                '<div style="display: flex; flex-wrap: wrap; gap: 20px;">' +
                    '<div class="metric" style="display: flex; gap: 8px;"><span>Total Interactions:</span><span class="metric-value">' + (stats.total_interactions || 0) + '</span></div>' +
                    '<div class="metric" style="display: flex; gap: 8px;"><span>Uptime:</span><span class="metric-value">' + uptimeDisplay + '</span></div>' +
                    '<div class="metric" style="display: flex; gap: 8px;"><span>Status:</span><span class="metric-value" style="color: #27ae60;">Running</span></div>' +
                '</div>';
        })
        .catch(err => {
            console.error('Stats error:', err);
            document.getElementById('stats').innerHTML = 
                '<div style="color: #e74c3c;">Error loading stats: ' + err + '</div>';
        });
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
            
            // Save open state of all expandable elements
            const openDetails = new Set();
            
            // Save request group states
            container.querySelectorAll('[id^="request-content-"]').forEach(content => {
                if (content.style.display !== 'none') {
                    const requestId = content.id.replace('request-content-', '');
                    openDetails.add(requestId + '-group');
                }
            });
            
            // Save tool entry states
            container.querySelectorAll('[id^="tool-content-"]').forEach(toolContent => {
                if (toolContent && toolContent.style.display !== 'none') {
                    const interactionId = toolContent.id.replace('tool-content-', '');
                    openDetails.add(interactionId + '-tool');
                }
            });
            
            // Save subsection states
            container.querySelectorAll('.expand-content').forEach(content => {
                if (content.style.display !== 'none') {
                    const header = content.previousElementSibling;
                    if (header && header.onclick) {
                        const onclickStr = header.onclick.toString();
                        const match = onclickStr.match(/'([^']+)'/);
                        if (match) {
                            openDetails.add(match[1]);
                        }
                    }
                }
            });
            
            // Group interactions by request_id, with fallback to individual display
            try {
                const groupedInteractions = groupInteractionsByRequest(data.slice(-50)); // Get more data for grouping
                
                if (groupedInteractions.length > 0) {
                    container.innerHTML = groupedInteractions.slice(-10).reverse().map((group, index) => 
                        createRequestGroupHTML(group, openDetails)
                    ).join('');
                } else {
                    // Fallback to individual interactions if grouping fails
                    container.innerHTML = data.slice(-10).reverse().map((interaction, index) => 
                        createInteractionHTML(interaction, openDetails, false)
                    ).join('');
                }
            } catch (err) {
                console.error('Grouping error:', err);
                // Fallback to individual interactions
                container.innerHTML = data.slice(-10).reverse().map((interaction, index) => 
                    createInteractionHTML(interaction, openDetails, false)
                ).join('');
            }
        })
        .catch(err => {
            document.getElementById('interactions').innerHTML = 'Error loading interactions: ' + err;
        });
}

function groupInteractionsByRequest(interactions) {
    const groups = new Map();
    
    interactions.forEach(interaction => {
        const requestId = interaction.request_id || 'unknown-' + interaction.id;
        
        if (!groups.has(requestId)) {
            groups.set(requestId, {
                requestId: requestId,
                interactions: [],
                startTime: interaction.timestamp,
                totalTime: 0
            });
        }
        
        const group = groups.get(requestId);
        group.interactions.push(interaction);
        
        // Update timing info
        const interactionTime = new Date(interaction.timestamp);
        const groupStartTime = new Date(group.startTime);
        if (interactionTime < groupStartTime) {
            group.startTime = interaction.timestamp;
        }
        
        // Calculate total processing time for the request
        group.totalTime += interaction.processing_ms || 0;
    });
    
    // Convert to array and sort by start time
    return Array.from(groups.values()).sort((a, b) => 
        new Date(a.startTime) - new Date(b.startTime)
    );
}

function createRequestGroupHTML(group, openDetails) {
    const groupOpen = openDetails.has(group.requestId + '-group');
    const toolNames = group.interactions.map(i => i.tool_name).join(', ');
    const startTime = new Date(group.startTime).toLocaleTimeString();
    
    return '<div class="interaction" style="border-left: 4px solid #e67e22;">' +
        '<div class="expand-header" onclick="toggleRequestGroup(this, \'' + group.requestId + '\')" style="margin-bottom: 10px; background: #e67e22; color: white;">' +
            '<span class="tool-name">Request ' + group.requestId + '</span> ' +
            '<span style="opacity: 0.8;">' + group.interactions.length + ' tools at ' + startTime + ' (' + group.totalTime + 'ms total)</span>' +
            '<div style="font-size: 0.9em; opacity: 0.8; margin-top: 2px;">' + toolNames + '</div>' +
        '</div>' +
        '<div id="request-content-' + group.requestId + '" class="expand-content" style="display: ' + (groupOpen ? 'block' : 'none') + '; padding: 15px;">' +
            group.interactions.map(interaction => createInteractionHTML(interaction, openDetails, true)).join('') +
        '</div>' +
    '</div>';
}

function createInteractionHTML(interaction, openDetails, isGrouped = false) {
    const analysis = analyzeInteraction(interaction);
    
    // Check if this tool entry should be open
    const toolOpen = openDetails.has(interaction.id + '-tool');
    
    const headerStyle = isGrouped ? 
        'margin-bottom: 10px; background: #34495e; color: white;' : 
        'margin-bottom: 10px; background: #007acc; color: white;';
        
    const containerClass = isGrouped ? 'tool-in-group' : 'interaction';
    
    return '<div class="' + containerClass + '" style="' + (isGrouped ? 'margin: 10px 0; background: #f9f9f9; border-radius: 5px;' : '') + '">' +
        '<div class="expand-header" onclick="toggleToolEntry(this, \'' + interaction.id + '\')" style="' + headerStyle + '">' +
            '<span class="tool-name">' + interaction.tool_name + '</span> ' +
            '<span style="opacity: 0.8;">at ' + new Date(interaction.timestamp).toLocaleTimeString() + ' (' + interaction.processing_ms + 'ms)</span>' +
        '</div>' +
        '<div id="tool-content-' + interaction.id + '" class="expand-content" style="display: ' + (toolOpen ? 'block' : 'none') + '; padding: 15px;">' +
            
            // 1. Arguments (what went in)
            '<div class="expandable">' +
                '<div class="expand-header" onclick="toggleSubSection(event, this, \'' + interaction.id + '-args\')" style="background: #34495e;">Arguments</div>' +
                '<div class="expand-content" style="display: ' + (openDetails.has(interaction.id + '-args') ? 'block' : 'none') + ';"><pre>' + JSON.stringify(interaction.arguments, null, 2) + '</pre></div>' +
            '</div>' +
            
            // 2. Response (what came back)
            '<div class="expandable">' +
                '<div class="expand-header" onclick="toggleSubSection(event, this, \'' + interaction.id + '-resp\')" style="background: #34495e;">Response</div>' +
                '<div class="expand-content" style="display: ' + (openDetails.has(interaction.id + '-resp') ? 'block' : 'none') + ';"><pre>' + JSON.stringify(interaction.response, null, 2) + '</pre></div>' +
            '</div>' +
            
            // 3. Coverage Mapping (how well validated - actionable)
            (analysis.coverage ? createCoverageMapping(analysis.coverage, interaction.id, openDetails) : '') +
            
            // 4. Similarity Analysis (technical details)
            (analysis.similarities ? createSimilarityVisualization(analysis.similarities, interaction.id, openDetails) : '') +
            
            // Enhanced Analysis Grid (content + performance metrics)
            '<div class="analysis-grid">' +
                createContentAnalysisPanel(analysis) +
                createPerformancePanel(analysis) +
            '</div>' +
            
            (interaction.error ? '<div style="color: red; margin-top: 10px;">Error: ' + interaction.error + '</div>' : '') +
        '</div>' +
    '</div>';
}

function analyzeInteraction(interaction) {
    const analysis = {
        contentLength: 0,
        estimatedTokens: 0,
        processingTime: interaction.processing_ms || 0,
        toolType: interaction.tool_name
    };
    
    // Analyze content
    if (interaction.arguments && interaction.arguments.content) {
        const content = interaction.arguments.content;
        analysis.contentLength = content.length;
        analysis.estimatedTokens = Math.ceil(content.length / 4); // Rough estimate
        
        // Extract similarities for validation tools
        if (interaction.tool_name.includes('validate') && interaction.response) {
            analysis.similarities = extractSimilarities(interaction.response);
            analysis.coverage = analyzeCoverage(content, analysis.similarities);
        }
    }
    
    return analysis;
}

function createContentAnalysisPanel(analysis) {
    return '<div class="analysis-panel">' +
        '<div class="analysis-title">Content Analysis</div>' +
        '<div class="metric"><span>Characters:</span><span class="metric-value">' + analysis.contentLength.toLocaleString() + '</span></div>' +
        '<div class="metric"><span>Est. Tokens:</span><span class="metric-value">' + analysis.estimatedTokens.toLocaleString() + '</span></div>' +
        '<div class="metric"><span>Tool:</span><span class="metric-value">' + analysis.toolType + '</span></div>' +
    '</div>';
}

function createPerformancePanel(analysis) {
    return '<div class="analysis-panel">' +
        '<div class="analysis-title">Performance</div>' +
        '<div class="metric"><span>Processing:</span><span class="metric-value">' + analysis.processingTime + 'ms</span></div>' +
        '<div class="metric"><span>Rate:</span><span class="metric-value">' + 
            (analysis.contentLength > 0 ? Math.round(analysis.contentLength / (analysis.processingTime / 1000)) : 0) + ' chars/sec</span></div>' +
        '<div class="perf-chart">' +
            '<div class="perf-bar" style="height: ' + Math.min(100, analysis.processingTime / 50) + '%;">' +
                '<div class="perf-label">Total</div>' +
            '</div>' +
        '</div>' +
    '</div>';
}

function createSimilarityVisualization(similarities, interactionId, openDetails) {
    if (!similarities || similarities.length === 0) return '';
    
    const sectionId = interactionId + '-similarity';
    const isOpen = openDetails.has(sectionId);
    
    return '<div class="expandable">' +
        '<div class="expand-header" onclick="toggleSubSection(event, this, \'' + sectionId + '\')">Similarity Analysis (' + similarities.length + ' matches)</div>' +
        '<div class="expand-content" style="display: ' + (isOpen ? 'block' : 'none') + ';">' +
            similarities.map((sim, i) => 
                '<div class="similarity-bar">' +
                    '<div class="similarity-fill" style="width: ' + (sim.score * 100) + '%;"></div>' +
                    '<div class="similarity-label">Ref ' + (i + 1) + '</div>' +
                    '<div class="similarity-score">' + (sim.score * 100).toFixed(1) + '%</div>' +
                '</div>' +
                '<div style="font-size: 0.8em; margin-bottom: 10px; color: #666;">' + sim.topic.substring(0, 100) + '...</div>'
            ).join('') +
        '</div>' +
    '</div>';
}

function createCoverageMapping(coverage, interactionId, openDetails) {
    if (!coverage || coverage.chunks.length === 0) return '';
    
    const sectionId = interactionId + '-coverage';
    const isOpen = openDetails.has(sectionId);
    
    return '<div class="expandable">' +
        '<div class="expand-header" onclick="toggleSubSection(event, this, \'' + sectionId + '\')">Coverage Mapping</div>' +
        '<div class="expand-content" style="display: ' + (isOpen ? 'block' : 'none') + ';">' +
            '<div class="coverage-map">' +
                coverage.chunks.map(chunk => 
                    '<div class="content-chunk coverage-' + chunk.level + '">' +
                        '<strong>' + chunk.level.toUpperCase() + ' Coverage:</strong> ' + chunk.text +
                    '</div>'
                ).join('') +
            '</div>' +
        '</div>' +
    '</div>';
}

function extractSimilarities(response) {
    try {
        if (response && response[0] && response[0].text) {
            const data = JSON.parse(response[0].text);
            if (data.references) {
                return data.references.map(ref => ({
                    score: ref.relevance || 0,
                    topic: ref.topic || ref.summary || 'Unknown'
                }));
            }
        }
    } catch (e) {
        console.log('Could not extract similarities:', e);
    }
    return [];
}

function analyzeCoverage(content, similarities) {
    if (!content || !similarities) return null;
    
    // Simple coverage analysis - split content into sentences
    const sentences = content.split(/[.!?]+/).filter(s => s.trim().length > 0);
    const avgSimilarity = similarities.length > 0 ? 
        similarities.reduce((sum, sim) => sum + sim.score, 0) / similarities.length : 0;
    
    return {
        chunks: sentences.map((sentence, i) => ({
            text: sentence.trim(),
            level: avgSimilarity > 0.9 ? 'high' : avgSimilarity > 0.7 ? 'medium' : 'low'
        }))
    };
}

function toggleRequestGroup(header, requestId) {
    const content = document.getElementById('request-content-' + requestId);
    const isVisible = content.style.display !== 'none';
    content.style.display = isVisible ? 'none' : 'block';
    header.style.borderRadius = isVisible ? '5px' : '5px 5px 0 0';
}

function toggleToolEntry(header, interactionId) {
    const content = document.getElementById('tool-content-' + interactionId);
    const isVisible = content.style.display !== 'none';
    content.style.display = isVisible ? 'none' : 'block';
    header.style.borderRadius = isVisible ? '5px' : '5px 5px 0 0';
}

function toggleSubSection(event, header, sectionId) {
    // Prevent event bubbling to parent tool entry
    event.stopPropagation();
    
    const content = header.nextElementSibling;
    const isVisible = content.style.display !== 'none';
    content.style.display = isVisible ? 'none' : 'block';
    header.style.borderRadius = isVisible ? '5px' : '5px 5px 0 0';
}

// Initialize
loadStats();
loadInteractions();
setInterval(() => {
    loadStats();
    loadInteractions();
}, 3000);