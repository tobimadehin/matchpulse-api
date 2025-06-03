package main

// HTML template for the API documentation homepage
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MatchPulse API</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            line-height: 1.6;
            color: #2d3748;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 2rem;
        }
        
        .header {
            text-align: center;
            color: white;
            margin-bottom: 3rem;
        }
        
        .header h1 {
            font-size: 3rem;
            font-weight: 800;
            margin-bottom: 0.5rem;
            text-shadow: 0 2px 4px rgba(0,0,0,0.3);
        }
        
        .header p {
            font-size: 1.2rem;
            opacity: 0.9;
            margin-bottom: 2rem;
        }
        
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-bottom: 3rem;
        }
        
        .stat-card {
            background: rgba(255, 255, 255, 0.1);
            border-radius: 12px;
            padding: 1.5rem;
            text-align: center;
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255, 255, 255, 0.2);
        }
        
        .stat-card h3 {
            color: white;
            font-size: 2rem;
            font-weight: 700;
            margin-bottom: 0.5rem;
        }
        
        .stat-card p {
            color: rgba(255, 255, 255, 0.8);
            font-size: 0.9rem;
        }
        
        .main-content {
            background: white;
            border-radius: 16px;
            padding: 2rem;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
        }
        
        .section {
            margin-bottom: 2.5rem;
        }
        
        .section h2 {
            color: #2d3748;
            font-size: 1.5rem;
            font-weight: 600;
            margin-bottom: 1rem;
            padding-bottom: 0.5rem;
            border-bottom: 2px solid #e2e8f0;
        }
        
        .endpoints-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
            gap: 1.5rem;
        }
        
        .endpoint {
            background: #f7fafc;
            border: 1px solid #e2e8f0;
            border-radius: 12px;
            padding: 1.5rem;
            transition: all 0.3s ease;
            position: relative;
            overflow: hidden;
        }
        
        .endpoint:hover {
            transform: translateY(-4px);
            box-shadow: 0 12px 24px rgba(0,0,0,0.1);
            border-color: #667eea;
        }
        
        .endpoint::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 4px;
            background: linear-gradient(90deg, #667eea, #764ba2);
            opacity: 0;
            transition: opacity 0.3s ease;
        }
        
        .endpoint:hover::before {
            opacity: 1;
        }
        
        .endpoint h3 {
            color: #2d3748;
            font-size: 1rem;
            font-weight: 600;
            margin-bottom: 0.5rem;
            font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
        }
        
        .endpoint p {
            color: #718096;
            font-size: 0.9rem;
            margin-bottom: 1rem;
        }
        
        .endpoint a {
            color: #667eea;
            text-decoration: none;
            font-weight: 500;
            font-size: 0.9rem;
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
        }
        
        .endpoint a:hover {
            color: #5a67d8;
        }
        
        .badge {
            display: inline-block;
            padding: 0.25rem 0.75rem;
            border-radius: 20px;
            font-size: 0.75rem;
            font-weight: 500;
            margin-bottom: 0.5rem;
        }
        
        .badge-fast {
            background: #c6f6d5;
            color: #22543d;
        }
        
        .badge-medium {
            background: #fbd38d;
            color: #744210;
        }
        
        .badge-slow {
            background: #fed7d7;
            color: #742a2a;
        }
        
        .badge-realtime {
            background: #e9d8fd;
            color: #553c9a;
        }
        
        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 2rem;
            margin-top: 2rem;
        }
        
        .info-section h3 {
            color: #2d3748;
            font-size: 1.1rem;
            font-weight: 600;
            margin-bottom: 1rem;
        }
        
        .info-section ul {
            list-style: none;
            space-y: 0.5rem;
        }
        
        .info-section li {
            color: #4a5568;
            font-size: 0.9rem;
            padding: 0.5rem 0;
            border-bottom: 1px solid #e2e8f0;
        }
        
        .info-section li:last-child {
            border-bottom: none;
        }
        
        .footer {
            text-align: center;
            padding: 2rem 0;
            color: #718096;
            font-size: 0.9rem;
            border-top: 1px solid #e2e8f0;
            margin-top: 2rem;
        }
        
        .footer a {
            color: #667eea;
            text-decoration: none;
        }
        
        .footer a:hover {
            text-decoration: underline;
        }
        
        .version-info {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-top: 1rem;
            padding-top: 1rem;
            border-top: 1px solid #e2e8f0;
            font-size: 0.8rem;
            color: #a0aec0;
        }
        
        @media (max-width: 768px) {
            .container {
                padding: 1rem;
            }
            
            .header h1 {
                font-size: 2rem;
            }
            
            .stats-grid,
            .endpoints-grid,
            .info-grid {
                grid-template-columns: 1fr;
            }
            
            .version-info {
                flex-direction: column;
                gap: 0.5rem;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚽ MatchPulse API</h1>
            <p>Real-time football match data for testing state management libraries</p>
            
            <div class="stats-grid">
                <div class="stat-card">
                    <h3>{{.ActiveMatches}}</h3>
                    <p>Live Matches</p>
                </div>
                <div class="stat-card">
                    <h3>{{.TotalEvents}}</h3>
                    <p>Match Events</p>
                </div>
                <div class="stat-card">
                    <h3>6</h3>
                    <p>API Endpoints</p>
                </div>
                <div class="stat-card">
                    <h3>∞</h3>
                    <p>Rate Limit</p>
                </div>
            </div>
        </div>
        
        <div class="main-content">
            <div class="section">
                <h2>API Endpoints</h2>
                <div class="endpoints-grid">
                    <div class="endpoint">
                        <span class="badge badge-realtime">Real-time</span>
                        <h3>GET /api/v1/health</h3>
                        <p>API health check and system status</p>
                        <a href="/api/v1/health" target="_blank">Try it →</a>
                    </div>
                    
                    <div class="endpoint">
                        <span class="badge badge-slow">20-45s updates</span>
                        <h3>GET /api/v1/matches</h3>
                        <p>All live matches with scores and status</p>
                        <a href="/api/v1/matches" target="_blank">Try it →</a>
                    </div>
                    
                    <div class="endpoint">
                        <span class="badge badge-slow">20-45s updates</span>
                        <h3>GET /api/v1/matches/{id}</h3>
                        <p>Detailed information for a specific match</p>
                        <a href="/api/v1/matches/1" target="_blank">Try it →</a>
                    </div>
                    
                    <div class="endpoint">
                        <span class="badge badge-fast">2-5s updates</span>
                        <h3>GET /api/v1/matches/{id}/stats</h3>
                        <p>Live statistics (possession, shots, cards)</p>
                        <a href="/api/v1/matches/1/stats" target="_blank">Try it →</a>
                    </div>
                    
                    <div class="endpoint">
                        <span class="badge badge-medium">12s updates</span>
                        <h3>GET /api/v1/global-stats</h3>
                        <p>Aggregate statistics across all matches</p>
                        <a href="/api/v1/global-stats" target="_blank">Try it →</a>
                    </div>
                    
                    <div class="endpoint">
                        <span class="badge badge-medium">8-25s updates</span>
                        <h3>GET /api/v1/events</h3>
                        <p>Recent match events (goals, cards, substitutions)</p>
                        <a href="/api/v1/events" target="_blank">Try it →</a>
                    </div>
                </div>
            </div>
            
            <div class="info-grid">
                <div class="info-section">
                    <h3>Features</h3>
                    <ul>
                        <li>✅ CORS enabled for all origins</li>
                        <li>✅ No authentication required</li>
                        <li>✅ No rate limiting</li>
                        <li>✅ Real Premier League & La Liga teams</li>
                        <li>✅ Realistic match progression</li>
                        <li>✅ Multiple update frequencies</li>
                    </ul>
                </div>
                
                <div class="info-section">
                    <h3>Use Cases</h3>
                    <ul>
                        <li>🔄 Testing real-time state management</li>
                        <li>📊 Dashboard development</li>
                        <li>⚡ WebSocket alternative testing</li>
                        <li>🎯 Performance optimization</li>
                        <li>🧪 Error handling scenarios</li>
                        <li>📱 Mobile app prototyping</li>
                    </ul>
                </div>
                
                <div class="info-section">
                    <h3>Data Sources</h3>
                    <ul>
                        <li>🏴󠁧󠁢󠁥󠁮󠁧󠁿 {{.PremierLeagueTeams}} Premier League teams</li>
                        <li>🇪🇸 {{.LaLigaTeams}} La Liga teams</li>
                        <li>🌍 {{.OtherTeams}} other European teams</li>
                        <li>👤 {{.TotalPlayers}} realistic player names</li>
                        <li>🏟️ Accurate stadium associations</li>
                        <li>🏆 Multiple competition types</li>
                    </ul>
                </div>
            </div>
            
            <div class="footer">
                <p>
                    Created for educational purposes to help developers test real-time applications.
                    <br>
                    All data is randomly generated and updates continuously.
                </p>
                
                <div class="version-info">
                    <span>Version 1.0.0</span>
                    <span>Last updated: {{.LastUpdated}}</span>
                    <a href="https://github.com/yourusername/matchpulse-api" target="_blank">GitHub Repository</a>
                </div>
            </div>
        </div>
    </div>
</body>
</html>`
