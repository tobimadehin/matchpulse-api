# ⚽ MatchPulse API

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Contributions Welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/tobimadehin/matchpulse-api/pulls)

> An advanced real-time football simulation API designed for testing dashboard applications, state management libraries, and real-time features. Features 90-second realistic matches, live player tracking, season management, and network condition simulation. Built by developers, for developers.

## 🎯 Why MatchPulse Exists

Real-world applications face challenges that static APIs can't simulate. MatchPulse provides:
- **Realistic temporal patterns**: Data updates at different frequencies like real sports APIs
- **Network condition simulation**: 95% blazing fast, 5% slow/timeout responses  
- **Complex state relationships**: Player characteristics, match events, season progression
- **Memory pressure testing**: Live player locations, audio-ready commentary, historical data
- **Lifecycle management**: Full season cycles with teardown and restart

## 🚀 Quick Start

### Try It Live
Before setting up locally, explore the enhanced API:
- **Demo API**: [https://matchpulse-api.onrender.com](https://matchpulse-api.onrender.com)
- **Live Matches**: [https://matchpulse-api.onrender.com/api/v1/matches](https://matchpulse-api.onrender.com/api/v1/matches)
- **Live Player Locations**: [https://matchpulse-api.onrender.com/api/v1/matches/1/locations](https://matchpulse-api.onrender.com/api/v1/matches/1/locations)

### Local Development

```bash
# Clone the repository
git clone https://github.com/tobimadehin/matchpulse-api.git
cd matchpulse-api

# Install dependencies
go mod download

# Start the server
go run main.go

# Visit http://localhost:8080 for interactive documentation
```

The API starts generating realistic match data immediately with:
- **90-second matches** with 15-second cooldown periods
- **Live player locations** updating every 2 seconds
- **Season progression** with automatic teardown and restart
- **10 seasons of historical data** maintained in memory

## 📊 Enhanced Data Flow & Simulation

MatchPulse runs a sophisticated multi-layered simulation:

### ⚡ Real-time Updates (2 seconds)
- Live player locations on the field (X,Y coordinates)
- Match events (goals, cards, substitutions)
- Audio-ready commentary with speed variations

### 🔄 Frequent Updates (5-8 seconds)  
- Match statistics (possession, shots, fouls)
- Player ratings during matches
- League table positions

### 📈 Moderate Updates (12-15 seconds)
- Global statistics and aggregations
- Season progress tracking
- Player season statistics

### 🏆 Lifecycle Events (Match/Season completion)
- Season winners and historical records
- Player of the season calculations
- Complete data reset and new season start

## 🎮 Advanced Features

### 🏃‍♂️ Live Player Tracking
Every match provides real-time player locations perfect for building live match visualizations:

```javascript
// Real-time player tracking
const fetchPlayerLocations = async (matchId) => {
  const response = await fetch(`/api/v1/matches/${matchId}/locations`);
  const data = await response.json();
  
  // data.locations contains array of {player_id, x, y, timestamp}
  // Perfect for canvas/WebGL visualizations
  updateField(data.locations);
};
```

### 📊 Player Characteristics System
Each player has permanent characteristics that affect match performance:

```json
{
  "characteristics": {
    "speed": 85,       // Affects position movement
    "shooting": 92,    // Influences goal probability  
    "passing": 78,     // Impacts assist likelihood
    "defending": 45,   // Defensive event frequency
    "physicality": 80, // Stamina and strength
    "mentality": 88,   // Performance under pressure
    "overall": 78      // Calculated average
  }
}
```

### 🏆 Season Management
Complete season lifecycle with historical tracking:

- **38 matches per season** with realistic scheduling
- **Automatic season progression** and table updates
- **Historical data** for last 10 seasons including:
  - Top scorer, top assists, most fouls
  - Player of the season (highest average rating)
  - Championship winners
  - Season statistics

### 🎵 Audio-Ready Commentary
Commentary system designed for streaming audio integration:

```json
{
  "text": "GOAL! Amazing strike finds the back of the net!",
  "audio_text": "GOOOOOAAAL! Amazing strike finds the back of the net!",
  "audio_speed": 1.2,
  "event_type": "GOAL"
}
```

## 🔧 API Reference

### Core Endpoints

| Endpoint | Purpose | Update Frequency | Best For Testing |
|----------|---------|------------------|------------------|
| `GET /api/v1/matches` | All live matches | 20-30 seconds | Basic state management |
| `GET /api/v1/matches/{id}/stats` | Live match statistics | 5-8 seconds | High-frequency updates |
| `GET /api/v1/global-stats` | Aggregate data | 12 seconds | Derived state |
| `GET /api/v1/players` | Player data with characteristics | Static | Player database |
| `GET /api/v1/teams` | Team information | Static | Team database |
| `GET /api/v1/matches/{id}/locations` | Live player positions | 2 seconds | Real-time visualizations |
| `GET /api/v1/matches/{id}/commentary` | Audio-ready commentary | Event-driven | Streaming features |
| `GET /api/v1/season/history` | Last 10 seasons data | Season completion | Historical analysis |
| `GET /api/v1/season/stats` | Current season progress | Match completion | Progress tracking |
| `GET /api/v1/league-table/{league}` | Real-time standings | Match completion | League standings |

## 🎮 How It Works - Complete Lifecycle

### Application Startup
```
1. Server Starts → Initialize Teams & Players → Generate Player Characteristics
2. Create League Tables → Launch Simulation Engines → Create First Match
```

### Match Lifecycle (90 seconds real-time)
```
⚽ KICKOFF
├── Every 2 seconds: Update player positions
├── Every 2 seconds: Check for events (goals, cards, etc.)
├── Minute 45: Halftime break
├── Minute 46: Second half begins  
└── Minute 90: Full time whistle
    ├── Calculate player ratings
    ├── Update league table
    ├── 15-second cooldown
    └── Create next match
```

### Season Management
```
📅 Season (38 matches) → Track Statistics → End Season → Crown Champions → Reset & Start New Season
```

This helps you build resilient applications that handle real-world network conditions.

## 🏗️ Contributing

MatchPulse thrives on community contributions. Here's how you can help:

### 🐛 Found a Bug?

1. **Search existing issues** to avoid duplicates
2. **Create a detailed issue** with:
   - What you expected vs. what happened
   - Steps to reproduce
   - Your environment (Go version, OS)

### 💡 Feature Ideas

We're especially interested in:
- **Additional sports simulations** (basketball, tennis, etc.)
- **Enhanced player behavior models** 
- **More realistic match simulation**
- **Advanced audio streaming features**
- **Performance optimizations**

### 🔧 Easy Contributions

- Add teams from other leagues (MLS, Serie A, Bundesliga)
- Expand player name diversity and nationalities
- Improve stadium accuracy and capacity data
- Add new weather conditions and their effects
- Create additional formation types

### 🚀 Advanced Contributions

- Implement WebSocket support for real-time updates
- Add configurable simulation parameters via API
- Create injury and transfer systems
- Implement referee decisions and VAR
- Add crowd reaction simulation

## 📊 Performance & Resource Usage

MatchPulse is optimized for development use:

- **Memory**: ~50MB baseline, +10MB per active season
- **CPU**: Minimal impact with smart goroutine management  
- **Network**: Optimized JSON responses with compression
- **Persistence**: In-memory only, perfect for testing

## 🤝 Community & Support

### Getting Help
- **Documentation Issues**: Search existing issues or create new ones
- **Usage Questions**: Check examples in source code
- **Feature Requests**: Use GitHub issues with detailed use cases

### Showcase Your Project
Built something awesome with MatchPulse? We'd love to feature it!
- Open an issue with the "showcase" label
- Tag a maintainer on X [@_techcyborg](https://x.com/_techcyborg)

## 📄 License

MatchPulse is open source software licensed under the [MIT License](LICENSE). You can:
- Use it commercially or personally
- Modify it for your needs  
- Distribute your modifications
- Include it in proprietary software

The only requirement is including the original license notice.

## 🙏 Contributors

MatchPulse exists because of contributions from developers worldwide:

<!-- Contributors will be automatically updated -->
<a href="https://github.com/tobimadehin/matchpulse-api/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=tobimadehin/matchpulse-api" />
</a>

---

<div align="center">

**Built with ❤️ by the developer community**

**Perfect for testing React, Vue, Angular, Svelte, Unity, Unreal Engine, Godot**

[Report Bug](https://github.com/tobimadehin/matchpulse-api/issues) · 
[Request Feature](https://github.com/tobimadehin/matchpulse-api/issues) · 
[Join Discussion](https://github.com/tobimadehin/matchpulse-api/discussions)

⭐ **Star this repo** if MatchPulse helps your development!

</div>