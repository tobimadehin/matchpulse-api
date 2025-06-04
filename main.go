package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// String constants for optimization
const (
	// Match statuses
	StatusLive      = "LIVE"
	StatusHalftime  = "HALFTIME"
	StatusFinished  = "FINISHED"
	StatusPostponed = "POSTPONED"
	
	// Leagues
	LeaguePremier = "Premier League"
	LeagueLaLiga  = "La Liga"
	
	// Positions
	PosGK  = "GK"
	PosCB  = "CB"
	PosLB  = "LB"
	PosRB  = "RB"
	PosCDM = "CDM"
	PosCM  = "CM"
	PosCAM = "CAM"
	PosLW  = "LW"
	PosRW  = "RW"
	PosST  = "ST"
	
	// Nationalities
	NatEngland = "England"
	NatSpain   = "Spain"
	NatItaly   = "Italy"
	
	// Event types
	EventGoal         = "GOAL"
	EventCard         = "CARD"
	EventSubstitution = "SUBSTITUTION"
	EventCommentary   = "COMMENTARY"
	
	// Formations
	Formation442  = "4-4-2"
	Formation433  = "4-3-3"
	Formation352  = "3-5-2"
	Formation4231 = "4-2-3-1"
	Formation532  = "5-3-2"
	
	// Weather conditions
	WeatherClear        = "Clear"
	WeatherCloudy       = "Cloudy"
	WeatherLightRain    = "Light Rain"
	WeatherOvercast     = "Overcast"
	WeatherSunny        = "Sunny"
	WeatherPartlyCloudy = "Partly Cloudy"
	
	// Form results
	FormWin  = "W"
	FormLoss = "L"
	FormDraw = "D"
)

var (
	// Formations slice
	formations = []string{Formation442, Formation433, Formation352, Formation4231, Formation532}
	
	// Weather conditions slice
	weatherConditions = []string{WeatherClear, WeatherCloudy, WeatherLightRain, WeatherOvercast, WeatherSunny, WeatherPartlyCloudy}
	
	// Form results slice
	formResults = []string{FormWin, FormLoss, FormDraw}
)

// Enhanced data structures for comprehensive sports API testing
type Match struct {
	ID           int       `json:"id"`
	HomeTeam     TeamInfo  `json:"home_team"`
	AwayTeam     TeamInfo  `json:"away_team"`
	HomeScore    int       `json:"home_score"`
	AwayScore    int       `json:"away_score"`
	Minute       int       `json:"minute"`
	Status       string    `json:"status"` // LIVE, HALFTIME, FINISHED, POSTPONED
	Competition  string    `json:"competition"`
	LastUpdate   time.Time `json:"last_update"`
	Venue        string    `json:"venue"`
	Attendance   int       `json:"attendance"`
	Weather      string    `json:"weather"`
	Temperature  int       `json:"temperature"`
	HomeFormation string   `json:"home_formation"`
	AwayFormation string   `json:"away_formation"`
}

type TeamInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ShortName string `json:"short_name"`
	LogoURL  string `json:"logo_url"`
	Stadium  string `json:"stadium"`
	Founded  int    `json:"founded"`
	Manager  string `json:"manager"`
	League   string `json:"league"`
}

type Player struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Position    string  `json:"position"`
	Number      int     `json:"number"`
	Age         int     `json:"age"`
	Nationality string  `json:"nationality"`
	AvatarURL   string  `json:"avatar_url"`
	TeamID      int     `json:"team_id"`
	Goals       int     `json:"goals"`
	Assists     int     `json:"assists"`
	YellowCards int     `json:"yellow_cards"`
	RedCards    int     `json:"red_cards"`
	Appearances int     `json:"appearances"`
	MarketValue int     `json:"market_value"` // in millions
	LastUpdate  time.Time `json:"last_update"`
}

type MatchStats struct {
	MatchID           int       `json:"match_id"`
	HomePossession    int       `json:"home_possession"`
	AwayPossession    int       `json:"away_possession"`
	HomeShots         int       `json:"home_shots"`
	AwayShots         int       `json:"away_shots"`
	HomeShotsOnTarget int       `json:"home_shots_on_target"`
	AwayShotsOnTarget int       `json:"away_shots_on_target"`
	HomeCorners       int       `json:"home_corners"`
	AwayCorners       int       `json:"away_corners"`
	HomeFouls         int       `json:"home_fouls"`
	AwayFouls         int       `json:"away_fouls"`
	HomeYellowCards   int       `json:"home_yellow_cards"`
	AwayYellowCards   int       `json:"away_yellow_cards"`
	HomeRedCards      int       `json:"home_red_cards"`
	AwayRedCards      int       `json:"away_red_cards"`
	HomePasses        int       `json:"home_passes"`
	AwayPasses        int       `json:"away_passes"`
	HomePassAccuracy  float64   `json:"home_pass_accuracy"`
	AwayPassAccuracy  float64   `json:"away_pass_accuracy"`
	LastUpdate        time.Time `json:"last_update"`
}

type LeagueTable struct {
	Position     int     `json:"position"`
	Team         TeamInfo `json:"team"`
	Played       int     `json:"played"`
	Won          int     `json:"won"`
	Drawn        int     `json:"drawn"`
	Lost         int     `json:"lost"`
	GoalsFor     int     `json:"goals_for"`
	GoalsAgainst int     `json:"goals_against"`
	GoalDiff     int     `json:"goal_difference"`
	Points       int     `json:"points"`
	Form         []string `json:"form"` // Last 5 results: W, L, D
	LastUpdate   time.Time `json:"last_update"`
}

type LiveCommentary struct {
	ID         int       `json:"id"`
	MatchID    int       `json:"match_id"`
	Minute     int       `json:"minute"`
	Text       string    `json:"text"`
	EventType  string    `json:"event_type"` // GOAL, CARD, SUBSTITUTION, COMMENTARY
	Player     *Player   `json:"player,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type GlobalStats struct {
	TotalMatches    int       `json:"total_matches"`
	TotalGoals      int       `json:"total_goals"`
	AverageGoals    float64   `json:"average_goals"`
	MostGoalsMatch  int       `json:"most_goals_match_id"`
	ActiveViewers   int       `json:"active_viewers"`
	TopScorer       Player    `json:"top_scorer"`
	MostActiveMatch int       `json:"most_active_match_id"`
	LastUpdate      time.Time `json:"last_update"`
}

// Thread-safe storage with enhanced data structures
var (
	matches          = make(map[int]*Match)
	matchStats       = make(map[int]*MatchStats)
	players          = make(map[int]*Player)
	teams            = make(map[int]*TeamInfo)
	leagueTables     = make(map[string][]*LeagueTable) // keyed by league name
	liveCommentary   = make(map[int][]*LiveCommentary) // keyed by match ID
	globalStats      = &GlobalStats{}
	commentaryCounter = 0
	mutex            = &sync.RWMutex{}
	version          = "1.0.0" // fallback version
)

// Enhanced team data with IDs and more information
var teamData = []struct {
	ID        int
	Name      string
	ShortName string
	Stadium   string
	League    string
	Manager   string
	Founded   int
	Country   string
}{
	// Premier League teams
	{1, "Arsenal FC", "ARS", "Emirates Stadium", LeaguePremier, "Mikel Arteta", 1886, "England"},
	{2, "Chelsea FC", "CHE", "Stamford Bridge", LeaguePremier, "Mauricio Pochettino", 1905, "England"},
	{3, "Liverpool FC", "LIV", "Anfield", LeaguePremier, "Jürgen Klopp", 1892, "England"},
	{4, "Manchester City FC", "MCI", "Etihad Stadium", LeaguePremier, "Pep Guardiola", 1880, "England"},
	{5, "Manchester United", "MUN", "Old Trafford", LeaguePremier, "Erik ten Hag", 1878, "England"},
	{6, "Tottenham Hotspur", "TOT", "Tottenham Hotspur Stadium", LeaguePremier, "Ange Postecoglou", 1882, "England"},
	{7, "Newcastle United", "NEW", "St. James' Park", LeaguePremier, "Eddie Howe", 1892, "England"},
	{8, "Brighton & Hove Albion", "BHA", "American Express Stadium", LeaguePremier, "Roberto De Zerbi", 1901, "England"},
	{9, "Aston Villa", "AVL", "Villa Park", LeaguePremier, "Unai Emery", 1874, "England"},
	{10, "West Ham United", "WHU", "London Stadium", LeaguePremier, "David Moyes", 1895, "England"},

	// La Liga teams
	{11, "Real Madrid", "RMA", "Santiago Bernabéu", LeagueLaLiga, "Carlo Ancelotti", 1902, "Spain"},
	{12, "FC Barcelona", "BAR", "Camp Nou", LeagueLaLiga, "Xavi Hernández", 1899, "Spain"},
	{13, "Atlético de Madrid", "ATM", "Metropolitano Stadium", LeagueLaLiga, "Diego Simeone", 1903, "Spain"},
	{14, "Athletic Bilbao", "ATH", "San Mamés", LeagueLaLiga, "Ernesto Valverde", 1898, "Spain"},
	{15, "Real Sociedad", "RSO", "Reale Arena", LeagueLaLiga, "Imanol Alguacil", 1909, "Spain"},
	{16, "Villarreal CF", "VIL", "Estadio de la Cerámica", LeagueLaLiga, "Marcelino García", 1923, "Spain"},
	{17, "Sevilla FC", "SEV", "Ramón Sánchez-Pizjuán Stadium", LeagueLaLiga, "José Luis Mendilibar", 1890, "Spain"},
	{18, "Real Betis Balompié", "BET", "Benito Villamarín Stadium", LeagueLaLiga, "Manuel Pellegrini", 1907, "Spain"},
	{19, "Valencia CF", "VAL", "Mestalla", LeagueLaLiga, "Rubén Baraja", 1919, "Spain"},
	{20, "Getafe CF", "GET", "Coliseum Alfonso Pérez", LeagueLaLiga, "José Bordalás", 1983, "Spain"},
}

// Player names with positions for realistic squads
var playerNames = []struct {
	Name        string
	Position    string
	Nationality string
}{
	{"Marcus Johnson", PosGK, NatEngland}, {"David Silva", PosGK, NatSpain}, {"Antonio López", PosGK, NatSpain},
	{"James Wilson", PosCB, NatEngland}, {"Carlos Hernández", PosCB, NatSpain}, {"Marco Rossi", PosCB, NatItaly},
	{"Alex Thompson", PosCB, NatEngland}, {"Diego Martínez", PosCB, NatSpain}, {"Francesco Romano", PosCB, NatItaly},
	{"Luke Roberts", PosLB, NatEngland}, {"Pablo García", PosLB, NatSpain}, {"Andrea Colombo", PosLB, NatItaly},
	{"Ryan Davis", PosRB, NatEngland}, {"Miguel Rodríguez", PosRB, NatSpain}, {"Luca Ferrari", PosRB, NatItaly},
	{"Jordan Smith", PosCDM, NatEngland}, {"Alejandro Pérez", PosCDM, NatSpain}, {"Matteo Conti", PosCDM, NatItaly},
	{"Oliver Brown", PosCM, NatEngland}, {"Francisco Ruiz", PosCM, NatSpain}, {"Davide Ricci", PosCM, NatItaly},
	{"Connor Wilson", PosCAM, NatEngland}, {"Sergio González", PosCAM, NatSpain}, {"Lorenzo Greco", PosCAM, NatItaly},
	{"Mason Taylor", PosLW, NatEngland}, {"Alberto Moreno", PosLW, NatSpain}, {"Alessandro Bruno", PosLW, NatItaly},
	{"Tyler Anderson", PosRW, NatEngland}, {"Rafael Jiménez", PosRW, NatSpain}, {"Simone Gallo", PosRW, NatItaly},
	{"Harry Clarke", PosST, NatEngland}, {"Fernando Torres", PosST, NatSpain}, {"Giovanni De Luca", PosST, NatItaly},
	{"Charlie Evans", PosST, NatEngland}, {"Adrián Vázquez", PosST, NatSpain}, {"Emilio Mancini", PosST, NatItaly},
}

var commentaryTemplates = []string{
	"Great save by the goalkeeper!",
	"{player} makes a brilliant run down the wing",
	"Corner kick for {team}",
	"Yellow card shown to {player}",
	"GOAL! {player} finds the back of the net!",
	"Close! {player} just wide of the target",
	"Substitution: {player} comes on",
	"Free kick awarded to {team}",
	"{player} with a crucial tackle",
	"The referee stops play for a {team} throw-in",
	"Penalty appeal waved away",
	"{team} on the attack here",
	"Excellent cross from {player}",
	"The crowd is on their feet!",
	"Weather conditions affecting play",
}

func init() {
	rand.Seed(time.Now().UnixNano())
	
	// Load version from file
	loadVersion()
	
	// Initialize teams
	initializeTeams()
	
	// Initialize players for each team
	initializePlayers()
	
	// Create realistic matches
	createRealisticMatches()
	
	// Initialize league tables
	initializeLeagueTables()
	
	updateGlobalStats()

	// Start background processes with different update frequencies
	go updateMatchScores()         // Every 20-45 seconds
	go updateMatchStats()          // Every 2-5 seconds  
	go updateGlobalStatsLoop()     // Every 12 seconds
	go generateLiveCommentary()    // Every 3-8 seconds
	go updatePlayerStats()         // Every 60 seconds
	go updateLeagueTables()        // Every 45 seconds
	go simulateMatchProgression()  // Every 30-60 seconds
}

func loadVersion() {
	if data, err := os.ReadFile("version.txt"); err == nil {
		version = strings.TrimSpace(string(data))
	}
}

func initializeTeams() {
	mutex.Lock()
	defer mutex.Unlock()
	
	for _, teamInfo := range teamData {
		// Determine the correct subfolder based on country and league
		var leagueFolder string
		switch teamInfo.Country {
		case "England":
			leagueFolder = "England%20-%20Premier%20League"
		case "Spain":
			leagueFolder = "Spain%20-%20LaLiga"
		default:
			leagueFolder = "Unknown"
		}
	
		// URL-encode team name
		teamName := strings.ReplaceAll(teamInfo.Name, " ", "%20")
	
		teams[teamInfo.ID] = &TeamInfo{
			ID:        teamInfo.ID,
			Name:      teamInfo.Name,
			ShortName: teamInfo.ShortName,
			LogoURL:   fmt.Sprintf("https://raw.githubusercontent.com/luukhopman/football-logos/master/logos/%s/%s.png", leagueFolder, teamName),
			Stadium:   teamInfo.Stadium,
			Founded:   teamInfo.Founded,
			Manager:   teamInfo.Manager,
			League:    teamInfo.League,
		}
	}
}

func initializePlayers() {
	mutex.Lock()
	defer mutex.Unlock()
	
	playerID := 1
	for _, team := range teams {
		// Create 25 players per team (realistic squad size)
		for i := 0; i < 25; i++ {
			playerTemplate := playerNames[rand.Intn(len(playerNames))]
			
			// Generate unique name by adding number if needed
			name := playerTemplate.Name
			if i > len(playerNames)-1 {
				name = fmt.Sprintf("%s %d", playerTemplate.Name, i)
			}
			
			players[playerID] = &Player{
				ID:          playerID,
				Name:        name,
				Position:    playerTemplate.Position,
				Number:      i + 1,
				Age:         18 + rand.Intn(20), // Age between 18-38
				Nationality: playerTemplate.Nationality,
				AvatarURL:   fmt.Sprintf("https://i.pravatar.cc/150?img=%d", (playerID%70)+1), // Cycling through available avatars
				TeamID:      team.ID,
				Goals:       rand.Intn(25),
				Assists:     rand.Intn(15),
				YellowCards: rand.Intn(8),
				RedCards:    rand.Intn(2),
				Appearances: rand.Intn(35) + 5,
				MarketValue: rand.Intn(100) + 5, // 5-105 million
				LastUpdate:  time.Now(),
			}
			playerID++
		}
	}
}

func createRealisticMatches() {
	mutex.Lock()
	defer mutex.Unlock()
	
	matchID := 1
	
	// Create matches within the same league for realism
	leagues := []string{LeaguePremier, LeagueLaLiga}
	
	for _, league := range leagues {
		leagueTeams := getTeamsByLeague(league)
		if len(leagueTeams) >= 2 {
			// Create 2-3 matches per league
			for i := 0; i < 3 && i < len(leagueTeams)-1; i++ {
				homeTeam := leagueTeams[rand.Intn(len(leagueTeams))]
				awayTeam := leagueTeams[rand.Intn(len(leagueTeams))]
				
				// Ensure different teams
				for awayTeam.ID == homeTeam.ID {
					awayTeam = leagueTeams[rand.Intn(len(leagueTeams))]
				}
				
				match := generateRealisticMatch(matchID, homeTeam, awayTeam, league)
				matches[matchID] = match
				matchStats[matchID] = generateRandomMatchStats(matchID)
				liveCommentary[matchID] = []*LiveCommentary{}
				matchID++
			}
		}
	}
}

func getTeamsByLeague(league string) []*TeamInfo {
	var result []*TeamInfo
	for _, team := range teams {
		if team.League == league {
			result = append(result, team)
		}
	}
	return result
}

func generateRealisticMatch(id int, homeTeam, awayTeam *TeamInfo, competition string) *Match {
	minute := rand.Intn(85) + 1
	status := StatusLive
	if minute > 45 && minute < 50 && rand.Float32() < 0.2 {
		status = StatusHalftime
	}
	
	return &Match{
		ID:           id,
		HomeTeam:     *homeTeam,
		AwayTeam:     *awayTeam,
		HomeScore:    rand.Intn(4),
		AwayScore:    rand.Intn(4),
		Minute:       minute,
		Status:       status,
		Competition:  competition,
		Venue:        homeTeam.Stadium,
		Attendance:   rand.Intn(80000) + 20000,
		Weather:      getRandomWeather(),
		Temperature:  rand.Intn(25) + 5, // 5-30°C
		HomeFormation: formations[rand.Intn(len(formations))],
		AwayFormation: formations[rand.Intn(len(formations))],
		LastUpdate:   time.Now(),
	}
}

func getRandomWeather() string {
	return weatherConditions[rand.Intn(len(weatherConditions))]
}

func generateRandomMatchStats(matchID int) *MatchStats {
	homePoss := rand.Intn(40) + 30
	awayPoss := 100 - homePoss
	
	homeShots := rand.Intn(20) + 5
	awayShots := rand.Intn(20) + 5
	
	return &MatchStats{
		MatchID:           matchID,
		HomePossession:    homePoss,
		AwayPossession:    awayPoss,
		HomeShots:         homeShots,
		AwayShots:         awayShots,
		HomeShotsOnTarget: min(homeShots, rand.Intn(homeShots/2+1)),
		AwayShotsOnTarget: min(awayShots, rand.Intn(awayShots/2+1)),
		HomeCorners:       rand.Intn(12),
		AwayCorners:       rand.Intn(12),
		HomeFouls:         rand.Intn(20) + 5,
		AwayFouls:         rand.Intn(20) + 5,
		HomeYellowCards:   rand.Intn(4),
		AwayYellowCards:   rand.Intn(4),
		HomeRedCards:      rand.Intn(2),
		AwayRedCards:      rand.Intn(2),
		HomePasses:        rand.Intn(400) + 200,
		AwayPasses:        rand.Intn(400) + 200,
		HomePassAccuracy:  70.0 + rand.Float64()*25.0, // 70-95%
		AwayPassAccuracy:  70.0 + rand.Float64()*25.0,
		LastUpdate:        time.Now(),
	}
}

func initializeLeagueTables() {
	mutex.Lock()
	defer mutex.Unlock()
	
	leagues := []string{LeaguePremier, LeagueLaLiga}
	
	for _, league := range leagues {
		leagueTeams := getTeamsByLeague(league)
		var table []*LeagueTable
		
		for i, team := range leagueTeams {
			form := generateRandomForm()
			played := rand.Intn(15) + 20 // 20-35 games played
			won := rand.Intn(played)
			lost := rand.Intn(played - won)
			drawn := played - won - lost
			goalsFor := won*2 + drawn + rand.Intn(20)
			goalsAgainst := lost*2 + rand.Intn(15)
			
			table = append(table, &LeagueTable{
				Position:     i + 1,
				Team:         *team,
				Played:       played,
				Won:          won,
				Drawn:        drawn,
				Lost:         lost,
				GoalsFor:     goalsFor,
				GoalsAgainst: goalsAgainst,
				GoalDiff:     goalsFor - goalsAgainst,
				Points:       won*3 + drawn,
				Form:         form,
				LastUpdate:   time.Now(),
			})
		}
		
		leagueTables[league] = table
	}
}

func generateRandomForm() []string {
	var form []string
	for i := 0; i < 5; i++ {
		form = append(form, formResults[rand.Intn(len(formResults))])
	}
	return form
}

// Background update functions
func updateMatchScores() {
	for {
		time.Sleep(time.Duration(20+rand.Intn(25)) * time.Second)

		mutex.Lock()
		for _, match := range matches {
			if match.Status != StatusLive {
				continue
			}

			if rand.Float32() < 0.4 {
				match.Minute = min(90, match.Minute+rand.Intn(2)+1)
				match.LastUpdate = time.Now()
			}

			if rand.Float32() < 0.08 {
				if rand.Float32() < 0.5 {
					match.HomeScore++
				} else {
					match.AwayScore++
				}
				match.LastUpdate = time.Now()
				
				// Add goal commentary
				addGoalCommentary(match)
			}
		}
		mutex.Unlock()
	}
}

func addGoalCommentary(match *Match) {
	commentaryCounter++
	scorer := getRandomPlayerFromTeam(match.HomeTeam.ID)
	if rand.Float32() < 0.5 {
		scorer = getRandomPlayerFromTeam(match.AwayTeam.ID)
	}
	
	commentary := &LiveCommentary{
		ID:        commentaryCounter,
		MatchID:   match.ID,
		Minute:    match.Minute,
		Text:      fmt.Sprintf("GOAL! %s finds the back of the net! What a finish!", scorer.Name),
		EventType: EventGoal,
		Player:    scorer,
		Timestamp: time.Now(),
	}
	
	if liveCommentary[match.ID] == nil {
		liveCommentary[match.ID] = []*LiveCommentary{}
	}
	
	liveCommentary[match.ID] = append([]*LiveCommentary{commentary}, liveCommentary[match.ID]...)
	
	// Keep only last 20 commentary items per match
	if len(liveCommentary[match.ID]) > 20 {
		liveCommentary[match.ID] = liveCommentary[match.ID][:20]
	}
}

func getRandomPlayerFromTeam(teamID int) *Player {
	var teamPlayers []*Player
	for _, player := range players {
		if player.TeamID == teamID {
			teamPlayers = append(teamPlayers, player)
		}
	}
	if len(teamPlayers) > 0 {
		return teamPlayers[rand.Intn(len(teamPlayers))]
	}
	return &Player{Name: "Unknown Player"}
}

func updateMatchStats() {
	for {
		time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)

		mutex.Lock()
		for id, stats := range matchStats {
			if matches[id].Status != StatusLive {
				continue
			}

			possessionChange := rand.Intn(8) - 4
			stats.HomePossession = max(15, min(85, stats.HomePossession+possessionChange))
			stats.AwayPossession = 100 - stats.HomePossession

			if rand.Float32() < 0.3 {
				if rand.Float32() < 0.5 {
					stats.HomeShots++
				} else {
					stats.AwayShots++
				}
			}

			stats.LastUpdate = time.Now()
		}
		mutex.Unlock()
	}
}

func generateLiveCommentary() {
	for {
		time.Sleep(time.Duration(3+rand.Intn(5)) * time.Second)

		mutex.Lock()
		for matchID, match := range matches {
			if match.Status != StatusLive || rand.Float32() > 0.4 {
				continue
			}

			commentaryCounter++
			template := commentaryTemplates[rand.Intn(len(commentaryTemplates))]
			
			// Replace placeholders
			text := template
			if strings.Contains(template, "{player}") {
				player := getRandomPlayerFromTeam(match.HomeTeam.ID)
				if rand.Float32() < 0.5 {
					player = getRandomPlayerFromTeam(match.AwayTeam.ID)
				}
				text = strings.ReplaceAll(text, "{player}", player.Name)
			}
			if strings.Contains(template, "{team}") {
				team := match.HomeTeam.Name
				if rand.Float32() < 0.5 {
					team = match.AwayTeam.Name
				}
				text = strings.ReplaceAll(text, "{team}", team)
			}

			commentary := &LiveCommentary{
				ID:        commentaryCounter,
				MatchID:   matchID,
				Minute:    match.Minute,
				Text:      text,
				EventType: EventCommentary,
				Timestamp: time.Now(),
			}

			if liveCommentary[matchID] == nil {
				liveCommentary[matchID] = []*LiveCommentary{}
			}

			liveCommentary[matchID] = append([]*LiveCommentary{commentary}, liveCommentary[matchID]...)

			if len(liveCommentary[matchID]) > 20 {
				liveCommentary[matchID] = liveCommentary[matchID][:20]
			}
		}
		mutex.Unlock()
	}
}

func updatePlayerStats() {
	for {
		time.Sleep(60 * time.Second)

		mutex.Lock()
		for _, player := range players {
			if rand.Float32() < 0.1 { // 10% chance to update any player
				if rand.Float32() < 0.5 {
					player.Goals++
				} else {
					player.Assists++
				}
				player.LastUpdate = time.Now()
			}
		}
		mutex.Unlock()
	}
}

func updateLeagueTables() {
	for {
		time.Sleep(45 * time.Second)

		mutex.Lock()
		for _, table := range leagueTables {
			for _, entry := range table {
				if rand.Float32() < 0.1 { // Small chance to update table position
					entry.LastUpdate = time.Now()
				}
			}
		}
		mutex.Unlock()
	}
}

func simulateMatchProgression() {
	for {
		time.Sleep(time.Duration(30+rand.Intn(30)) * time.Second)

		mutex.Lock()
		for id, match := range matches {
			if match.Minute >= 45 && match.Minute < 50 && match.Status == StatusLive && rand.Float32() < 0.3 {
				match.Status = StatusHalftime
				match.LastUpdate = time.Now()
			} else if match.Status == StatusHalftime && rand.Float32() < 0.4 {
				match.Status = StatusLive
				match.Minute = 46 + rand.Intn(5)
				match.LastUpdate = time.Now()
			} else if match.Minute >= 90 && rand.Float32() < 0.2 {
				match.Status = StatusFinished
				match.LastUpdate = time.Now()
				replaceFinishedMatch(id)
			}
		}
		mutex.Unlock()
	}
}

func replaceFinishedMatch(matchID int) {
	leagues := []string{LeaguePremier, LeagueLaLiga}
	league := leagues[rand.Intn(len(leagues))]
	leagueTeams := getTeamsByLeague(league)
	
	if len(leagueTeams) >= 2 {
		homeTeam := leagueTeams[rand.Intn(len(leagueTeams))]
		awayTeam := leagueTeams[rand.Intn(len(leagueTeams))]
		
		for awayTeam.ID == homeTeam.ID {
			awayTeam = leagueTeams[rand.Intn(len(leagueTeams))]
		}
		
		newMatch := generateRealisticMatch(matchID, homeTeam, awayTeam, league)
		matches[matchID] = newMatch
		matchStats[matchID] = generateRandomMatchStats(matchID)
		liveCommentary[matchID] = []*LiveCommentary{}
	}
}

func updateGlobalStatsLoop() {
	for {
		time.Sleep(12 * time.Second)
		updateGlobalStats()
	}
}

func updateGlobalStats() {
	mutex.Lock()
	defer mutex.Unlock()

	totalGoals := 0
	maxGoals := 0
	maxGoalsMatchID := 0
	var topScorer *Player

	for id, match := range matches {
		if match.Status == StatusLive {
			goals := match.HomeScore + match.AwayScore
			totalGoals += goals
			if goals > maxGoals {
				maxGoals = goals
				maxGoalsMatchID = id
			}
		}
	}

	// Find top scorer
	maxGoals = 0
	for _, player := range players {
		if player.Goals > maxGoals {
			maxGoals = player.Goals
			topScorer = player
		}
	}

	if topScorer == nil {
		topScorer = &Player{Name: "Unknown", Goals: 0}
	}

	liveMatches := len(matches)
	globalStats.TotalMatches = liveMatches
	globalStats.TotalGoals = totalGoals
	if liveMatches > 0 {
		globalStats.AverageGoals = float64(totalGoals) / float64(liveMatches)
	}
	globalStats.MostGoalsMatch = maxGoalsMatchID
	globalStats.ActiveViewers = 45000 + rand.Intn(200000)
	globalStats.TopScorer = *topScorer
	globalStats.LastUpdate = time.Now()
}

// Enhanced HTTP Handlers

func getAllMatches(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	matchList := make([]*Match, 0, len(matches))
	for _, match := range matches {
		matchList = append(matchList, match)
	}
	mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"matches":   matchList,
		"count":     len(matchList),
		"timestamp": time.Now(),
	})
}

func getMatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	mutex.RLock()
	match, exists := matches[id]
	mutex.RUnlock()

	if !exists {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(match)
}

func getMatchStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	mutex.RLock()
	stats, exists := matchStats[id]
	mutex.RUnlock()

	if !exists {
		http.Error(w, "Match stats not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func getMatchCommentary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	mutex.RLock()
	commentary, exists := liveCommentary[id]
	if !exists {
		commentary = []*LiveCommentary{}
	}
	mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"commentary": commentary,
		"match_id":   id,
		"count":      len(commentary),
		"timestamp":  time.Now(),
	})
}

func getLeagueTable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	league := vars["league"]

	mutex.RLock()
	table, exists := leagueTables[league]
	mutex.RUnlock()

	if !exists {
		http.Error(w, "League not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"table":     table,
		"league":    league,
		"timestamp": time.Now(),
	})
}

func getAllPlayers(w http.ResponseWriter, r *http.Request) {
	teamIDStr := r.URL.Query().Get("team_id")
	position := r.URL.Query().Get("position")
	search := r.URL.Query().Get("search")

	mutex.RLock()
	var playerList []*Player
	for _, player := range players {
		// Filter by team if specified
		if teamIDStr != "" {
			teamID, err := strconv.Atoi(teamIDStr)
			if err == nil && player.TeamID != teamID {
				continue
			}
		}

		// Filter by position if specified
		if position != "" && player.Position != position {
			continue
		}

		// Filter by search term if specified
		if search != "" && !strings.Contains(strings.ToLower(player.Name), strings.ToLower(search)) {
			continue
		}

		playerList = append(playerList, player)
	}
	mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"players":   playerList,
		"count":     len(playerList),
		"timestamp": time.Now(),
	})
}

func getPlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid player ID", http.StatusBadRequest)
		return
	}

	mutex.RLock()
	player, exists := players[id]
	mutex.RUnlock()

	if !exists {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func getAllTeams(w http.ResponseWriter, r *http.Request) {
	league := r.URL.Query().Get("league")

	mutex.RLock()
	var teamList []*TeamInfo
	for _, team := range teams {
		if league == "" || team.League == league {
			teamList = append(teamList, team)
		}
	}
	mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"teams":     teamList,
		"count":     len(teamList),
		"timestamp": time.Now(),
	})
}

func getTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	mutex.RLock()
	team, exists := teams[id]
	mutex.RUnlock()

	if !exists {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

func getGlobalStats(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	stats := *globalStats
	mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func searchAPI(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	query = strings.ToLower(query)
	
	mutex.RLock()
	
	var results = struct {
		Players []*Player   `json:"players"`
		Teams   []*TeamInfo `json:"teams"`
		Count   int         `json:"count"`
	}{
		Players: []*Player{},
		Teams:   []*TeamInfo{},
	}

	// Search players
	for _, player := range players {
		if strings.Contains(strings.ToLower(player.Name), query) ||
		   strings.Contains(strings.ToLower(player.Position), query) ||
		   strings.Contains(strings.ToLower(player.Nationality), query) {
			results.Players = append(results.Players, player)
		}
	}

	// Search teams
	for _, team := range teams {
		if strings.Contains(strings.ToLower(team.Name), query) ||
		   strings.Contains(strings.ToLower(team.League), query) ||
		   strings.Contains(strings.ToLower(team.Manager), query) {
			results.Teams = append(results.Teams, team)
		}
	}
	
	mutex.RUnlock()

	results.Count = len(results.Players) + len(results.Teams)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	matchCount := len(matches)
	playerCount := len(players)
	teamCount := len(teams)
	mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "healthy",
		"name":           "MatchPulse API",
		"version":        version,
		"active_matches": matchCount,
		"total_players":  playerCount,
		"total_teams":    teamCount,
		"timestamp":      time.Now(),
	})
}

func serveHomepage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	
	mutex.RLock()
	templateData := struct {
		ActiveMatches int
		TotalPlayers  int
		TotalTeams    int
		LastUpdated   string
		Version       string
	}{
		ActiveMatches: len(matches),
		TotalPlayers:  len(players),
		TotalTeams:    len(teams),
		LastUpdated:   time.Now().Format("2006-01-02 15:04:05"),
		Version:       version,
	}
	mutex.RUnlock()
	
	const htmlTemplate = `<!DOCTYPE html>
<html>
<head><title>MatchPulse API v{{.Version}}</title></head>
<body style="font-family: system-ui; max-width: 800px; margin: 0 auto; padding: 2rem;">
<h1>MatchPulse API v{{.Version}}</h1>
<p>Enhanced real-time sports data for state management testing</p>
<h2>Statistics</h2>
<ul>
<li>Active Matches: {{.ActiveMatches}}</li>
<li>Total Players: {{.TotalPlayers}}</li>
<li>Total Teams: {{.TotalTeams}}</li>
</ul>
<h2>New Endpoints</h2>
<ul>
<li><a href="/api/v1/players">Players</a></li>
<li><a href="/api/v1/teams">Teams</a></li>
<li><a href="/api/v1/league-table/Premier%20League">League Tables</a></li>
<li><a href="/api/v1/matches/1/commentary">Live Commentary</a></li>
<li><a href="/api/v1/search?q=manchester">Search</a></li>
</ul>
</body>
</html>`
	
	tmpl, err := template.New("homepage").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	
	err = tmpl.Execute(w, templateData)
	if err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}

func min(a, b int) int {
	if a < b { return a }
	return b
}

func max(a, b int) int {
	if a > b { return a }
	return b
}

func main() {
	r := mux.NewRouter()

	// Enhanced API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	
	// Core endpoints
	api.HandleFunc("/health", healthCheck).Methods("GET")
	api.HandleFunc("/matches", getAllMatches).Methods("GET")
	api.HandleFunc("/matches/{id:[0-9]+}", getMatch).Methods("GET")
	api.HandleFunc("/matches/{id:[0-9]+}/stats", getMatchStats).Methods("GET")
	api.HandleFunc("/matches/{id:[0-9]+}/commentary", getMatchCommentary).Methods("GET")
	api.HandleFunc("/global-stats", getGlobalStats).Methods("GET")
	
	// New enhanced endpoints
	api.HandleFunc("/players", getAllPlayers).Methods("GET")
	api.HandleFunc("/players/{id:[0-9]+}", getPlayer).Methods("GET")
	api.HandleFunc("/teams", getAllTeams).Methods("GET")
	api.HandleFunc("/teams/{id:[0-9]+}", getTeam).Methods("GET")
	api.HandleFunc("/league-table/{league}", getLeagueTable).Methods("GET")
	api.HandleFunc("/search", searchAPI).Methods("GET")

	r.HandleFunc("/", serveHomepage).Methods("GET")

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%s", port)
	}

	fmt.Printf("🚀 MatchPulse API v%s starting on port %s\n", version, port)
	fmt.Printf("📊 Enhanced Documentation: %s\n", baseURL)
	fmt.Printf("⚽ Live matches: %s/api/v1/matches\n", baseURL)
	fmt.Printf("👥 Players: %s/api/v1/players\n", baseURL)
	fmt.Printf("🏆 Teams: %s/api/v1/teams\n", baseURL)
	fmt.Printf("📈 League Tables: %s/api/v1/league-table/Premier%%20League\n", baseURL)
	fmt.Printf("🔍 Search: %s/api/v1/search?q=manchester\n", baseURL)
	
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
