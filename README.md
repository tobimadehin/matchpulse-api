# ⚽ MatchPulse API

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Contributions Welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/tobimadehin/matchpulse-api/pulls)

> A real-time football API simulation specifically for testing dashboard applications, state management libraries, and real-time features. Built by developers, for developers.

## 🎯 Why MatchPulse Exists

When you're building real-time applications, testing with static JSON files or overly simple APIs doesn't reveal the challenges you'll face in production. Real applications deal with data that updates at different frequencies, network failures, and complex state relationships. 

## 🚀 Quick Start

### Try It Live
Before setting up locally, you can explore MatchPulse immediately:
- **Demo API**: [https://matchpulse-api.onrender.com](https://matchpulse-api.onrender.com)
- **Live Matches**: [https://matchpulse-api.onrender.com/api/v1/matches](https://matchpulse-api.onrender.com/api/v1/matches)

### Local Development

```bash
# Clone the repository
git clone https://github.com/tobimadehin/matchpulse-api.git
cd matchpulse-api

# Install dependencies
go mod download

# Start the server
go run main.go data.go templates.go

# Visit http://localhost:8080 for interactive documentation
```

That's it! The API will start generating realistic match data immediately.

## 📊 Understanding the Data Flow

MatchPulse runs multiple background processes that simulate how different types of data update in real applications:

**Fast Updates (2-5 seconds)**
- Match statistics (possession, shots, fouls)

**Medium Updates (8-12 seconds)**  
- Global statistics and trending data

**Slow Updates (20-45 seconds)**
- Match scores and core information

**Event-Driven Updates (Irregular)**
- Goals, cards, substitutions


## 🔧 API Reference

### Core Endpoints

| Endpoint | Purpose | Update Frequency | Best For Testing |
|----------|---------|------------------|------------------|
| `GET /api/v1/matches` | All live matches | 20-45 seconds | Slow state updates |
| `GET /api/v1/matches/{id}/stats` | Live match statistics | 2-5 seconds | High-frequency streams |
| `GET /api/v1/global-stats` | Aggregate data | 12 seconds | Derived state |
| `GET /api/v1/events` | Recent match events | 8-25 seconds | Event-driven updates |
| `GET /api/v1/health` | API status | Real-time | Service monitoring |

### Example Usage

Understanding how to use MatchPulse effectively starts with recognizing which endpoint tests which aspect of your application:

**Testing Single Data Streams**
```javascript
// Start simple - test basic state management
useEffect(() => {
  const fetchMatches = async () => {
    const response = await fetch('/api/v1/matches');
    const data = await response.json();
    setMatches(data.matches);
  };

  fetchMatches();
  const interval = setInterval(fetchMatches, 30000);
  return () => clearInterval(interval);
}, []);
```

**Testing Multiple Concurrent Streams**
```javascript
// Advanced - test handling multiple update frequencies
useEffect(() => {
  const streams = [
    { endpoint: '/api/v1/matches', interval: 25000, handler: setMatches },
    { endpoint: '/api/v1/global-stats', interval: 12000, handler: setStats },
    { endpoint: '/api/v1/events', interval: 15000, handler: setEvents }
  ];

  const intervals = streams.map(({ endpoint, interval, handler }) => 
    setInterval(async () => {
      const response = await fetch(endpoint);
      const data = await response.json();
      handler(data);
    }, interval)
  );

  return () => intervals.forEach(clearInterval);
}, []);
```

## 🏗️ Contributing

MatchPulse thrives because developers like you identify gaps and contribute improvements. Here's how you can help make it better:

### 🐛 Found a Bug?

1. **Search existing issues** to avoid duplicates
2. **Create a detailed issue** with:
   - What you expected to happen
   - What actually happened
   - Steps to reproduce
   - Your environment (Go version, OS)

### 💡 Have a Feature Idea?

We love hearing how MatchPulse could better serve your testing needs:

1. **Check the roadmap** below to see if it's already planned
2. **Open a feature request** describing:
   - The problem you're trying to solve
   - How this feature would help your testing
   - Any implementation ideas you have

### 🔧 Want to Contribute Code?

**Easy First Contributions:**
- Add teams from other leagues (MLS, Serie A, Bundesliga etc.)
- Expand player name diversity
- Improve stadium accuracy
- Add new competition types

**More Advanced Contributions:**
- Implement WebSocket support for real-time updates
- Add configurable update frequencies
- Create new endpoint types (transfers, injuries, etc.)
- Improve network simulation realism

**Getting Started:**
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes with clear commit messages
4. Add tests if applicable
5. Submit a pull request

### 📋 Development Guidelines

**Code Style**
- Use `gofmt` for formatting
- Write descriptive commit messages
- Add comments explaining complex logic
- Keep functions focused and readable

**Data Quality**
- Verify team-stadium associations are correct
- Ensure realistic statistical relationships
- Test that match generation makes contextual sense

**Testing Additions**
- Test new endpoints thoroughly
- Verify thread safety for concurrent updates
- Ensure memory usage remains stable

## 🗺️ Roadmap
- **WebSocket Support**: Real-time push notifications for events
- **Configuration API**: Allow customizing update frequencies
- **More Leagues**: Add MLS, Liga MX, and Championship teams


### Want to Influence the Roadmap?
Your feedback shapes MatchPulse's direction. The most requested features get prioritized, and the best ideas often come from developers actively using the API. Share your thoughts in issues or discussions!

## 🤝 Community

### Getting Help
- **Documentation Issues**: Search for existing issues or open one with the "documentation" label

### Showcase Your Project
Built something cool with MatchPulse? We'd love to feature it! Share your projects by:
- Opening an issue with the "showcase" label
- Tag a maintainer on X [@_techcyborg](https://x.com/_techcyborg)

### Recognition
Contributors who help improve MatchPulse get recognition in our:
- Contributors section (below)
- Release notes for their contributions
- Featured contributor spotlight (monthly)


## 📄 License

MatchPulse is open source software licensed under the [MIT License](LICENSE). This means you can:
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

[Report Bug](https://github.com/tobimadehin/matchpulse-api/issues) · 
[Request Feature](https://github.com/tobimadehin/matchpulse-api/issues) · 
[Join Discussion](https://github.com/tobimadehin/matchpulse-api/discussions)

⭐ **Star this repo** if MatchPulse helps your development!

</div>