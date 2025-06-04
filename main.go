package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
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
	StatusCooldown  = "COOLDOWN"
	StatusBreak     = "BREAK"

	// Leagues
	LeaguePremier = "Premier League"
	LeagueLaLiga  = "La Liga"

	// League configuration
	TeamsPerLeague   = 20 // Each league has 20 teams
	MatchesPerTeam   = 38 // Each team plays 38 matches (19 home, 19 away)
	MatchdaysPerWeek = 2  // 2 matchdays per week

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
	EventKickoff      = "KICKOFF"
	EventFoul         = "FOUL"
	EventCorner       = "CORNER"
	EventOffside      = "OFFSIDE"

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

	// Game constants
	MatchDurationSeconds   = 90
	HalftimeBreakSeconds   = 15 // New: 15 second halftime break
	PostMatchBreakSeconds  = 60 // New: 1 minute break after match
	CooldownSeconds        = 15
	SeasonMatches          = 38
	MaxSeasonHistory       = 10
	FieldWidth             = 100.0
	FieldHeight            = 64.0
	MaxSimultaneousMatches = 4    // Maximum number of matches that can run at once PER LEAGUE
	MaxNewsEntries         = 100  // Maximum news entries to keep
	MaxLogEntries          = 1000 // Maximum log entries to keep
)

var (
	formations        = []string{Formation442, Formation433, Formation352, Formation4231, Formation532}
	weatherConditions = []string{WeatherClear, WeatherCloudy, WeatherLightRain, WeatherOvercast, WeatherSunny, WeatherPartlyCloudy}
	formResults       = []string{FormWin, FormLoss, FormDraw}
)

// New structures for enhanced features
type NewsEntry struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	MatchID   int       `json:"match_id,omitempty"`
	Type      string    `json:"type"` // "halftime", "fulltime", "general"
	Timestamp time.Time `json:"timestamp"`
	Generated bool      `json:"generated"` // Whether this was AI generated
}

type LogEntry struct {
	ID        int       `json:"id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Enhanced data structures maintaining backward compatibility
type Match struct {
	ID            int       `json:"id"`
	HomeTeam      TeamInfo  `json:"home_team"`
	AwayTeam      TeamInfo  `json:"away_team"`
	HomeScore     int       `json:"home_score"`
	AwayScore     int       `json:"away_score"`
	Minute        int       `json:"minute"`
	Status        string    `json:"status"`
	Competition   string    `json:"competition"`
	LastUpdate    time.Time `json:"last_update"`
	Venue         string    `json:"venue"`
	Attendance    int       `json:"attendance"`
	Weather       string    `json:"weather"`
	Temperature   int       `json:"temperature"`
	HomeFormation string    `json:"home_formation"`
	AwayFormation string    `json:"away_formation"`
	// New fields for extended simulation
	Season        int             `json:"season"`
	MatchweekNum  int             `json:"matchweek"`
	StartTime     time.Time       `json:"start_time"`
	EndTime       *time.Time      `json:"end_time,omitempty"`
	PlayerRatings map[int]float64 `json:"player_ratings,omitempty"`
	// New fields for breaks and injury time
	InjuryTime      int       `json:"injury_time"` // Additional minutes
	HalftimeEndTime time.Time `json:"halftime_end_time,omitempty"`
	IsInBreak       bool      `json:"is_in_break"`
}

type TeamInfo struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	ShortName  string   `json:"short_name"`
	LogoURL    string   `json:"logo_url"`
	Stadium    string   `json:"stadium"`
	Founded    int      `json:"founded"`
	Manager    string   `json:"manager"`
	League     string   `json:"league"`
	Form       []string `json:"form"`        // Last 5 results (W/D/L)
	FormPoints int      `json:"form_points"` // Points from last 5 matches
	HomeStreak int      `json:"home_streak"` // Consecutive home wins/losses
	AwayStreak int      `json:"away_streak"` // Consecutive away wins/losses
}

type Player struct {
	ID              int                   `json:"id"`
	Name            string                `json:"name"`
	Position        string                `json:"position"`
	Number          int                   `json:"number"`
	Age             int                   `json:"age"`
	Nationality     string                `json:"nationality"`
	AvatarURL       string                `json:"avatar_url"`
	TeamID          int                   `json:"team_id"`
	Goals           int                   `json:"goals"`
	Assists         int                   `json:"assists"`
	YellowCards     int                   `json:"yellow_cards"`
	RedCards        int                   `json:"red_cards"`
	Appearances     int                   `json:"appearances"`
	MarketValue     int                   `json:"market_value"`
	LastUpdate      time.Time             `json:"last_update"`
	Characteristics PlayerCharacteristics `json:"characteristics"`
	SeasonStats     PlayerSeasonStats     `json:"season_stats"`
	CurrentRating   float64               `json:"current_rating"`
}

type PlayerCharacteristics struct {
	Speed       int `json:"speed"`       // 1-100
	Shooting    int `json:"shooting"`    // 1-100
	Passing     int `json:"passing"`     // 1-100
	Defending   int `json:"defending"`   // 1-100
	Physicality int `json:"physicality"` // 1-100
	Mentality   int `json:"mentality"`   // 1-100
	Overall     int `json:"overall"`     // Calculated average
}

type PlayerSeasonStats struct {
	MatchesPlayed         int     `json:"matches_played"`
	MinutesPlayed         int     `json:"minutes_played"`
	GoalsThisSeason       int     `json:"goals_this_season"`
	AssistsThisSeason     int     `json:"assists_this_season"`
	YellowCardsThisSeason int     `json:"yellow_cards_this_season"`
	RedCardsThisSeason    int     `json:"red_cards_this_season"`
	AverageRating         float64 `json:"average_rating"`
	TotalRating           float64 `json:"total_rating"`
}

type PlayerLocation struct {
	PlayerID  int       `json:"player_id"`
	X         float64   `json:"x"` // 0-100 (field width percentage)
	Y         float64   `json:"y"` // 0-64 (field height percentage)
	Timestamp time.Time `json:"timestamp"`
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
	Position     int       `json:"position"`
	Team         TeamInfo  `json:"team"`
	Played       int       `json:"played"`
	Won          int       `json:"won"`
	Drawn        int       `json:"drawn"`
	Lost         int       `json:"lost"`
	GoalsFor     int       `json:"goals_for"`
	GoalsAgainst int       `json:"goals_against"`
	GoalDiff     int       `json:"goal_difference"`
	Points       int       `json:"points"`
	Form         []string  `json:"form"`
	LastUpdate   time.Time `json:"last_update"`
}

type LiveCommentary struct {
	ID         int       `json:"id"`
	MatchID    int       `json:"match_id"`
	Minute     int       `json:"minute"`
	Text       string    `json:"text"`
	EventType  string    `json:"event_type"`
	Player     *Player   `json:"player,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	AudioText  string    `json:"audio_text,omitempty"`
	AudioSpeed float64   `json:"audio_speed,omitempty"`
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
	// New fields for extended simulation
	CurrentSeason    int     `json:"current_season"`
	CurrentMatchweek int     `json:"current_matchweek"`
	SeasonProgress   float64 `json:"season_progress"`
}

type SeasonWinners struct {
	PremierLeagueWinner *TeamInfo `json:"premier_league_winner"`
	LaLigaWinner        *TeamInfo `json:"la_liga_winner"`
}

type SeasonHistory struct {
	Season         int           `json:"season"`
	Winners        SeasonWinners `json:"winners"`
	TopScorer      Player        `json:"top_scorer"`
	TopAssists     Player        `json:"top_assists"`
	MostFouls      Player        `json:"most_fouls"`
	PlayerOfSeason Player        `json:"player_of_season"`
	Champion       TeamInfo      `json:"champion"`
	TotalGoals     int           `json:"total_goals"`
	TotalMatches   int           `json:"total_matches"`
	EndDate        time.Time     `json:"end_date"`
}

type SeasonSchedule struct {
	Matchday    int       `json:"matchday"`
	League      string    `json:"league"`
	HomeTeam    *TeamInfo `json:"home_team"`
	AwayTeam    *TeamInfo `json:"away_team"`
	IsPlayed    bool      `json:"is_played"`
	MatchID     int       `json:"match_id,omitempty"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

// League configuration for extensibility
type LeagueConfig struct {
	Name        string `json:"name"`
	TeamCount   int    `json:"team_count"`
	Matchdays   int    `json:"matchdays"`
	StartTeamID int    `json:"start_team_id"`
	EndTeamID   int    `json:"end_team_id"`
}

// Global league configurations
var leagueConfigs = map[string]LeagueConfig{
	LeaguePremier: {
		Name:        LeaguePremier,
		TeamCount:   TeamsPerLeague,
		Matchdays:   MatchesPerTeam,
		StartTeamID: 1,
		EndTeamID:   20,
	},
	LeagueLaLiga: {
		Name:        LeagueLaLiga,
		TeamCount:   TeamsPerLeague,
		Matchdays:   MatchesPerTeam,
		StartTeamID: 21,
		EndTeamID:   40,
	},
}

// In-memory database - add season schedules
var (
	// Original storage
	matches        = make(map[int]*Match)
	matchStats     = make(map[int]*MatchStats)
	players        = make(map[int]*Player)
	teams          = make(map[int]*TeamInfo)
	leagueTables   = make(map[string][]*LeagueTable)
	liveCommentary = make(map[int][]*LiveCommentary)
	globalStats    = &GlobalStats{}

	// Extended storage
	playerLocations  = make(map[int]map[int]*PlayerLocation) // matchID -> playerID -> location
	seasonHistory    = make([]SeasonHistory, 0, MaxSeasonHistory)
	seasonSchedules  = make(map[string][]*SeasonSchedule) // league -> schedules
	currentSeason    = 1
	currentMatchweek = 1

	// New storage for enhanced features
	newsEntries = make([]*NewsEntry, 0, MaxNewsEntries)
	logEntries  = make([]*LogEntry, 0, MaxLogEntries)
	newsCounter = 0
	logCounter  = 0

	// Counters and synchronization
	commentaryCounter = 0
	matchCounter      = 0
	mutex             = &sync.RWMutex{}
	version           = "1.2.0"

	// Add this at the top of the file with other global variables
	startTime = time.Now()
)

// Team and player data
var teamData = []struct {
	ID        int
	Name      string
	ShortName string
	Stadium   string
	League    string
	Manager   string
	Founded   int
}{
	// Premier League Teams (1-20)
	{1, "Arsinel", "ARS", "Emerita Stadium", LeaguePremier, "Miguel Artetta", 1886},
	{2, "Chelsey", "CHE", "Stamferd Bridge", LeaguePremier, "Maurizio Pochetino", 1905},
	{3, "Liverpul", "LIV", "The New Anfeld", LeaguePremier, "Jurgen Klopp", 1892},
	{4, "Menchester Citie", "MCI", "Blue Park Stadium", LeaguePremier, "Josep Guardyola", 1880},
	{5, "Menchester Unighted", "MUN", "New Trafford", LeaguePremier, "Eric ten Haag", 1878},
	{6, "Totenham", "TOT", "Totenham Fiery Stadium", LeaguePremier, "Angelo Postecoglu", 1882},
	{7, "Newkastle Unighted", "NEW", "Saint Timothy Park", LeaguePremier, "Edward Howe", 1892},
	{8, "Brighten", "BHA", "Watergate Express Stadium", LeaguePremier, "Robertu Di Zerbi", 1901},
	{9, "Asten Vila", "AVL", "Vila Gates", LeaguePremier, "Unei Emary", 1874},
	{10, "Westham Unighted", "WHU", "London Free City Stadium", LeaguePremier, "Davyd Mois", 1895},
	{11, "Crystel Palas", "CRY", "Selhurst Gardens", LeaguePremier, "Roy Hodgsen", 1905},
	{12, "Evertin", "EVE", "Goodisen Gardens", LeaguePremier, "Sean Dyche", 1878},
	{13, "Fulhem", "FUL", "Cottage Park", LeaguePremier, "Marco Sylva", 1879},
	{14, "Bournemoth", "BOU", "Vitality Gardens", LeaguePremier, "Andoni Iraola", 1899},
	{15, "Lutin Town", "LUT", "Kenilworth Stadium", LeaguePremier, "Rob Edwards", 1885},
	{16, "Notingham Forst", "NFO", "City Gardens", LeaguePremier, "Nuno Espirito", 1865},
	{17, "Shefild Unighted", "SHU", "Bramall Fields", LeaguePremier, "Paul Heckingbottom", 1889},
	{18, "Burnly", "BUR", "Turf Fields", LeaguePremier, "Vincent Kompany", 1882},
	{19, "Wolfs", "WOL", "Molineux Gardens", LeaguePremier, "Gary O'Neil", 1877},
	{20, "Breintford", "BRE", "Community Stadium", LeaguePremier, "Thomas Frank", 1889},

	// La Liga Teams (21-40)
	{21, "Reel Madred", "RMA", "Santiego De Ramon", LeagueLaLiga, "Carlo Ancheloti", 1902},
	{22, "Barselona", "BAR", "Camp Nu", LeagueLaLiga, "Chavi Ernandes", 1899},
	{23, "Atletiko Madred", "ATM", "Metropolitan Alfredo Stadium", LeagueLaLiga, "Diego Simeoane", 1903},
	{24, "Athletik Bilbau", "ATH", "San Marino De Valdes", LeagueLaLiga, "Ernesto Valverdi", 1898},
	{25, "Reel Sosyedad", "RSO", "Reale Areno", LeagueLaLiga, "Imanuel Alguasil", 1909},
	{26, "Vilareal", "VIL", "Estadio de la Submarino", LeagueLaLiga, "Marselino Garsia", 1923},
	{27, "Sevilia", "SEV", "Ramon Kareem Stadium", LeagueLaLiga, "Jose Luis Mendilebar", 1890},
	{28, "Reel Betis", "BET", "Benitu New Park Stadium", LeagueLaLiga, "Manuel Pellegrini", 1907},
	{29, "Valensia", "VAL", "Mestaya", LeagueLaLiga, "Ruben Baraha", 1919},
	{30, "Getaffe", "GET", "Coliseum Alfonsu Dias", LeagueLaLiga, "Jose Bordalas", 1983},
	{31, "Espanyol", "ESP", "Cornella-El Prat", LeagueLaLiga, "Luis Garcia", 1900},
	{32, "Rayo Valekano", "RAY", "Estadio de Vallekas", LeagueLaLiga, "Inigo Perez", 1924},
	{33, "Celta Vigo", "CEL", "Balaidos", LeagueLaLiga, "Claudio Giraldez", 1923},
	{34, "Deportivo Alaves", "ALA", "Mendizorrotza", LeagueLaLiga, "Luis Garcia Plaza", 1921},
	{35, "Real Mallorca", "MAL", "Visit Mallorca Estadi", LeagueLaLiga, "Javier Aguirre", 1916},
	{36, "Las Palmas", "LAS", "Estadio Gran Canaria", LeagueLaLiga, "Garcia Pimienta", 1949},
	{37, "Girona FC", "GIR", "Estadi Montilivi", LeagueLaLiga, "Michel Sanchez", 1930},
	{38, "Osasuna", "OSA", "El Sadar", LeagueLaLiga, "Jagoba Arrasate", 1920},
	{39, "Granada CF", "GRA", "Nuevo Los Carmenes", LeagueLaLiga, "Paco Lopez", 1931},
	{40, "Cadiz CF", "CAD", "Estadio Ramon de Carranza", LeagueLaLiga, "Mauricio Pellegrino", 1910},
}

var playerNames = []struct {
	Name        string
	Position    string
	Nationality string
}{
	{"Marcus Johnson", PosGK, NatEngland}, {"Davido Silva", PosGK, NatSpain}, {"Antonio López", PosGK, NatSpain},
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
	"{player} dribbles past the defender",
	"Shot blocked by {player}",
	"Offside flag raised against {team}",
	"Beautiful passing move by {team}",
	"The ball goes out for a throw-in",
}

func applicationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func init() {
	rand.Seed(time.Now().UnixNano())
	loadVersion()
	initializeSimulation()
	startSimulationEngine()
}

func loadVersion() {
	if data, err := os.ReadFile("version.txt"); err == nil {
		version = strings.TrimSpace(string(data))
	}
}

func initializeSimulation() {
	mutex.Lock()
	defer mutex.Unlock()

	// Initialize teams with form tracking
	for _, teamInfo := range teamData {
		teams[teamInfo.ID] = &TeamInfo{
			ID:         teamInfo.ID,
			Name:       teamInfo.Name,
			ShortName:  teamInfo.ShortName,
			LogoURL:    fmt.Sprintf("https://ui-avatars.com/api/?name=%s&background=random&size=128", strings.ReplaceAll(teamInfo.ShortName, " ", "+")),
			Stadium:    teamInfo.Stadium,
			Founded:    teamInfo.Founded,
			Manager:    teamInfo.Manager,
			League:     teamInfo.League,
			Form:       []string{},
			FormPoints: 0,
			HomeStreak: 0,
			AwayStreak: 0,
		}
	}

	playerID := 1
	for _, team := range teams {
		for i := 0; i < 25; i++ {
			playerTemplate := playerNames[rand.Intn(len(playerNames))]
			name := playerTemplate.Name
			if i > len(playerNames)-1 {
				name = fmt.Sprintf("%s %d", playerTemplate.Name, i)
			}

			characteristics := generatePlayerCharacteristics(playerTemplate.Position)

			players[playerID] = &Player{
				ID:              playerID,
				Name:            name,
				Position:        playerTemplate.Position,
				Number:          i + 1,
				Age:             18 + rand.Intn(20),
				Nationality:     playerTemplate.Nationality,
				AvatarURL:       fmt.Sprintf("https://i.pravatar.cc/150?img=%d", (playerID%70)+1),
				TeamID:          team.ID,
				Goals:           0,
				Assists:         0,
				YellowCards:     0,
				RedCards:        0,
				Appearances:     0,
				MarketValue:     calculateMarketValue(characteristics),
				LastUpdate:      time.Now(),
				Characteristics: characteristics,
				SeasonStats:     PlayerSeasonStats{},
				CurrentRating:   6.0,
			}
			playerID++
		}
	}

	initializeLeagueTables()

	// Generate season schedules for all leagues
	for league := range leagueConfigs {
		generateSeasonSchedule(league)
	}

	updateGlobalStats()

	// Create initial matches if none exists - fill up to MaxSimultaneousMatches per league
	if len(matches) == 0 {
		log.Printf("🏁 Creating initial matches up to maximum (%d per league)...", MaxSimultaneousMatches)
		matchesCreated := 0

		// Create matches for each league
		for _, league := range []string{LeaguePremier, LeagueLaLiga} {
			for i := 0; i < MaxSimultaneousMatches; i++ {
				// Find next match for this specific league
				nextMatch := getNextUnplayedMatchForLeague(league)
				if nextMatch != nil {
					log.Printf("📊 Creating initial match %d/%d for %s: %s vs %s",
						i+1, MaxSimultaneousMatches, league, nextMatch.HomeTeam.ShortName, nextMatch.AwayTeam.ShortName)
					createNextMatch()
					matchesCreated++
				} else {
					log.Printf("⚠️  No more matches available for %s after %d matches", league, i)
					break
				}
			}
		}
		log.Printf("✅ Created %d initial matches", matchesCreated)
	}

	log.Printf("🏆 Simulation initialized: %d leagues, %d teams (%d players each), %d active matches",
		len(leagueConfigs), len(teams), 25, len(matches))
}

func generatePlayerCharacteristics(position string) PlayerCharacteristics {
	// Base stats vary by position
	var speed, shooting, passing, defending, physicality, mentality int

	switch position {
	case PosGK:
		speed = 20 + rand.Intn(30)
		shooting = 10 + rand.Intn(20)
		passing = 40 + rand.Intn(40)
		defending = 60 + rand.Intn(40)
		physicality = 60 + rand.Intn(40)
		mentality = 70 + rand.Intn(30)
	case PosCB:
		speed = 30 + rand.Intn(40)
		shooting = 20 + rand.Intn(30)
		passing = 50 + rand.Intn(40)
		defending = 70 + rand.Intn(30)
		physicality = 70 + rand.Intn(30)
		mentality = 60 + rand.Intn(40)
	case PosLB, PosRB:
		speed = 60 + rand.Intn(40)
		shooting = 30 + rand.Intn(40)
		passing = 60 + rand.Intn(40)
		defending = 60 + rand.Intn(40)
		physicality = 50 + rand.Intn(40)
		mentality = 50 + rand.Intn(40)
	case PosCDM:
		speed = 40 + rand.Intn(40)
		shooting = 40 + rand.Intn(40)
		passing = 70 + rand.Intn(30)
		defending = 70 + rand.Intn(30)
		physicality = 60 + rand.Intn(40)
		mentality = 60 + rand.Intn(40)
	case PosCM:
		speed = 50 + rand.Intn(40)
		shooting = 50 + rand.Intn(40)
		passing = 70 + rand.Intn(30)
		defending = 50 + rand.Intn(40)
		physicality = 50 + rand.Intn(40)
		mentality = 60 + rand.Intn(40)
	case PosCAM:
		speed = 60 + rand.Intn(40)
		shooting = 70 + rand.Intn(30)
		passing = 70 + rand.Intn(30)
		defending = 30 + rand.Intn(40)
		physicality = 40 + rand.Intn(40)
		mentality = 70 + rand.Intn(30)
	case PosLW, PosRW:
		speed = 70 + rand.Intn(30)
		shooting = 60 + rand.Intn(40)
		passing = 60 + rand.Intn(40)
		defending = 30 + rand.Intn(40)
		physicality = 40 + rand.Intn(40)
		mentality = 60 + rand.Intn(40)
	case PosST:
		speed = 60 + rand.Intn(40)
		shooting = 80 + rand.Intn(20)
		passing = 50 + rand.Intn(40)
		defending = 20 + rand.Intn(30)
		physicality = 60 + rand.Intn(40)
		mentality = 70 + rand.Intn(30)
	default:
		speed = 50 + rand.Intn(40)
		shooting = 50 + rand.Intn(40)
		passing = 50 + rand.Intn(40)
		defending = 50 + rand.Intn(40)
		physicality = 50 + rand.Intn(40)
		mentality = 50 + rand.Intn(40)
	}

	overall := (speed + shooting + passing + defending + physicality + mentality) / 6

	return PlayerCharacteristics{
		Speed:       speed,
		Shooting:    shooting,
		Passing:     passing,
		Defending:   defending,
		Physicality: physicality,
		Mentality:   mentality,
		Overall:     overall,
	}
}

func calculateMarketValue(characteristics PlayerCharacteristics) int {
	// Market value based on overall rating (5-200 million)
	baseValue := characteristics.Overall / 2
	variation := rand.Intn(30) - 15 // +/- 15%
	return max(5, min(200, baseValue+variation))
}

func startSimulationEngine() {
	logInfo("🚀 Starting simulation engine with %d teams and %d players", len(teams), len(players))

	// Create a context that won't be cancelled immediately
	ctx := context.Background()

	// Only start goroutines that actually do different work
	go matchEngine(ctx)         // Handles live match simulation
	go seasonManager(ctx)       // Handles season transitions
	go statisticsProcessor(ctx) // Processes player/team stats
	go updateGlobalStatsLoop()  // Global statistics updates

	logInfo("✅ All simulation engines started successfully")
	logInfo("🔍 Monitoring: Match engine, Season manager, Statistics processor, Global stats updater")
}

func matchEngine(ctx context.Context) {
	logInfo("🎮 Match engine started - checking every 2 seconds")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mutex.Lock()
			activeMatches := 0
			liveMatches := 0
			halftimeMatches := 0
			breakMatches := 0
			finishedMatches := 0

			// Count matches by status first
			for _, match := range matches {
				switch match.Status {
				case StatusLive:
					liveMatches++
					activeMatches++
				case StatusHalftime:
					halftimeMatches++
					activeMatches++
				case StatusBreak:
					breakMatches++
					activeMatches++
				case StatusFinished:
					finishedMatches++
				}
			}

			logInfo("📊 Match status: Active: %d (Live: %d, Halftime: %d, Break: %d), Finished: %d, Total: %d",
				activeMatches, liveMatches, halftimeMatches, breakMatches, finishedMatches, len(matches))

			// Update live matches using comprehensive logic
			matchesUpdated := 0
			for matchID, match := range matches {
				if match.Status == StatusLive || match.Status == StatusHalftime || match.Status == StatusBreak {
					elapsed := time.Since(match.StartTime).Seconds()
					logInfo("⚽ Updating match %d: %s vs %s (Minute %d→%.0f, Status: %s, Elapsed: %.1fs)",
						matchID, match.HomeTeam.ShortName, match.AwayTeam.ShortName,
						match.Minute, elapsed, match.Status, elapsed)

					updateLiveMatchWithBreaks(matchID, match)
					matchesUpdated++

					// Check if match just finished to update league table
					if match.Status == StatusFinished {
						logInfo("🏁 Match %d finished: %s %d-%d %s",
							matchID, match.HomeTeam.ShortName, match.HomeScore,
							match.AwayScore, match.AwayTeam.ShortName)
						updateLeagueTable(match)

						// Start cooldown timer for next match
						go startCooldownAndCreateNext(matchID)
					}
				}
			}

			if activeMatches == 0 {
				logInfo("⚠️  No active matches found - total matches: %d", len(matches))
				logInfo("🆕 Attempting to create new matches...")

				// Try to create new matches if we have none active
				newMatchesCreated := 0
				for i := 0; i < MaxSimultaneousMatches; i++ {
					nextMatch := getNextUnplayedMatch()
					if nextMatch != nil {
						logInfo("🔄 Creating replacement match %d: %s vs %s",
							i+1, nextMatch.HomeTeam.ShortName, nextMatch.AwayTeam.ShortName)
						createNextMatch()
						newMatchesCreated++
					} else {
						logInfo("⚠️  No more scheduled matches available")
						break
					}
				}
				logInfo("✅ Created %d new matches", newMatchesCreated)
			} else {
				logInfo("✅ Updated %d active matches", matchesUpdated)
			}

			mutex.Unlock()

		case <-ctx.Done():
			logInfo("🛑 Match engine stopped")
			return
		}
	}
}

func updateLiveMatchWithBreaks(matchID int, match *Match) {
	now := time.Now()
	elapsed := now.Sub(match.StartTime).Seconds()

	// Handle different match states
	switch match.Status {
	case StatusLive:
		// Update minute based on elapsed time
		oldMinute := match.Minute
		match.Minute = int(elapsed)

		if match.Minute != oldMinute {
			logInfo("⏱️  Match %d: Minute %d → %d (Elapsed: %.1fs)", matchID, oldMinute, match.Minute, elapsed)
		}

		// Check for halftime (45 minutes + injury time)
		if match.Minute >= 45 && match.Minute < 46 {
			logInfo("🏃‍♂️ Match %d: HALFTIME! Teams head to the tunnel", matchID)
			match.Status = StatusHalftime
			match.HalftimeEndTime = now.Add(HalftimeBreakSeconds * time.Second)
			match.IsInBreak = true
			addLiveCommentary(matchID, match.Minute, "Halftime! Teams head to the tunnel", EventCommentary, nil)
			return
		}

		// Check for full time (90 minutes + injury time)
		totalMatchTime := MatchDurationSeconds + match.InjuryTime
		if elapsed >= float64(totalMatchTime) {
			logInfo("⏰ Match %d: Time's up! Finishing match...", matchID)
			finishMatch(matchID, match)
			return
		}

		// Generate events during live play
		if rand.Float32() < 0.15 {
			logInfo("🎲 Match %d: Event triggered! Generating match event...", matchID)
			generateMatchEvent(match)
		}

	case StatusHalftime:
		// Check if halftime break is over
		if now.After(match.HalftimeEndTime) {
			logInfo("🏃‍♂️ Match %d: Second half underway!", matchID)
			match.Status = StatusLive
			match.IsInBreak = false
			// Reset start time to account for break
			breakDuration := now.Sub(match.HalftimeEndTime.Add(-HalftimeBreakSeconds * time.Second))
			match.StartTime = match.StartTime.Add(breakDuration)
			addLiveCommentary(matchID, match.Minute, "Second half underway!", EventCommentary, nil)
		}
		return

	case StatusBreak:
		// This is handled by the post-match break logic
		return
	}

	// Update player locations every cycle (only during live play)
	if match.Status == StatusLive {
		updatePlayerLocationsForMatch(matchID)
	}

	// Update match statistics
	updateMatchStatistics(matchID, match)
	match.LastUpdate = time.Now()
}

func generateMatchEvent(match *Match) {
	logInfo("🎯 Generating event for match %d at minute %d", match.ID, match.Minute)

	// Calculate team strengths
	homeStrength := calculateTeamStrength(&match.HomeTeam)
	awayStrength := calculateTeamStrength(&match.AwayTeam)

	// Base event probabilities
	baseProbabilities := map[string]float32{
		EventGoal:       0.15,
		EventCard:       0.20,
		EventCorner:     0.25,
		EventFoul:       0.25,
		EventCommentary: 0.15,
	}

	// Adjust probabilities based on team strengths and match minute
	strengthDiff := homeStrength - awayStrength

	// TODO: Allow real-time events adjust the probabilities
	// minuteFactor := float32(match.Minute) / 90.0 // More events likely in later minutes

	// Adjust goal probability based on team strengths
	if strengthDiff > 0 {
		baseProbabilities[EventGoal] *= float32(1.0 + strengthDiff*0.5)
	} else {
		baseProbabilities[EventGoal] *= float32(1.0 - strengthDiff*0.5)
	}

	// Adjust card probability based on team strengths (weaker teams more likely to commit fouls)
	if strengthDiff < 0 {
		baseProbabilities[EventCard] *= float32(1.0 - strengthDiff*0.3)
		baseProbabilities[EventFoul] *= float32(1.0 - strengthDiff*0.3)
	}

	// Normalize probabilities
	totalProb := float32(0)
	for _, prob := range baseProbabilities {
		totalProb += prob
	}
	for event := range baseProbabilities {
		baseProbabilities[event] /= totalProb
	}

	// Generate event based on adjusted probabilities
	r := rand.Float32()
	cumulativeProb := float32(0)

	for eventType, prob := range baseProbabilities {
		cumulativeProb += prob
		if r <= cumulativeProb {
			// Handle the selected event
			switch eventType {
			case EventGoal:
				handleGoalEvent(match.ID, match)
			case EventCard:
				handleCardEvent(match.ID, match)
			case EventCorner:
				handleCornerEvent(match.ID, match)
			case EventFoul:
				handleFoulEvent(match.ID, match)
			case EventCommentary:
				generateGenericCommentary(match.ID, match)
			}
			break
		}
	}

	// Update match stats
	if stats := matchStats[match.ID]; stats != nil {
		stats.LastUpdate = time.Now()
	}
}

func handleGoalEvent(matchID int, match *Match) {
	// Calculate goal probability based on team strengths
	homeStrength := calculateTeamStrength(&match.HomeTeam)
	awayStrength := calculateTeamStrength(&match.AwayTeam)

	// Determine scoring team based on strengths
	isHomeGoal := rand.Float32() < float32(homeStrength/(homeStrength+awayStrength))
	var scorer *Player

	if isHomeGoal {
		scorer = getRandomPlayerFromTeam(match.HomeTeam.ID)
		match.HomeScore++
		logInfo("⚽ GOAL! %s scores for %s! New score: %s %d-%d %s",
			scorer.Name, match.HomeTeam.ShortName,
			match.HomeTeam.ShortName, match.HomeScore, match.AwayScore, match.AwayTeam.ShortName)
	} else {
		scorer = getRandomPlayerFromTeam(match.AwayTeam.ID)
		match.AwayScore++
		logInfo("⚽ GOAL! %s scores for %s! New score: %s %d-%d %s",
			scorer.Name, match.AwayTeam.ShortName,
			match.HomeTeam.ShortName, match.HomeScore, match.AwayScore, match.AwayTeam.ShortName)
	}

	if scorer != nil {
		scorer.Goals++
		scorer.SeasonStats.GoalsThisSeason++
		scorer.CurrentRating += 1.5

		addLiveCommentary(matchID, match.Minute,
			fmt.Sprintf("GOAL! %s scores! What a fantastic finish!", scorer.Name),
			EventGoal, scorer)

		logInfo("📈 Player stats updated: %s now has %d goals this season", scorer.Name, scorer.SeasonStats.GoalsThisSeason)
	}
}

func handleCardEvent(matchID int, match *Match) {
	isHomePlayer := rand.Float32() < 0.5
	var player *Player

	if isHomePlayer {
		player = getRandomPlayerFromTeam(match.HomeTeam.ID)
	} else {
		player = getRandomPlayerFromTeam(match.AwayTeam.ID)
	}

	if player != nil {
		cardType := "yellow"
		if rand.Float32() < 0.1 { // 10% chance for red card
			cardType = "red"
			player.RedCards++
			player.SeasonStats.RedCardsThisSeason++
			player.CurrentRating -= 2.0
			logInfo("🟥 RED CARD! %s receives a red card!", player.Name)
		} else {
			player.YellowCards++
			player.SeasonStats.YellowCardsThisSeason++
			player.CurrentRating -= 0.5
			logInfo("🟨 YELLOW CARD! %s receives a yellow card", player.Name)
		}

		addLiveCommentary(matchID, match.Minute,
			fmt.Sprintf("%s card shown to %s", strings.Title(cardType), player.Name),
			EventCard, player)
	}
}

// Cooldown and next match creation
func startCooldownAndCreateNext(finishedMatchID int) {
	logInfo("⏳ Starting %d-second post-match break for match %d...", PostMatchBreakSeconds, finishedMatchID)
	time.Sleep(PostMatchBreakSeconds * time.Second)

	mutex.Lock()
	// Remove finished match from active matches after cooldown
	delete(matches, finishedMatchID)
	logInfo("🗑️  Removed finished match %d from active matches", finishedMatchID)

	// Create next match
	logInfo("🆕 Creating next match after post-match break...")
	createNextMatch()
	mutex.Unlock()
}

func createNextMatch() {
	// Get next scheduled match
	scheduledMatch := getNextUnplayedMatch()
	if scheduledMatch == nil {
		log.Printf("⚠️  No more scheduled matches available")
		return
	}

	// Check if we've reached the maximum number of simultaneous matches for this league
	leagueActiveCount := 0
	for _, match := range matches {
		if (match.Status == StatusLive || match.Status == StatusHalftime || match.Status == StatusBreak) &&
			match.Competition == scheduledMatch.League {
			leagueActiveCount++
		}
	}

	if leagueActiveCount >= MaxSimultaneousMatches {
		log.Printf("⚠️  Maximum simultaneous matches reached for %s (%d), waiting for a match to finish",
			scheduledMatch.League, MaxSimultaneousMatches)
		return
	}

	matchCounter++

	// Calculate match probabilities based on team form
	homeWin, draw, awayWin := calculateMatchProbabilities(scheduledMatch.HomeTeam, scheduledMatch.AwayTeam)

	// Calculate attack strengths
	homeAttackStrength := calculateAttackStrength(scheduledMatch.HomeTeam)
	awayAttackStrength := calculateAttackStrength(scheduledMatch.AwayTeam)

	// Generate random injury time (0-6 minutes)
	injuryTime := rand.Intn(7) // 0-6 additional seconds (representing minutes)

	// Create match from schedule
	match := &Match{
		ID:            matchCounter,
		HomeTeam:      *scheduledMatch.HomeTeam,
		AwayTeam:      *scheduledMatch.AwayTeam,
		HomeScore:     0,
		AwayScore:     0,
		Minute:        0,
		Status:        StatusLive,
		Competition:   scheduledMatch.League,
		Venue:         scheduledMatch.HomeTeam.Stadium,
		Attendance:    rand.Intn(80000) + 20000,
		Weather:       weatherConditions[rand.Intn(len(weatherConditions))],
		Temperature:   rand.Intn(25) + 5,
		HomeFormation: formations[rand.Intn(len(formations))],
		AwayFormation: formations[rand.Intn(len(formations))],
		Season:        currentSeason,
		MatchweekNum:  scheduledMatch.Matchday,
		StartTime:     time.Now(),
		LastUpdate:    time.Now(),
		PlayerRatings: make(map[int]float64),
		// New fields for breaks and injury time
		InjuryTime: injuryTime,
		IsInBreak:  false,
	}

	// Mark schedule as being played
	scheduledMatch.IsPlayed = true
	scheduledMatch.MatchID = matchCounter

	matches[matchCounter] = match
	matchStats[matchCounter] = generateInitialMatchStats(matchCounter)
	liveCommentary[matchCounter] = []*LiveCommentary{}
	playerLocations[matchCounter] = make(map[int]*PlayerLocation)

	// Add kickoff commentary with form and probability information
	homeForm := fmt.Sprintf("Form: %v", scheduledMatch.HomeTeam.Form)
	awayForm := fmt.Sprintf("Form: %v", scheduledMatch.AwayTeam.Form)
	probabilityInfo := fmt.Sprintf("Match odds: Home %.1f%% Draw %.1f%% Away %.1f%%",
		homeWin*100, draw*100, awayWin*100)

	addLiveCommentary(matchCounter, 0,
		fmt.Sprintf("⚽ Kickoff! %s (%s) vs %s (%s) at %s\n%s",
			scheduledMatch.HomeTeam.Name, homeForm,
			scheduledMatch.AwayTeam.Name, awayForm,
			scheduledMatch.HomeTeam.Stadium,
			probabilityInfo),
		EventKickoff, nil)

	logInfo("🆕 New match created: %s vs %s (ID: %d, League: %s, Matchday: %d, Injury Time: +%d min) - %s active matches: %d/%d",
		scheduledMatch.HomeTeam.ShortName, scheduledMatch.AwayTeam.ShortName,
		matchCounter, scheduledMatch.League, scheduledMatch.Matchday, injuryTime,
		scheduledMatch.League, leagueActiveCount+1, MaxSimultaneousMatches)
	logInfo("🏟️  Venue: %s, Weather: %s, Temperature: %d°C",
		scheduledMatch.HomeTeam.Stadium, match.Weather, match.Temperature)
	logInfo("📋 Formations: %s (%s) vs %s (%s)",
		match.HomeFormation, scheduledMatch.HomeTeam.ShortName,
		match.AwayFormation, scheduledMatch.AwayTeam.ShortName)
	logInfo("📊 Form - Home: %v (Points: %d) | Away: %v (Points: %d)",
		scheduledMatch.HomeTeam.Form, scheduledMatch.HomeTeam.FormPoints,
		scheduledMatch.AwayTeam.Form, scheduledMatch.AwayTeam.FormPoints)
	logInfo("🎲 Match probabilities: Home %.1f%% Draw %.1f%% Away %.1f%%",
		homeWin*100, draw*100, awayWin*100)
	logInfo("⚔️  Attack strengths: Home %.2f | Away %.2f",
		homeAttackStrength, awayAttackStrength)
	logInfo("🕐 Match %d started at %s (Expected duration: %d + %d minutes)",
		matchCounter, match.StartTime.Format("15:04:05"), MatchDurationSeconds, injuryTime)
}

func updateGlobalStatsLoop() {
	logInfo("📊 Global stats updater started - updating every 5 seconds")
	for {
		time.Sleep(5 * time.Second)
		mutex.Lock()
		updateGlobalStats()
		mutex.Unlock()
		logInfo("📈 Global stats updated: %d matches, %d goals, %.1f avg goals, %d viewers",
			globalStats.TotalMatches, globalStats.TotalGoals, globalStats.AverageGoals, globalStats.ActiveViewers)
	}
}

func seasonManager(ctx context.Context) {
	logInfo("🏆 Season manager started - checking daily")
	// Only runs when there's actual season management work
	seasonTicker := time.NewTicker(24 * time.Hour) // Check daily
	defer seasonTicker.Stop()

	for {
		select {
		case <-seasonTicker.C:
			logInfo("🗓️  Daily season check: Season %d, Week %d", currentSeason, currentMatchweek)
			if shouldEndSeason() {
				logInfo("🏁 Ending season %d...", currentSeason)
				procesSeasonEnd()
				startNewSeason()
			}

		case <-ctx.Done():
			logInfo("🛑 Season manager stopped")
			return
		}
	}
}

func statisticsProcessor(ctx context.Context) {
	logInfo("🔢 Statistics processor started - updating every 30 seconds")
	// Batch process statistics to avoid constant DB hits
	statsTicker := time.NewTicker(30 * time.Second)
	defer statsTicker.Stop()

	for {
		select {
		case <-statsTicker.C:
			logInfo("📊 Processing player and team statistics...")
			batchUpdatePlayerStats()
			batchUpdateTeamStats()

		case <-ctx.Done():
			logInfo("🛑 Statistics processor stopped")
			return
		}
	}
}

// Implementation of actual functions
func updateLeagueTable(match *Match) {
	logInfo("📊 Updating league table after match %d", match.ID)

	// Update season progress when match finishes
	currentMatchweek++
	if currentMatchweek > SeasonMatches {
		logInfo("🏁 Season complete! Starting season transition...")
		endSeason()
	}

	// Immediately update league standings when match finishes
	if match.HomeScore > match.AwayScore {
		// Home win
		updateTeamStats(match.HomeTeam.ID, 3, 1, 0, 0, match) // 3 points, 1 win
		updateTeamStats(match.AwayTeam.ID, 0, 0, 0, 1, match) // 0 points, 1 loss
		logInfo("🏆 %s wins against %s", match.HomeTeam.ShortName, match.AwayTeam.ShortName)
	} else if match.AwayScore > match.HomeScore {
		// Away win
		updateTeamStats(match.AwayTeam.ID, 3, 1, 0, 0, match)
		updateTeamStats(match.HomeTeam.ID, 0, 0, 0, 1, match)
		logInfo("🏆 %s wins against %s", match.AwayTeam.ShortName, match.HomeTeam.ShortName)
	} else {
		// Draw
		updateTeamStats(match.HomeTeam.ID, 1, 0, 1, 0, match) // 1 point, 1 draw
		updateTeamStats(match.AwayTeam.ID, 1, 0, 1, 0, match)
		logInfo("🤝 Draw between %s and %s", match.HomeTeam.ShortName, match.AwayTeam.ShortName)
	}
}

func updateGlobalStats() {
	totalGoals := 0
	maxGoals := 0
	maxGoalsMatchID := 0
	topScorer := findTopScorer()

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

	liveMatches := len(matches)
	globalStats.TotalMatches = liveMatches
	globalStats.TotalGoals = totalGoals
	if liveMatches > 0 {
		globalStats.AverageGoals = float64(totalGoals) / float64(liveMatches)
	}
	globalStats.MostGoalsMatch = maxGoalsMatchID
	globalStats.ActiveViewers = 45000 + rand.Intn(200000)
	if topScorer != nil {
		globalStats.TopScorer = *topScorer
	}
	globalStats.CurrentSeason = currentSeason
	globalStats.CurrentMatchweek = currentMatchweek
	globalStats.SeasonProgress = float64(currentMatchweek) / float64(SeasonMatches) * 100
	globalStats.LastUpdate = time.Now()
}

// HTTP Handlers (maintaining backward compatibility)
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

func getMatchLocations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	mutex.RLock()
	locations, exists := playerLocations[id]
	if !exists {
		locations = make(map[int]*PlayerLocation)
	}

	locationList := make([]*PlayerLocation, 0, len(locations))
	for _, location := range locations {
		locationList = append(locationList, location)
	}
	mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"locations": locationList,
		"match_id":  id,
		"count":     len(locationList),
		"timestamp": time.Now(),
	})
}

func getSeasonHistory(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	history := make([]SeasonHistory, len(seasonHistory))
	copy(history, seasonHistory)
	mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history":   history,
		"count":     len(history),
		"timestamp": time.Now(),
	})
}

func getSeasonStats(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	stats := map[string]interface{}{
		"current_season":    currentSeason,
		"current_matchweek": currentMatchweek,
		"season_progress":   float64(currentMatchweek) / float64(SeasonMatches) * 100,
		"total_seasons":     len(seasonHistory),
		"timestamp":         time.Now(),
	}
	mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
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
		if teamIDStr != "" {
			teamID, err := strconv.Atoi(teamIDStr)
			if err == nil && player.TeamID != teamID {
				continue
			}
		}

		if position != "" && player.Position != position {
			continue
		}

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

	for _, player := range players {
		if strings.Contains(strings.ToLower(player.Name), query) ||
			strings.Contains(strings.ToLower(player.Position), query) ||
			strings.Contains(strings.ToLower(player.Nationality), query) {
			results.Players = append(results.Players, player)
		}
	}

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

// Add this new function for goroutine monitoring
func getGoroutineStats() map[string]interface{} {
	numGoroutines := runtime.NumGoroutine()

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"goroutine_count": numGoroutines,
		"memory_alloc":    memStats.Alloc,
		"memory_total":    memStats.TotalAlloc,
		"memory_sys":      memStats.Sys,
		"num_gc":          memStats.NumGC,
		"status":          getGoroutineStatus(numGoroutines),
	}
}

func getGoroutineStatus(count int) string {
	if count < 50 {
		return "healthy"
	} else if count < 100 {
		return "warning"
	} else {
		return "critical"
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	matchCount := len(matches)
	playerCount := len(players)
	teamCount := len(teams)

	// Get goroutine stats
	goroutineStats := getGoroutineStats()

	// Get active matches count
	activeMatches := 0
	for _, match := range matches {
		if match.Status == StatusLive || match.Status == StatusHalftime {
			activeMatches++
		}
	}

	// Get current season progress
	seasonProgress := float64(currentMatchweek) / float64(SeasonMatches) * 100

	// Get system uptime
	uptime := time.Since(startTime).Round(time.Second)

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Prepare detailed health check response
	healthData := map[string]interface{}{
		"status":          "healthy",
		"name":            "MatchPulse API Extended",
		"version":         version,
		"uptime":          uptime.String(),
		"active_matches":  activeMatches,
		"max_matches":     MaxSimultaneousMatches,
		"total_matches":   matchCount,
		"total_players":   playerCount,
		"total_teams":     teamCount,
		"current_season":  currentSeason,
		"matchweek":       currentMatchweek,
		"season_progress": fmt.Sprintf("%.1f%%", seasonProgress),
		"goroutines":      goroutineStats,
		"memory": map[string]interface{}{
			"alloc":       memStats.Alloc,
			"total_alloc": memStats.TotalAlloc,
			"sys":         memStats.Sys,
			"num_gc":      memStats.NumGC,
		},
		"timestamp": time.Now(),
	}

	// Log health check details
	log.Printf("🏥 Health Check: %d/%d active matches, %d total matches, %d goroutines (%s)",
		activeMatches, MaxSimultaneousMatches, matchCount, goroutineStats["goroutine_count"], goroutineStats["status"])

	mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthData)
}

func serveHomepage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	mutex.RLock()
	templateData := struct {
		ActiveMatches    int
		TotalPlayers     int
		TotalTeams       int
		CurrentSeason    int
		CurrentMatchweek int
		LastUpdated      string
		Version          string
	}{
		ActiveMatches:    len(matches),
		TotalPlayers:     len(players),
		TotalTeams:       len(teams),
		CurrentSeason:    currentSeason,
		CurrentMatchweek: currentMatchweek,
		LastUpdated:      time.Now().Format("2006-01-02 15:04:05"),
		Version:          version,
	}
	mutex.RUnlock()

	const htmlTemplate = `<!DOCTYPE html>
<html>
<head><title>MatchPulse API v{{.Version}} - Extended Football Simulation</title></head>
<body style="font-family: system-ui; max-width: 800px; margin: 0 auto; padding: 2rem;">
<h1>MatchPulse API v{{.Version}}</h1>
<p>Extended real-time football simulation for state management testing</p>
<h2>Simulation Status</h2>
<ul>
<li>Active Matches: {{.ActiveMatches}}</li>
<li>Total Players: {{.TotalPlayers}}</li>
<li>Total Teams: {{.TotalTeams}}</li>
<li>Current Season: {{.CurrentSeason}}</li>
<li>Current Matchweek: {{.CurrentMatchweek}}</li>
</ul>
<h2>Core Endpoints</h2>
<ul>
<li><a href="/api/v1/players">Players</a></li>
<li><a href="/api/v1/teams">Teams</a></li>
<li><a href="/api/v1/matches">Matches</a></li>
<li><a href="/api/v1/league-table/Premier%20League">League Tables</a></li>
<li><a href="/api/v1/global-stats">Global Stats</a></li>
</ul>
<h2>Extended Features</h2>
<ul>
<li><a href="/api/v1/matches/1/locations">Live Player Locations</a></li>
<li><a href="/api/v1/season/history">Season History</a></li>
<li><a href="/api/v1/season/stats">Season Stats</a></li>
<li><a href="/api/v1/search?q=manchester">Search</a></li>
<li><a href="/tables">View Tables</a></li>
</ul>
<h2>Features</h2>
<ul>
<li>90-second realistic matches with 15-second cooldown</li>
<li>Live player locations and ratings</li>
<li>Season simulation with historical data (10 seasons)</li>
<li>Audio-ready commentary with speed controls</li>
<li>Player characteristics and dynamic stats</li>
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
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func shouldEndSeason() bool {
	return currentMatchweek > SeasonMatches
}

func procesSeasonEnd() {
	endSeason()
}

func startNewSeason() {
	log.Printf("🆕 Starting season %d...", currentSeason)
	resetForNewSeason()
	currentSeason++
	currentMatchweek = 1
	createNextMatch()
}

func batchUpdatePlayerStats() {
	// Batch update player statistics to reduce lock contention
	mutex.Lock()
	defer mutex.Unlock()

	playersUpdated := 0
	for _, player := range players {
		if player.SeasonStats.MatchesPlayed > 0 {
			player.SeasonStats.AverageRating = player.SeasonStats.TotalRating / float64(player.SeasonStats.MatchesPlayed)
			playersUpdated++
		}
	}

	if playersUpdated > 0 {
		log.Printf("📈 Updated stats for %d players", playersUpdated)
	}
}

func batchUpdateTeamStats() {
	// Update team statistics in batches
	updateGlobalStats()
}

func endSeason() {
	log.Printf("🏆 Season %d ended! Calculating final standings...", currentSeason)

	// Calculate season winners and stats
	seasonWinner := calculateSeasonWinner()
	topScorer := findTopScorer()
	topAssists := findTopAssists()
	mostFouls := findMostFouls()
	playerOfSeason := findPlayerOfSeason()

	// Store season history
	seasonRecord := SeasonHistory{
		Season:         currentSeason,
		TopScorer:      *topScorer,
		TopAssists:     *topAssists,
		MostFouls:      *mostFouls,
		PlayerOfSeason: *playerOfSeason,
		Champion:       *seasonWinner,
		TotalGoals:     calculateTotalSeasonGoals(),
		TotalMatches:   SeasonMatches,
		EndDate:        time.Now(),
	}

	seasonHistory = append(seasonHistory, seasonRecord)
	if len(seasonHistory) > MaxSeasonHistory {
		seasonHistory = seasonHistory[1:]
	}

	log.Printf("🥇 Season %d Champions: %s", currentSeason, seasonWinner.Name)
	log.Printf("⚽ Top Scorer: %s (%d goals)", topScorer.Name, topScorer.SeasonStats.GoalsThisSeason)
	log.Printf("🅰️  Top Assists: %s (%d assists)", topAssists.Name, topAssists.SeasonStats.AssistsThisSeason)

	// Reset for new season
	resetForNewSeason()

	currentSeason++
	currentMatchweek = 1
}

func calculateSeasonWinner() *TeamInfo {
	// Get the team at the top of Premier League table (or default to first team)
	if table, exists := leagueTables[LeaguePremier]; exists && len(table) > 0 {
		return &table[0].Team
	}

	// Fallback to any team
	for _, team := range teams {
		return team
	}

	return &TeamInfo{Name: "Unknown"}
}

func findTopScorer() *Player {
	var topScorer *Player
	maxGoals := 0

	for _, player := range players {
		if player.SeasonStats.GoalsThisSeason > maxGoals {
			maxGoals = player.SeasonStats.GoalsThisSeason
			topScorer = player
		}
	}

	if topScorer == nil {
		for _, player := range players {
			return player
		}
	}

	return topScorer
}

func findTopAssists() *Player {
	var topAssists *Player
	maxAssists := 0

	for _, player := range players {
		if player.SeasonStats.AssistsThisSeason > maxAssists {
			maxAssists = player.SeasonStats.AssistsThisSeason
			topAssists = player
		}
	}

	if topAssists == nil {
		for _, player := range players {
			return player
		}
	}

	return topAssists
}

func findMostFouls() *Player {
	var mostFouls *Player
	maxFouls := 0

	for _, player := range players {
		totalFouls := player.SeasonStats.YellowCardsThisSeason + player.SeasonStats.RedCardsThisSeason*2
		if totalFouls > maxFouls {
			maxFouls = totalFouls
			mostFouls = player
		}
	}

	if mostFouls == nil {
		for _, player := range players {
			return player
		}
	}

	return mostFouls
}

func findPlayerOfSeason() *Player {
	var playerOfSeason *Player
	maxRating := 0.0

	for _, player := range players {
		if player.SeasonStats.AverageRating > maxRating && player.SeasonStats.MatchesPlayed >= 10 {
			maxRating = player.SeasonStats.AverageRating
			playerOfSeason = player
		}
	}

	if playerOfSeason == nil {
		for _, player := range players {
			return player
		}
	}

	return playerOfSeason
}

func calculateTotalSeasonGoals() int {
	total := 0
	for _, player := range players {
		total += player.SeasonStats.GoalsThisSeason
	}
	return total
}

func resetForNewSeason() {
	log.Printf("🔄 Resetting for season %d...", currentSeason+1)

	// Reset season stats for all players
	playersReset := 0
	for _, player := range players {
		// Move season stats to cumulative stats
		player.Goals += player.SeasonStats.GoalsThisSeason
		player.Assists += player.SeasonStats.AssistsThisSeason
		player.YellowCards += player.SeasonStats.YellowCardsThisSeason
		player.RedCards += player.SeasonStats.RedCardsThisSeason

		// Reset season stats
		player.SeasonStats = PlayerSeasonStats{}
		player.CurrentRating = 6.0
		playersReset++
	}

	// Reset league tables
	initializeLeagueTables()

	log.Printf("📊 Reset complete: %d players reset, league tables reinitialized", playersReset)
}

type CommentaryEntry struct {
	MatchID   int       `json:"match_id"`
	Minute    int       `json:"minute"`
	Text      string    `json:"text"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
}

func updateTeamStats(teamID, points, wins, draws, losses int, match *Match) {
	team := teams[teamID]
	if team == nil {
		return
	}

	// Determine result for form tracking
	result := FormLoss
	if wins > 0 {
		result = FormWin
	} else if draws > 0 {
		result = FormDraw
	}

	// Properly determine if it's a home game
	isHome := match.HomeTeam.ID == teamID

	// Update team form with correct home/away status
	updateTeamForm(team, result, isHome)

	// Update team statistics in league table
	for league, table := range leagueTables {
		for _, teamEntry := range table {
			if teamEntry.Team.ID == teamID {
				teamEntry.Points += points
				teamEntry.Won += wins
				teamEntry.Drawn += draws
				teamEntry.Lost += losses
				teamEntry.Played = teamEntry.Won + teamEntry.Drawn + teamEntry.Lost

				// Update goals for and against based on match result
				if isHome {
					teamEntry.GoalsFor += match.HomeScore
					teamEntry.GoalsAgainst += match.AwayScore
				} else {
					teamEntry.GoalsFor += match.AwayScore
					teamEntry.GoalsAgainst += match.HomeScore
				}
				teamEntry.GoalDiff = teamEntry.GoalsFor - teamEntry.GoalsAgainst

				// Update the team reference with form data
				teamEntry.Team = *team

				teamEntry.LastUpdate = time.Now()

				log.Printf("📋 %s: %d points (%d played, %d won, %d drawn, %d lost, %d GF, %d GA, %d GD) - Form: %v",
					teamEntry.Team.ShortName, teamEntry.Points, teamEntry.Played,
					teamEntry.Won, teamEntry.Drawn, teamEntry.Lost, teamEntry.GoalsFor,
					teamEntry.GoalsAgainst, teamEntry.GoalDiff, team.Form)
				break
			}
		}

		// Resort league table by points
		sortLeagueTable(leagueTables[league])
	}
}

func sortLeagueTable(table []*LeagueTable) {
	// Simple bubble sort by points, then goal difference
	for i := 0; i < len(table)-1; i++ {
		for j := 0; j < len(table)-i-1; j++ {
			if table[j].Points < table[j+1].Points ||
				(table[j].Points == table[j+1].Points && table[j].GoalDiff < table[j+1].GoalDiff) {
				table[j], table[j+1] = table[j+1], table[j]
			}
		}
	}

	// Update positions
	for i, team := range table {
		team.Position = i + 1
	}
}

func getSeasonSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	league := vars["league"]

	mutex.RLock()
	schedules, exists := seasonSchedules[league]
	if !exists {
		mutex.RUnlock()
		http.Error(w, "League not found", http.StatusNotFound)
		return
	}

	// Filter by matchday if specified
	matchdayStr := r.URL.Query().Get("matchday")
	if matchdayStr != "" {
		matchday, err := strconv.Atoi(matchdayStr)
		if err == nil {
			filteredSchedules := getScheduledMatches(league, matchday)
			mutex.RUnlock()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"league":    league,
				"matchday":  matchday,
				"schedules": filteredSchedules,
				"count":     len(filteredSchedules),
				"timestamp": time.Now(),
			})
			return
		}
	}

	// Return all schedules for the league
	mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"league":    league,
		"schedules": schedules,
		"count":     len(schedules),
		"timestamp": time.Now(),
	})
}

func getTeamForm(w http.ResponseWriter, r *http.Request) {
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

	formData := map[string]interface{}{
		"team_id":     team.ID,
		"team_name":   team.Name,
		"short_name":  team.ShortName,
		"league":      team.League,
		"form":        team.Form,
		"form_points": team.FormPoints,
		"home_streak": team.HomeStreak,
		"away_streak": team.AwayStreak,
		"timestamp":   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(formData)
}

func getLeagueForm(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	league := vars["league"]

	mutex.RLock()
	var leagueTeams []*TeamInfo
	for _, team := range teams {
		if team.League == league {
			leagueTeams = append(leagueTeams, team)
		}
	}
	mutex.RUnlock()

	if len(leagueTeams) == 0 {
		http.Error(w, "League not found", http.StatusNotFound)
		return
	}

	// Sort teams by form points (descending)
	for i := 0; i < len(leagueTeams)-1; i++ {
		for j := 0; j < len(leagueTeams)-i-1; j++ {
			if leagueTeams[j].FormPoints < leagueTeams[j+1].FormPoints {
				leagueTeams[j], leagueTeams[j+1] = leagueTeams[j+1], leagueTeams[j]
			}
		}
	}

	formTable := make([]map[string]interface{}, len(leagueTeams))
	for i, team := range leagueTeams {
		formTable[i] = map[string]interface{}{
			"position":    i + 1,
			"team_id":     team.ID,
			"team_name":   team.Name,
			"short_name":  team.ShortName,
			"form":        team.Form,
			"form_points": team.FormPoints,
			"home_streak": team.HomeStreak,
			"away_streak": team.AwayStreak,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"league":     league,
		"form_table": formTable,
		"count":      len(formTable),
		"timestamp":  time.Now(),
	})
}

func getMatchdaySchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchdayStr := vars["matchday"]

	matchday, err := strconv.Atoi(matchdayStr)
	if err != nil {
		http.Error(w, "Invalid matchday", http.StatusBadRequest)
		return
	}

	mutex.RLock()
	allMatches := make(map[string][]*SeasonSchedule)

	// Get matches for all leagues
	for league := range leagueConfigs {
		matches := getScheduledMatches(league, matchday)
		if len(matches) > 0 {
			allMatches[league] = matches
		}
	}
	mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"matchday":  matchday,
		"matches":   allMatches,
		"timestamp": time.Now(),
	})
}

// Update main function to include new routes
func main() {
	r := mux.NewRouter()

	// Apply application middleware
	r.Use(applicationMiddleware)

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Core endpoints (backward compatible)
	api.HandleFunc("/health", healthCheck).Methods("GET")
	api.HandleFunc("/matches", getAllMatches).Methods("GET")
	api.HandleFunc("/matches/{id:[0-9]+}", getMatch).Methods("GET")
	api.HandleFunc("/matches/{id:[0-9]+}/stats", getMatchStats).Methods("GET")
	api.HandleFunc("/matches/{id:[0-9]+}/commentary", getMatchCommentary).Methods("GET")
	api.HandleFunc("/global-stats", getGlobalStats).Methods("GET")
	api.HandleFunc("/players", getAllPlayers).Methods("GET")
	api.HandleFunc("/players/{id:[0-9]+}", getPlayer).Methods("GET")
	api.HandleFunc("/teams", getAllTeams).Methods("GET")
	api.HandleFunc("/teams/{id:[0-9]+}", getTeam).Methods("GET")
	api.HandleFunc("/league-table/{league}", getLeagueTable).Methods("GET")
	api.HandleFunc("/search", searchAPI).Methods("GET")

	// Extended endpoints
	api.HandleFunc("/matches/{id:[0-9]+}/locations", getMatchLocations).Methods("GET")
	api.HandleFunc("/season/history", getSeasonHistory).Methods("GET")
	api.HandleFunc("/season/stats", getSeasonStats).Methods("GET")

	// New season scheduling and form endpoints
	api.HandleFunc("/season/schedule/{league}", getSeasonSchedule).Methods("GET")
	api.HandleFunc("/teams/{id:[0-9]+}/form", getTeamForm).Methods("GET")
	api.HandleFunc("/league/{league}/form", getLeagueForm).Methods("GET")
	api.HandleFunc("/matchday/{matchday:[0-9]+}", getMatchdaySchedule).Methods("GET")

	// Add the new table view endpoint
	api.HandleFunc("/tables", getTableData).Methods("GET")

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

	fmt.Printf("🚀 MatchPulse API v%s Extended starting on port %s\n", version, port)
	fmt.Printf("📊 Documentation: %s\n", baseURL)
	fmt.Printf("⚽ Live matches: %s/api/v1/matches\n", baseURL)
	fmt.Printf("📍 Player locations: %s/api/v1/matches/1/locations\n", baseURL)
	fmt.Printf("🏆 Season history: %s/api/v1/season/history\n", baseURL)
	fmt.Printf("📈 Season stats: %s/api/v1/season/stats\n", baseURL)
	fmt.Printf("📅 Season schedule: %s/api/v1/season/schedule/Premier%%20League\n", baseURL)
	fmt.Printf("📊 Team form: %s/api/v1/teams/1/form\n", baseURL)
	fmt.Printf("🏅 League form table: %s/api/v1/league/Premier%%20League/form\n", baseURL)
	fmt.Printf("📋 Matchday schedule: %s/api/v1/matchday/1\n", baseURL)
	fmt.Printf("\n🎯 Enhanced simulation with proper season scheduling and team form!\n")

	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// Enhanced match statistics update
func updateMatchStatistics(matchID int, match *Match) {
	stats := matchStats[matchID]
	if stats == nil {
		return
	}

	// Update possession based on match minute
	basePossession := 50
	variation := int(math.Sin(float64(match.Minute)/10) * 10)
	stats.HomePossession = basePossession + variation
	stats.AwayPossession = 100 - stats.HomePossession

	// Increment passes based on possession
	if rand.Float32() < 0.7 {
		if stats.HomePossession > 50 {
			stats.HomePasses += rand.Intn(3) + 1
		} else {
			stats.AwayPasses += rand.Intn(3) + 1
		}
	}

	// Update pass accuracy
	stats.HomePassAccuracy = 75.0 + rand.Float64()*20
	stats.AwayPassAccuracy = 75.0 + rand.Float64()*20

	stats.LastUpdate = time.Now()
}

func finishMatch(matchID int, match *Match) {
	now := time.Now()
	match.Status = StatusFinished
	match.EndTime = &now

	// Calculate player ratings based on performance
	calculatePlayerRatings(match)

	// Add final whistle commentary
	addLiveCommentary(matchID, match.Minute,
		fmt.Sprintf("Full time! %s %d - %d %s",
			match.HomeTeam.Name, match.HomeScore,
			match.AwayScore, match.AwayTeam.Name),
		EventCommentary, nil)

	log.Printf("🏁 Match %d finished: %s %d-%d %s",
		matchID, match.HomeTeam.ShortName, match.HomeScore,
		match.AwayScore, match.AwayTeam.ShortName)
}

func handleCornerEvent(matchID int, match *Match) {
	isHomeCorner := rand.Float32() < 0.5
	teamName := match.AwayTeam.Name
	if isHomeCorner {
		teamName = match.HomeTeam.Name
		matchStats[matchID].HomeCorners++
	} else {
		matchStats[matchID].AwayCorners++
	}

	log.Printf("⚽ Corner kick for %s in match %d", teamName, matchID)
	addLiveCommentary(matchID, match.Minute,
		fmt.Sprintf("Corner kick for %s", teamName),
		EventCorner, nil)
}

func handleFoulEvent(matchID int, match *Match) {
	isHomeFoul := rand.Float32() < 0.5
	var player *Player

	if isHomeFoul {
		player = getRandomPlayerFromTeam(match.HomeTeam.ID)
		matchStats[matchID].HomeFouls++
	} else {
		player = getRandomPlayerFromTeam(match.AwayTeam.ID)
		matchStats[matchID].AwayFouls++
	}

	if player != nil {
		log.Printf("🦵 Foul committed by %s in match %d", player.Name, matchID)
		addLiveCommentary(matchID, match.Minute,
			fmt.Sprintf("Foul committed by %s", player.Name),
			EventFoul, player)
	}
}

func generateGenericCommentary(matchID int, match *Match) {
	template := commentaryTemplates[rand.Intn(len(commentaryTemplates))]
	text := template

	if strings.Contains(template, "{player}") {
		player := getRandomPlayerFromTeam(match.HomeTeam.ID)
		if rand.Float32() < 0.5 {
			player = getRandomPlayerFromTeam(match.AwayTeam.ID)
		}
		if player != nil {
			text = strings.ReplaceAll(text, "{player}", player.Name)
		}
	}

	if strings.Contains(template, "{team}") {
		team := match.HomeTeam.Name
		if rand.Float32() < 0.5 {
			team = match.AwayTeam.Name
		}
		text = strings.ReplaceAll(text, "{team}", team)
	}

	log.Printf("💬 Commentary for match %d: %s", matchID, text)
	addLiveCommentary(matchID, match.Minute, text, EventCommentary, nil)
}

func addLiveCommentary(matchID, minute int, text, eventType string, player *Player) {
	commentaryCounter++

	commentary := &LiveCommentary{
		ID:         commentaryCounter,
		MatchID:    matchID,
		Minute:     minute,
		Text:       text,
		EventType:  eventType,
		Player:     player,
		Timestamp:  time.Now(),
		AudioText:  generateAudioText(text, eventType),
		AudioSpeed: generateAudioSpeed(eventType),
	}

	if liveCommentary[matchID] == nil {
		liveCommentary[matchID] = []*LiveCommentary{}
	}

	liveCommentary[matchID] = append([]*LiveCommentary{commentary}, liveCommentary[matchID]...)

	if len(liveCommentary[matchID]) > 30 {
		liveCommentary[matchID] = liveCommentary[matchID][:30]
	}

	log.Printf("📝 Commentary added for match %d: %s", matchID, text)
}

func generateAudioText(text, eventType string) string {
	// Enhanced text for audio streaming
	switch eventType {
	case EventGoal:
		return "GOOOOOAAAL! " + text
	case EventCard:
		return "Card shown. " + text
	default:
		return text
	}
}

func generateAudioSpeed(eventType string) float64 {
	// Different speeds for different events
	switch eventType {
	case EventGoal:
		return 1.2 // Faster, more excitement
	case EventCard:
		return 0.9 // Slower, more serious
	default:
		return 1.0 // Normal speed
	}
}

func updatePlayerLocationsForMatch(matchID int) {
	match := matches[matchID]
	if match == nil {
		return
	}

	if playerLocations[matchID] == nil {
		playerLocations[matchID] = make(map[int]*PlayerLocation)
	}

	// Get players from both teams
	homePlayers := getPlayersFromTeam(match.HomeTeam.ID)
	awayPlayers := getPlayersFromTeam(match.AwayTeam.ID)

	// Update locations for 11 players from each team (starting lineup)
	updateTeamLocations(matchID, homePlayers[:min(11, len(homePlayers))], true)
	updateTeamLocations(matchID, awayPlayers[:min(11, len(awayPlayers))], false)
}

func updateTeamLocations(matchID int, teamPlayers []*Player, isHome bool) {
	match := matches[matchID]
	if match == nil {
		return
	}

	for i, player := range teamPlayers {
		if player == nil {
			continue
		}

		// Generate realistic positions based on formation and game flow
		var x, y float64

		if isHome {
			// Home team starts from left side
			x = generateXPosition(player.Position, true, match.Minute)
			y = generateYPosition(player.Position, i)
		} else {
			// Away team starts from right side
			x = generateXPosition(player.Position, false, match.Minute)
			y = generateYPosition(player.Position, i)
		}

		// Add some randomness for realistic movement
		x += (rand.Float64() - 0.5) * 10
		y += (rand.Float64() - 0.5) * 8

		// Keep within field bounds
		x = math.Max(0, math.Min(FieldWidth, x))
		y = math.Max(0, math.Min(FieldHeight, y))

		playerLocations[matchID][player.ID] = &PlayerLocation{
			PlayerID:  player.ID,
			X:         x,
			Y:         y,
			Timestamp: time.Now(),
		}
	}
}

func generateXPosition(position string, isHome bool, minute int) float64 {
	var baseX float64

	if isHome {
		switch position {
		case PosGK:
			baseX = 5
		case PosCB:
			baseX = 15
		case PosLB, PosRB:
			baseX = 25
		case PosCDM:
			baseX = 35
		case PosCM:
			baseX = 45
		case PosCAM:
			baseX = 55
		case PosLW, PosRW:
			baseX = 65
		case PosST:
			baseX = 75
		default:
			baseX = 40
		}
	} else {
		// Mirror for away team
		switch position {
		case PosGK:
			baseX = 95
		case PosCB:
			baseX = 85
		case PosLB, PosRB:
			baseX = 75
		case PosCDM:
			baseX = 65
		case PosCM:
			baseX = 55
		case PosCAM:
			baseX = 45
		case PosLW, PosRW:
			baseX = 35
		case PosST:
			baseX = 25
		default:
			baseX = 60
		}
	}

	// Add game flow variation
	flowVariation := math.Sin(float64(minute)/10) * 10
	return baseX + flowVariation
}

func generateYPosition(position string, playerIndex int) float64 {
	switch position {
	case PosGK:
		return FieldHeight / 2
	case PosCB:
		if playerIndex%2 == 0 {
			return FieldHeight/2 - 8
		}
		return FieldHeight/2 + 8
	case PosLB:
		return FieldHeight * 0.8
	case PosRB:
		return FieldHeight * 0.2
	case PosCDM, PosCM:
		return FieldHeight/2 + float64((playerIndex%3-1)*12)
	case PosCAM:
		return FieldHeight / 2
	case PosLW:
		return FieldHeight * 0.8
	case PosRW:
		return FieldHeight * 0.2
	case PosST:
		if playerIndex%2 == 0 {
			return FieldHeight/2 - 6
		}
		return FieldHeight/2 + 6
	default:
		return FieldHeight / 2
	}
}

func calculatePlayerRatings(match *Match) {
	if match.PlayerRatings == nil {
		match.PlayerRatings = make(map[int]float64)
	}

	homePlayers := getPlayersFromTeam(match.HomeTeam.ID)
	awayPlayers := getPlayersFromTeam(match.AwayTeam.ID)

	for _, player := range homePlayers {
		rating := calculateIndividualRating(player, match, true)
		match.PlayerRatings[player.ID] = rating
		updatePlayerSeasonStats(player, rating, 90)
	}

	for _, player := range awayPlayers {
		rating := calculateIndividualRating(player, match, false)
		match.PlayerRatings[player.ID] = rating
		updatePlayerSeasonStats(player, rating, 90)
	}
}

func calculateIndividualRating(player *Player, match *Match, isHome bool) float64 {
	baseRating := 6.0

	// Adjust based on result
	if (isHome && match.HomeScore > match.AwayScore) || (!isHome && match.AwayScore > match.HomeScore) {
		baseRating += 0.5 // Win bonus
	} else if match.HomeScore == match.AwayScore {
		baseRating += 0.2 // Draw bonus
	}

	// Add current match rating adjustments from events
	baseRating += math.Max(-3.0, math.Min(3.0, player.CurrentRating-6.0))

	// Add some randomness based on characteristics
	performanceVariation := (rand.Float64() - 0.5) * 2 * (float64(player.Characteristics.Mentality) / 100)
	baseRating += performanceVariation

	// Reset current rating for next match
	player.CurrentRating = 6.0

	return math.Max(1.0, math.Min(10.0, baseRating))
}

func updatePlayerSeasonStats(player *Player, rating float64, minutesPlayed int) {
	player.SeasonStats.MatchesPlayed++
	player.SeasonStats.MinutesPlayed += minutesPlayed
	player.SeasonStats.TotalRating += rating
	player.SeasonStats.AverageRating = player.SeasonStats.TotalRating / float64(player.SeasonStats.MatchesPlayed)
	player.Appearances++
}

// Utility functions
func getRandomPlayerFromTeam(teamID int) *Player {
	teamPlayers := getPlayersFromTeam(teamID)
	if len(teamPlayers) > 0 {
		return teamPlayers[rand.Intn(len(teamPlayers))]
	}
	return &Player{Name: "Unknown Player"}
}

func getPlayersFromTeam(teamID int) []*Player {
	var teamPlayers []*Player
	for _, player := range players {
		if player.TeamID == teamID {
			teamPlayers = append(teamPlayers, player)
		}
	}
	return teamPlayers
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

func initializeLeagueTables() {
	leagues := []string{LeaguePremier, LeagueLaLiga}

	for _, league := range leagues {
		leagueTeams := getTeamsByLeague(league)
		var table []*LeagueTable

		for i, team := range leagueTeams {
			table = append(table, &LeagueTable{
				Position:     i + 1,
				Team:         *team,
				Played:       0,
				Won:          0,
				Drawn:        0,
				Lost:         0,
				GoalsFor:     0,
				GoalsAgainst: 0,
				GoalDiff:     0,
				Points:       0,
				Form:         []string{},
				LastUpdate:   time.Now(),
			})
		}

		leagueTables[league] = table
	}
}

func generateInitialMatchStats(matchID int) *MatchStats {
	return &MatchStats{
		MatchID:           matchID,
		HomePossession:    50,
		AwayPossession:    50,
		HomeShots:         0,
		AwayShots:         0,
		HomeShotsOnTarget: 0,
		AwayShotsOnTarget: 0,
		HomeCorners:       0,
		AwayCorners:       0,
		HomeFouls:         0,
		AwayFouls:         0,
		HomeYellowCards:   0,
		AwayYellowCards:   0,
		HomeRedCards:      0,
		AwayRedCards:      0,
		HomePasses:        0,
		AwayPasses:        0,
		HomePassAccuracy:  100.0,
		AwayPassAccuracy:  100.0,
		LastUpdate:        time.Now(),
	}
}

// Season scheduling functions
func generateSeasonSchedule(league string) {
	config := leagueConfigs[league]
	leagueTeams := getTeamsByLeague(league)

	if len(leagueTeams) != config.TeamCount {
		log.Printf("❌ League %s has %d teams, expected %d", league, len(leagueTeams), config.TeamCount)
		return
	}

	var schedules []*SeasonSchedule
	matchday := 1

	// Generate home and away fixtures (38 matchdays total)
	for round := 0; round < 2; round++ { // Two rounds: home and away
		for i := 0; i < len(leagueTeams); i++ {
			for j := i + 1; j < len(leagueTeams); j++ {
				var homeTeam, awayTeam *TeamInfo

				if round == 0 {
					homeTeam = leagueTeams[i]
					awayTeam = leagueTeams[j]
				} else {
					homeTeam = leagueTeams[j]
					awayTeam = leagueTeams[i]
				}

				schedule := &SeasonSchedule{
					Matchday:    matchday,
					League:      league,
					HomeTeam:    homeTeam,
					AwayTeam:    awayTeam,
					IsPlayed:    false,
					ScheduledAt: time.Now().Add(time.Duration(matchday) * 24 * time.Hour),
				}

				schedules = append(schedules, schedule)
				matchday++
			}
		}
	}

	seasonSchedules[league] = schedules
	log.Printf("📅 Generated season schedule for %s: %d matches across %d matchdays",
		league, len(schedules), matchday-1)
}

func getScheduledMatches(league string, matchday int) []*SeasonSchedule {
	var matches []*SeasonSchedule

	if schedules, exists := seasonSchedules[league]; exists {
		for _, schedule := range schedules {
			if schedule.Matchday == matchday {
				matches = append(matches, schedule)
			}
		}
	}

	return matches
}

func getNextUnplayedMatchForLeague(league string) *SeasonSchedule {
	if schedules, exists := seasonSchedules[league]; exists {
		for _, schedule := range schedules {
			if !schedule.IsPlayed {
				return schedule
			}
		}
	}
	return nil
}

func getNextUnplayedMatch() *SeasonSchedule {
	// Get the current minimum matches played across all teams to ensure fair scheduling
	minMatchesPlayed := getMinimumMatchesPlayed()

	// First, try to find matches for teams that have played the minimum number of matches
	for _, league := range []string{LeaguePremier, LeagueLaLiga} {
		if schedules, exists := seasonSchedules[league]; exists {
			for _, schedule := range schedules {
				if !schedule.IsPlayed {
					// Check if both teams have played the minimum number of matches
					homeMatchesPlayed := getTeamMatchesPlayed(schedule.HomeTeam.ID)
					awayMatchesPlayed := getTeamMatchesPlayed(schedule.AwayTeam.ID)

					// Prioritize matches where both teams have played minimum matches
					if homeMatchesPlayed <= minMatchesPlayed && awayMatchesPlayed <= minMatchesPlayed {
						logInfo("🎯 Selected match (balanced): %s vs %s (Home: %d played, Away: %d played)",
							schedule.HomeTeam.ShortName, schedule.AwayTeam.ShortName,
							homeMatchesPlayed, awayMatchesPlayed)
						return schedule
					}
				}
			}
		}
	}

	// If no balanced matches found, get any unplayed match (fallback)
	for _, league := range []string{LeaguePremier, LeagueLaLiga} {
		if schedules, exists := seasonSchedules[league]; exists {
			for _, schedule := range schedules {
				if !schedule.IsPlayed {
					homeMatchesPlayed := getTeamMatchesPlayed(schedule.HomeTeam.ID)
					awayMatchesPlayed := getTeamMatchesPlayed(schedule.AwayTeam.ID)
					logInfo("🎯 Selected match (fallback): %s vs %s (Home: %d played, Away: %d played)",
						schedule.HomeTeam.ShortName, schedule.AwayTeam.ShortName,
						homeMatchesPlayed, awayMatchesPlayed)
					return schedule
				}
			}
		}
	}

	return nil
}

func getMinimumMatchesPlayed() int {
	minMatches := 999
	for _, team := range teams {
		matchesPlayed := getTeamMatchesPlayed(team.ID)
		if matchesPlayed < minMatches {
			minMatches = matchesPlayed
		}
	}
	return minMatches
}

func getTeamMatchesPlayed(teamID int) int {
	matchesPlayed := 0
	for _, league := range []string{LeaguePremier, LeagueLaLiga} {
		if schedules, exists := seasonSchedules[league]; exists {
			for _, schedule := range schedules {
				if schedule.IsPlayed && (schedule.HomeTeam.ID == teamID || schedule.AwayTeam.ID == teamID) {
					matchesPlayed++
				}
			}
		}
	}
	return matchesPlayed
}

// Form calculation functions
func calculateMatchProbabilities(homeTeam, awayTeam *TeamInfo) (homeWin, draw, awayWin float64) {
	// Calculate base team strengths
	homeStrength := calculateTeamStrength(homeTeam)
	awayStrength := calculateTeamStrength(awayTeam)

	// Home advantage factor (1.1 multiplier)
	homeStrength *= 1.1

	// Calculate total strength
	totalStrength := homeStrength + awayStrength

	// Base probabilities
	homeWin = homeStrength / totalStrength
	awayWin = awayStrength / totalStrength
	draw = 0.3 * (1.0 - math.Abs(homeWin-awayWin)) // More likely to draw if teams are evenly matched

	// Normalize probabilities
	total := homeWin + draw + awayWin
	homeWin /= total
	draw /= total
	awayWin /= total

	return homeWin, draw, awayWin
}

func calculateAttackStrength(team *TeamInfo) float64 {
	baseStrength := 0.5

	// Adjust based on form
	formAdjustment := float64(team.FormPoints) * 0.05

	// Consider home/away streak
	streakAdjustment := 0.0
	if team.HomeStreak > 0 {
		streakAdjustment += float64(team.HomeStreak) * 0.1
	} else if team.HomeStreak < 0 {
		streakAdjustment += float64(team.HomeStreak) * 0.1
	}

	return baseStrength + formAdjustment + streakAdjustment
}

func updateTeamForm(team *TeamInfo, result string, isHome bool) {
	// Add result to form (most recent first)
	team.Form = append([]string{result}, team.Form...)
	if len(team.Form) > 5 {
		team.Form = team.Form[:5]
	}

	// Recalculate form points
	team.FormPoints = 0
	for _, r := range team.Form {
		switch r {
		case FormWin:
			team.FormPoints += 3
		case FormDraw:
			team.FormPoints += 1
		case FormLoss:
			team.FormPoints += 0
		}
	}

	// Update streaks
	if isHome {
		if result == FormWin {
			if team.HomeStreak >= 0 {
				team.HomeStreak++
			} else {
				team.HomeStreak = 1
			}
		} else {
			if team.HomeStreak <= 0 {
				team.HomeStreak--
			} else {
				team.HomeStreak = -1
			}
		}
	} else {
		if result == FormWin {
			if team.AwayStreak >= 0 {
				team.AwayStreak++
			} else {
				team.AwayStreak = 1
			}
		} else {
			if team.AwayStreak <= 0 {
				team.AwayStreak--
			} else {
				team.AwayStreak = -1
			}
		}
	}

	log.Printf("📊 %s form updated: %v (Points: %d, Home streak: %d, Away streak: %d)",
		team.ShortName, team.Form, team.FormPoints, team.HomeStreak, team.AwayStreak)
}

// Calculate team strength based on player ratings and form
func calculateTeamStrength(team *TeamInfo) float64 {
	// Get all players for the team
	teamPlayers := getPlayersFromTeam(team.ID)
	if len(teamPlayers) == 0 {
		return 0.5 // Default strength if no players found
	}

	// Calculate average player rating
	totalRating := 0.0
	for _, player := range teamPlayers {
		totalRating += player.CurrentRating
	}
	avgRating := totalRating / float64(len(teamPlayers))

	// Calculate form impact (0.8 to 1.2 multiplier)
	formMultiplier := 1.0
	if len(team.Form) > 0 {
		formPoints := team.FormPoints
		maxPossiblePoints := 15 // 5 matches * 3 points
		formMultiplier = 0.8 + (float64(formPoints)/float64(maxPossiblePoints))*0.4
	}

	// Calculate team characteristics impact
	characteristicsImpact := 0.0
	for _, player := range teamPlayers {
		characteristicsImpact += float64(player.Characteristics.Overall) / 100.0
	}
	characteristicsImpact /= float64(len(teamPlayers))

	// Combine all factors
	teamStrength := (avgRating / 10.0) * formMultiplier * (0.7 + characteristicsImpact*0.3)
	return math.Max(0.1, math.Min(1.0, teamStrength))
}

func getTableData(w http.ResponseWriter, r *http.Request) {
	tableType := r.URL.Query().Get("type")
	if tableType == "" {
		tableType = "matches" // Default view
	}

	// Get page number from query parameter
	pageStr := r.URL.Query().Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	// Get league selection for league tables
	selectedLeague := r.URL.Query().Get("league")
	if selectedLeague == "" {
		selectedLeague = "Premier League" // Default league
	}

	// Items per page
	const itemsPerPage = 10

	mutex.RLock()
	defer mutex.RUnlock()

	// Common CSS styles for consistent formatting
	const styles = `
		<style>
			body {
				font-family: 'Segoe UI', system-ui, -apple-system, sans-serif;
				line-height: 1.5;
				margin: 0;
				padding: 20px;
				background: #f8f9fa;
			}
			.container {
				max-width: 1200px;
				margin: 0 auto;
				background: white;
				padding: 20px;
				border-radius: 8px;
				box-shadow: 0 2px 4px rgba(0,0,0,0.1);
			}
			.table-nav {
				margin-bottom: 1rem;
				display: flex;
				gap: 0.5rem;
				flex-wrap: wrap;
			}
			.table-nav a {
				padding: 6px 12px;
				border: 1px solid #dee2e6;
				border-radius: 4px;
				color: #495057;
				text-decoration: none;
				font-size: 0.875rem;
			}
			.table-nav a:hover {
				background-color: #f8f9fa;
			}
			.table-nav a.active {
				background-color: #007bff;
				color: white;
				border-color: #007bff;
			}
			.league-selector {
				margin-bottom: 1rem;
				padding: 1rem;
				background: #f8f9fa;
				border-radius: 4px;
			}
			.league-selector label {
				margin-right: 0.5rem;
				font-weight: 600;
			}
			.league-selector select {
				padding: 4px 8px;
				border: 1px solid #dee2e6;
				border-radius: 4px;
				margin-right: 1rem;
			}
			table {
				width: 100%;
				border-collapse: collapse;
				margin: 1rem 0;
				font-size: 0.875rem;
				background: white;
			}
			th {
				background: #f8f9fa;
				padding: 8px;
				text-align: left;
				font-weight: 600;
				color: #495057;
				border-bottom: 2px solid #dee2e6;
				white-space: nowrap;
			}
			td {
				padding: 6px 8px;
				border-bottom: 1px solid #dee2e6;
				color: #212529;
			}
			tr:hover {
				background-color: #f8f9fa;
			}
			.form-win {
				color: #28a745;
				font-weight: bold;
			}
			.form-draw {
				color: #ffc107;
				font-weight: bold;
			}
			.form-loss {
				color: #dc3545;
				font-weight: bold;
			}
			.status-live {
				color: #28a745;
				font-weight: bold;
			}
			.status-finished {
				color: #6c757d;
			}
			.status-halftime {
				color: #ffc107;
				font-weight: bold;
			}
			.pagination {
				display: flex;
				justify-content: center;
				align-items: center;
				margin: 1rem 0;
				gap: 0.5rem;
			}
			.pagination a {
				padding: 4px 8px;
				border: 1px solid #dee2e6;
				border-radius: 4px;
				color: #495057;
				text-decoration: none;
				font-size: 0.875rem;
			}
			.pagination a:hover {
				background-color: #f8f9fa;
			}
			.pagination .active {
				background-color: #007bff;
				color: white;
				border-color: #007bff;
			}
			.pagination .disabled {
				color: #6c757d;
				pointer-events: none;
			}
			.info-box {
				background: #e7f3ff;
				border: 1px solid #b3d9ff;
				border-radius: 4px;
				padding: 0.75rem;
				margin-bottom: 1rem;
				font-size: 0.875rem;
			}
		</style>
	`

	// Navigation links for different tables
	navLinks := fmt.Sprintf(`
		<div class="table-nav">
			<a href="?type=matches" class="%s">Matches</a>
			<a href="?type=teams" class="%s">Teams</a>
			<a href="?type=players" class="%s">Players</a>
			<a href="?type=league-tables&league=%s" class="%s">League Tables</a>
			<a href="?type=season-stats" class="%s">Season Stats</a>
		</div>
	`,
		getActiveClass(tableType == "matches"),
		getActiveClass(tableType == "teams"),
		getActiveClass(tableType == "players"),
		selectedLeague,
		getActiveClass(tableType == "league-tables"),
		getActiveClass(tableType == "season-stats"),
	)

	// Generate table HTML based on type
	var tableHTML string
	var totalItems int
	switch tableType {
	case "matches":
		tableHTML, totalItems = generateMatchesTable(page, itemsPerPage)
	case "teams":
		tableHTML, totalItems = generateTeamsTable(page, itemsPerPage)
	case "players":
		tableHTML, totalItems = generatePlayersTable(page, itemsPerPage)
	case "league-tables":
		// Add league selector for league tables
		extraContent := fmt.Sprintf(`
			<div class="league-selector">
				<label for="league-select">Select League:</label>
				<select id="league-select" onchange="changeLeague()">
					<option value="Premier League"%s>Premier League</option>
					<option value="La Liga"%s>La Liga</option>
				</select>
				<div class="info-box">
					<strong>Home Streak:</strong> Consecutive wins/losses at home stadium<br>
					<strong>Away Streak:</strong> Consecutive wins/losses away from home<br>
					<strong>Form:</strong> Results from last 5 matches (W=Win, D=Draw, L=Loss)
				</div>
			</div>
			<script>
				function changeLeague() {
					const select = document.getElementById('league-select');
					const selectedLeague = select.value;
					window.location.href = '?type=league-tables&league=' + encodeURIComponent(selectedLeague) + '&page=1';
				}
			</script>
		`,
			map[bool]string{true: " selected", false: ""}[selectedLeague == "Premier League"],
			map[bool]string{true: " selected", false: ""}[selectedLeague == "La Liga"],
		)
		tableHTML, totalItems = generateLeagueTablesTableForLeague(selectedLeague, page, itemsPerPage)
		tableHTML = extraContent + tableHTML
	case "season-stats":
		tableHTML, totalItems = generateSeasonStatsTable(page, itemsPerPage)
	default:
		tableHTML, totalItems = generateMatchesTable(page, itemsPerPage)
	}

	// Calculate pagination
	totalPages := (totalItems + itemsPerPage - 1) / itemsPerPage
	pagination := generatePagination(page, totalPages, tableType, selectedLeague)

	// Combine everything into final HTML
	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>MatchPulse Data Tables</title>
			%s
		</head>
		<body>
			<div class="table-header">
				<h1 class="table-title">MatchPulse Data Tables</h1>
				<div class="table-info">Showing %d-%d of %d items</div>
			</div>
			%s
			<div class="table-container">
				%s
				%s
			</div>
		</body>
		</html>
	`, styles,
		(page-1)*itemsPerPage+1,
		min(page*itemsPerPage, totalItems),
		totalItems,
		navLinks,
		tableHTML,
		pagination)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func generatePagination(currentPage, totalPages int, tableType, selectedLeague string) string {
	if totalPages <= 1 {
		return ""
	}

	var html strings.Builder
	html.WriteString(`<div class="pagination">`)

	// Build base URL with league parameter for league tables
	baseURL := fmt.Sprintf("?type=%s", tableType)
	if tableType == "league-tables" && selectedLeague != "" {
		baseURL += fmt.Sprintf("&league=%s", selectedLeague)
	}

	// Previous page
	if currentPage > 1 {
		html.WriteString(fmt.Sprintf(`<a href="%s&page=%d">Previous</a>`, baseURL, currentPage-1))
	}

	// Page numbers
	startPage := max(1, currentPage-2)
	endPage := min(totalPages, currentPage+2)

	if startPage > 1 {
		html.WriteString(fmt.Sprintf(`<a href="%s&page=1">1</a>`, baseURL))
		if startPage > 2 {
			html.WriteString(`<span>...</span>`)
		}
	}

	for i := startPage; i <= endPage; i++ {
		if i == currentPage {
			html.WriteString(fmt.Sprintf(`<a href="%s&page=%d" class="active">%d</a>`, baseURL, i, i))
		} else {
			html.WriteString(fmt.Sprintf(`<a href="%s&page=%d">%d</a>`, baseURL, i, i))
		}
	}

	if endPage < totalPages {
		if endPage < totalPages-1 {
			html.WriteString(`<span>...</span>`)
		}
		html.WriteString(fmt.Sprintf(`<a href="%s&page=%d">%d</a>`, baseURL, totalPages, totalPages))
	}

	// Next page
	if currentPage < totalPages {
		html.WriteString(fmt.Sprintf(`<a href="%s&page=%d">Next</a>`, baseURL, currentPage+1))
	}

	html.WriteString(`</div>`)
	return html.String()
}

func generateLeagueTablesTableForLeague(league string, page, itemsPerPage int) (string, int) {
	var html strings.Builder
	html.WriteString(`
		<table>
			<tr>
				<th>Pos</th>
				<th>Team</th>
				<th>P</th>
				<th>W</th>
				<th>D</th>
				<th>L</th>
				<th>GF</th>
				<th>GA</th>
				<th>GD</th>
				<th>Pts</th>
				<th>Form</th>
			</tr>
	`)

	if table, exists := leagueTables[league]; exists {
		// Calculate start and end indices for pagination
		start := (page - 1) * itemsPerPage
		end := start + itemsPerPage

		if start < len(table) {
			if end > len(table) {
				end = len(table)
			}
			for _, entry := range table[start:end] {
				formHTML := ""
				for _, result := range entry.Form {
					class := ""
					switch result {
					case FormWin:
						class = "form-win"
					case FormDraw:
						class = "form-draw"
					case FormLoss:
						class = "form-loss"
					}
					formHTML += fmt.Sprintf(`<span class="%s">%s</span> `, class, result)
				}

				html.WriteString(fmt.Sprintf(`
					<tr>
						<td>%d</td>
						<td>%s</td>
						<td>%d</td>
						<td>%d</td>
						<td>%d</td>
						<td>%d</td>
						<td>%d</td>
						<td>%d</td>
						<td>%d</td>
						<td>%d</td>
						<td>%s</td>
					</tr>
				`, entry.Position, entry.Team.ShortName, entry.Played,
					entry.Won, entry.Drawn, entry.Lost, entry.GoalsFor,
					entry.GoalsAgainst, entry.GoalDiff, entry.Points, formHTML))
			}
		}

		html.WriteString("</table>")
		return html.String(), len(table)
	}

	html.WriteString("</table>")
	return html.String(), 0
}

func getActiveClass(isActive bool) string {
	if isActive {
		return "active"
	}
	return ""
}

func logInfo(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	log.Printf(message)          // Console output
	addLogEntry("INFO", message) // Capture for API
}

func addLogEntry(level, message string) {
	// Store in logEntries slice for /api/v1/logs endpoint
	// Keep only last 1000 entries as specified
}

func generateMatchesTable(page, itemsPerPage int) (string, int) {
	var html strings.Builder
	html.WriteString(`
		<table>
			<tr>
				<th>ID</th>
				<th>Home</th>
				<th>Score</th>
				<th>Away</th>
				<th>Min</th>
				<th>Status</th>
				<th>Comp</th>
				<th>Venue</th>
				<th>Weather</th>
			</tr>
	`)

	// Calculate start and end indices for pagination
	start := (page - 1) * itemsPerPage
	end := start + itemsPerPage

	// Convert matches map to slice for pagination
	var matchList []*Match
	for _, match := range matches {
		matchList = append(matchList, match)
	}

	// Sort matches by ID
	sort.Slice(matchList, func(i, j int) bool {
		return matchList[i].ID < matchList[j].ID
	})

	// Apply pagination
	if start < len(matchList) {
		if end > len(matchList) {
			end = len(matchList)
		}
		for _, match := range matchList[start:end] {
			statusClass := ""
			switch match.Status {
			case StatusLive:
				statusClass = "status-live"
			case StatusFinished:
				statusClass = "status-finished"
			case StatusHalftime:
				statusClass = "status-halftime"
			}

			html.WriteString(fmt.Sprintf(`
				<tr>
					<td>%d</td>
					<td>%s</td>
					<td>%d-%d</td>
					<td>%s</td>
					<td>%d'</td>
					<td class="%s">%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
				</tr>
			`, match.ID, match.HomeTeam.ShortName, match.HomeScore, match.AwayScore,
				match.AwayTeam.ShortName, match.Minute, statusClass, match.Status,
				match.Competition, match.Venue, match.Weather))
		}
	}

	html.WriteString("</table>")
	return html.String(), len(matchList)
}

func generateTeamsTable(page, itemsPerPage int) (string, int) {
	var html strings.Builder
	html.WriteString(`
		<table>
			<tr>
				<th>ID</th>
				<th>Name</th>
				<th>League</th>
				<th>Manager</th>
				<th>Form</th>
				<th>Form Points</th>
				<th>Home Streak</th>
				<th>Away Streak</th>
			</tr>
	`)

	// Calculate start and end indices for pagination
	start := (page - 1) * itemsPerPage
	end := start + itemsPerPage

	// Convert teams map to slice for pagination
	var teamList []*TeamInfo
	for _, team := range teams {
		teamList = append(teamList, team)
	}

	// Sort teams by ID
	sort.Slice(teamList, func(i, j int) bool {
		return teamList[i].ID < teamList[j].ID
	})

	// Apply pagination
	if start < len(teamList) {
		if end > len(teamList) {
			end = len(teamList)
		}
		for _, team := range teamList[start:end] {
			formHTML := ""
			for _, result := range team.Form {
				class := ""
				switch result {
				case FormWin:
					class = "form-win"
				case FormDraw:
					class = "form-draw"
				case FormLoss:
					class = "form-loss"
				}
				formHTML += fmt.Sprintf(`<span class="%s">%s</span> `, class, result)
			}

			html.WriteString(fmt.Sprintf(`
				<tr>
					<td>%d</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%d</td>
					<td>%d</td>
					<td>%d</td>
				</tr>
			`, team.ID, team.Name, team.League, team.Manager, formHTML,
				team.FormPoints, team.HomeStreak, team.AwayStreak))
		}
	}

	html.WriteString("</table>")
	return html.String(), len(teamList)
}

func generatePlayersTable(page, itemsPerPage int) (string, int) {
	var html strings.Builder
	html.WriteString(`
		<table>
			<tr>
				<th>ID</th>
				<th>Name</th>
				<th>Position</th>
				<th>Team</th>
				<th>Age</th>
				<th>Rating</th>
				<th>Goals</th>
				<th>Assists</th>
				<th>Cards</th>
			</tr>
	`)

	// Calculate start and end indices for pagination
	start := (page - 1) * itemsPerPage
	end := start + itemsPerPage

	// Convert players map to slice for pagination
	var playerList []*Player
	for _, player := range players {
		playerList = append(playerList, player)
	}

	// Sort players by ID
	sort.Slice(playerList, func(i, j int) bool {
		return playerList[i].ID < playerList[j].ID
	})

	// Apply pagination
	if start < len(playerList) {
		if end > len(playerList) {
			end = len(playerList)
		}
		for _, player := range playerList[start:end] {
			team := teams[player.TeamID]
			teamName := "Unknown"
			if team != nil {
				teamName = team.ShortName
			}

			html.WriteString(fmt.Sprintf(`
				<tr>
					<td>%d</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%d</td>
					<td>%.1f</td>
					<td>%d</td>
					<td>%d</td>
					<td>%dY %dR</td>
				</tr>
			`, player.ID, player.Name, player.Position, teamName,
				player.Age, player.CurrentRating, player.Goals, player.Assists,
				player.YellowCards, player.RedCards))
		}
	}

	html.WriteString("</table>")
	return html.String(), len(playerList)
}

func generateSeasonStatsTable(page, itemsPerPage int) (string, int) {
	var html strings.Builder
	html.WriteString(`
		<table>
			<tr>
				<th>Season</th>
				<th>Champion</th>
				<th>Top Scorer</th>
				<th>Goals</th>
				<th>Top Assists</th>
				<th>Assists</th>
				<th>Player of Season</th>
				<th>Rating</th>
				<th>Total Goals</th>
				<th>Total Matches</th>
			</tr>
	`)

	// Calculate start and end indices for pagination
	start := (page - 1) * itemsPerPage
	end := start + itemsPerPage

	// Create a copy of season history for pagination
	historyList := make([]SeasonHistory, len(seasonHistory))
	copy(historyList, seasonHistory)

	// Sort history by season (descending - most recent first)
	sort.Slice(historyList, func(i, j int) bool {
		return historyList[i].Season > historyList[j].Season
	})

	// Apply pagination
	if start < len(historyList) {
		if end > len(historyList) {
			end = len(historyList)
		}
		for _, history := range historyList[start:end] {
			html.WriteString(fmt.Sprintf(`
				<tr>
					<td>%d</td>
					<td>%s</td>
					<td>%s</td>
					<td>%d</td>
					<td>%s</td>
					<td>%d</td>
					<td>%s</td>
					<td>%.1f</td>
					<td>%d</td>
					<td>%d</td>
				</tr>
			`, history.Season, history.Champion.ShortName,
				history.TopScorer.Name, history.TopScorer.SeasonStats.GoalsThisSeason,
				history.TopAssists.Name, history.TopAssists.SeasonStats.AssistsThisSeason,
				history.PlayerOfSeason.Name, history.PlayerOfSeason.SeasonStats.AverageRating,
				history.TotalGoals, history.TotalMatches))
		}
	}

	html.WriteString("</table>")
	return html.String(), len(historyList)
}
