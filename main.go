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
	LeaguePremier         = "Premier League"
	LeagueCommunityLeague = "Community League"

	// League configuration
	TeamsPerLeague   = 10 // Each league has 10 teams
	MatchdaysPerWeek = 2  // 2 matchdays per week
	MatchesPerTeam   = 18 // Each team plays 38 matches (9 home, 9 away)

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
	EventThrowIn      = "THROW_IN"
	EventPenalty      = "PENALTY"
	EventFreekick     = "FREEKICK"

	// Ball event states
	BallEventPlay     = "PLAY"
	BallEventKickoff  = "KICKOFF"
	BallEventFreekick = "FREEKICK"
	BallEventCorner   = "CORNER"
	BallEventThrowIn  = "THROW_IN"
	BallEventPenalty  = "PENALTY"
	BallEventGoalkick = "GOAL_KICK"

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

	// Offensive tactics
	TacticTikiTaka      = "TIKI_TAKA"
	TacticCounterAttack = "COUNTER_ATTACK"
	TacticDirectPlay    = "DIRECT_PLAY"
	TacticWingPlay      = "WING_PLAY"
	TacticPressing      = "HIGH_PRESSING"

	// Defensive tactics
	TacticCompactDefense = "COMPACT_DEFENSE"
	TacticManMarking     = "MAN_MARKING"
	TacticZonalMarking   = "ZONAL_MARKING"
	TacticOffside        = "OFFSIDE_TRAP"
	TacticLowBlock       = "LOW_BLOCK"

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

	// Player availability statuses
	PlayerAvailable   = "available"
	PlayerRedCard     = "red_card"
	PlayerInjured     = "injured"
	PlayerSubstituted = "substituted"
)

// Match momentum and dynamics
type MatchMomentum struct {
	HomeTeamMomentum     float64   `json:"home_momentum"` // -1.0 to 1.0
	AwayTeamMomentum     float64   `json:"away_momentum"` // -1.0 to 1.0
	LastGoalTime         int       `json:"last_goal_time"`
	LastGoalTeam         int       `json:"last_goal_team"`
	LastRedCardTime      int       `json:"last_red_card_time"`
	LastRedCardTeam      int       `json:"last_red_card_team"`
	ConsecutiveGoals     int       `json:"consecutive_goals"`
	LastUpdateTime       time.Time `json:"last_update"`
	PressureIndex        float64   `json:"pressure_index"` // 0.0 to 1.0
	FormationAdjustments int       `json:"formation_adjustments"`
}

// Player availability during matches
type PlayerAvailability struct {
	PlayerID        int    `json:"player_id"`
	Status          string `json:"status"`           // available, red_card, injured, substituted
	UnavailableFrom int    `json:"unavailable_from"` // minute when became unavailable
	Reason          string `json:"reason"`
}

// Dynamic match probabilities that change during the game
type DynamicMatchProbabilities struct {
	MatchID             int                `json:"match_id"`
	HomeWinProbability  float64            `json:"home_win_prob"`
	DrawProbability     float64            `json:"draw_prob"`
	AwayWinProbability  float64            `json:"away_win_prob"`
	NextGoalProbability float64            `json:"next_goal_prob"`
	HomeNextGoalProb    float64            `json:"home_next_goal_prob"`
	AwayNextGoalProb    float64            `json:"away_next_goal_prob"`
	LastUpdate          time.Time          `json:"last_update"`
	Factors             map[string]float64 `json:"factors"` // What's influencing the probabilities
}

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

type BallPosition struct {
	X            float64   `json:"x"`
	Y            float64   `json:"y"`
	PossessorID  int       `json:"possessor_id"`  // Player ID who has the ball
	LastTouchID  int       `json:"last_touch_id"` // Previous possessor (for assists)
	Speed        float64   `json:"speed"`
	Direction    float64   `json:"direction"`
	Timestamp    time.Time `json:"timestamp"`
	EventType    string    `json:"event_type"`    // Current ball event (PLAY, FREEKICK, CORNER, etc.)
	EventStarted time.Time `json:"event_started"` // When current event started
}

var (
	ballPositions = make(map[int]*BallPosition) // matchID -> ball position
	matchTactics  = make(map[int]*MatchTactics) // matchID -> tactics
)

type MatchTactics struct {
	HomeOffensive string `json:"home_offensive"`
	HomeDefensive string `json:"home_defensive"`
	AwayOffensive string `json:"away_offensive"`
	AwayDefensive string `json:"away_defensive"`
}

type FormationPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
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
	TotalMatches     int       `json:"total_matches"`
	TotalGoals       int       `json:"total_goals"`
	AverageGoals     float64   `json:"average_goals"`
	MostGoalsMatch   int       `json:"most_goals_match_id"`
	ActiveViewers    int       `json:"active_viewers"`
	TopScorer        Player    `json:"top_scorer"`
	MostActiveMatch  int       `json:"most_active_match_id"`
	LastUpdate       time.Time `json:"last_update"`
	CurrentSeason    int       `json:"current_season"`
	CurrentMatchweek int       `json:"current_matchweek"`
	SeasonProgress   float64   `json:"season_progress"`
}

type SeasonWinners struct {
	PremierLeagueWinner   *TeamInfo `json:"premier_league_winner"`
	CommunityLeagueWinner *TeamInfo `json:"community_league_winner"`
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
		EndTeamID:   10,
	},
	LeagueCommunityLeague: {
		Name:        LeagueCommunityLeague,
		TeamCount:   TeamsPerLeague,
		Matchdays:   MatchesPerTeam,
		StartTeamID: 11,
		EndTeamID:   20,
	},
}

// In-memory database - add season schedules
var (
	// Original storage
	matches         = make(map[int]*Match)
	finishedMatches = make(map[int]*Match) // New map for finished matches
	matchStats      = make(map[int]*MatchStats)
	players         = make(map[int]*Player)
	teams           = make(map[int]*TeamInfo)
	leagueTables    = make(map[string][]*LeagueTable)
	liveCommentary  = make(map[int][]*LiveCommentary)
	globalStats     = &GlobalStats{}

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

	// Enhanced simulation variables
	matchMomentum        = make(map[int]*MatchMomentum)              // MatchID -> Momentum
	playerAvailability   = make(map[int]map[int]*PlayerAvailability) // MatchID -> PlayerID -> Availability
	dynamicProbabilities = make(map[int]*DynamicMatchProbabilities)  // MatchID -> Probabilities

	// Add this at the top of the file with other global variables
	startTime = time.Now()

	// Add scheduling synchronization
	schedulingMutex = &sync.Mutex{} // Separate mutex for schedule operations
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
	// Premier League - 10 Teams
	{1, "Capricon FC", "CAP", "Stellar Stadium", LeaguePremier, "Viktor Cosmos", 2180},
	{2, "The Galacticons", "GAL", "Nebula Arena", LeaguePremier, "Zara Starfield", 2175},
	{3, "Axton Brothers", "AXT", "Quantum Park", LeaguePremier, "Rex Axiom", 2182},
	{4, "Deuteron United", "DEU", "Fusion Field", LeaguePremier, "Nova Nucleus", 2178},
	{5, "Saturn Rovers", "SAT", "Ring Stadium", LeaguePremier, "Luna Orbit", 2179},
	{6, "Meteor City", "MET", "Impact Zone", LeaguePremier, "Comet Trail", 2181},
	{7, "Cosmic Wanderers", "COS", "Infinity Ground", LeaguePremier, "Astro Nova", 2177},
	{8, "Pulsar Athletic", "PUL", "Photon Arena", LeaguePremier, "Ray Beacon", 2183},
	{9, "Nebula FC", "NEB", "Star Dust Stadium", LeaguePremier, "Cloud Walker", 2176},
	{10, "Eclipse United", "ECL", "Shadow Grounds", LeaguePremier, "Dark Matter", 2184},

	// Community League - 10 Teams
	{11, "Nova Dynamics", "NOV", "Quantum Field", LeagueCommunityLeague, "Atlas Prime", 2178},
	{12, "Starlight FC", "SFC", "Celestial Arena", LeagueCommunityLeague, "Vega Solaris", 2181},
	{13, "Orion Warriors", "ORI", "Constellation Park", LeagueCommunityLeague, "Leo Sterling", 2176},
	{14, "Zenith United", "ZEN", "Horizon Stadium", LeagueCommunityLeague, "Aurora Borealis", 2183},
	{15, "Quasar City", "QUA", "Plasma Ground", LeagueCommunityLeague, "Sirius Flux", 2179},
	{16, "Astral Rovers", "AST", "Galaxy Dome", LeagueCommunityLeague, "Helios Star", 2182},
	{17, "Eclipse Knights", "EKN", "Shadow Field", LeagueCommunityLeague, "Umbra Knight", 2177},
	{18, "Neutron FC", "NEU", "Energy Arena", LeagueCommunityLeague, "Proton Wave", 2180},
	{19, "Vortex Athletic", "VOR", "Cyclone Stadium", LeagueCommunityLeague, "Tempest Storm", 2175},
	{20, "Cosmic Rangers", "COS", "Meteor Ground", LeagueCommunityLeague, "Comet Chase", 2184},
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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

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
		}
	}

	playerID := 1
	for _, team := range teams {
		// Realistic squad composition: 18-20 players per team
		positions := []string{
			PosGK, PosGK, // 2 goalkeepers
			PosCB, PosCB, PosCB, // 3 center backs
			PosLB, PosLB, // 2 left backs
			PosRB, PosRB, // 2 right backs
			PosCDM, PosCDM, // 2 defensive mids
			PosCM, PosCM, PosCM, // 3 central mids
			PosCAM,       // 1 attacking mid
			PosLW, PosRW, // 2 wingers
			PosST, PosST, // 2 strikers
		}

		for i, position := range positions {
			playerTemplate := playerNames[rand.Intn(len(playerNames))]
			name := fmt.Sprintf("%s %d", playerTemplate.Name, i+1)

			characteristics := generatePlayerCharacteristics(position)

			players[playerID] = &Player{
				ID:              playerID,
				Name:            name,
				Position:        position, // Use actual position, not template
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

	// Log fixture summary
	logFixtureSummary()

	updateGlobalStats()

	// Create initial matches if none exists - fill up to MaxSimultaneousMatches per league
	if len(matches) == 0 {
		log.Printf("🏁 Creating initial matches up to maximum (%d per league)...", MaxSimultaneousMatches)
		matchesCreated := 0

		// Create matches for each league
		for _, league := range []string{LeaguePremier, LeagueCommunityLeague} {
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

	// Goroutines working in parallel
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
	ball := ballPositions[matchID]
	if ball == nil {
		return
	}

	// O(1) scoring: whoever has the ball scores (realistic!)
	var scorer *Player
	if ball.PossessorID > 0 {
		if player, exists := players[ball.PossessorID]; exists {
			// Only attackers and midfielders can score in attacking third
			if ball.X > 70 || ball.X < 30 { // In attacking third
				if player.Position != PosGK && rand.Float64() < getGoalProbability(player, ball) {
					scorer = player
				}
			}
		}
	}

	// If no possessor or GK has ball, find nearest attacking player
	if scorer == nil {
		scorer = findNearestAttackingPlayer(matchID, ball)
	}

	if scorer != nil {
		isHomeGoal := scorer.TeamID == match.HomeTeam.ID
		if isHomeGoal {
			match.HomeScore++
		} else {
			match.AwayScore++
		}

		// Update scorer stats
		scorer.Goals++
		scorer.SeasonStats.GoalsThisSeason++
		scorer.CurrentRating += 2.0

		// O(1) assist: previous ball possessor gets assist
		var assister *Player
		if ball.LastTouchID != 0 && ball.LastTouchID != scorer.ID {
			if player, exists := players[ball.LastTouchID]; exists &&
				player.TeamID == scorer.TeamID { // Same team
				assister = player
				assister.Assists++
				assister.SeasonStats.AssistsThisSeason++
				assister.CurrentRating += 1.0
			}
		}

		// Update momentum - goals significantly impact the game
		updateMatchMomentum(matchID, match, "goal", scorer.TeamID)

		// Recalculate match probabilities after goal
		recalculateMatchProbabilities(matchID, match)

		// Reset ball for kickoff
		setBallEvent(matchID, BallEventKickoff, FieldWidth/2, FieldHeight/2, 0)

		// Commentary with assist info
		commentary := fmt.Sprintf("GOAL! %s scores!", scorer.Name)
		if assister != nil {
			commentary += fmt.Sprintf(" Assisted by %s.", assister.Name)
		}
		addLiveCommentary(matchID, match.Minute, commentary, EventGoal, scorer)
	}
}

func getGoalProbability(player *Player, ball *BallPosition) float64 {
	baseProbability := 0.25

	// Determine which goal we're attacking
	goalX := FieldWidth
	if ball.X < FieldWidth/2 {
		goalX = 0
	}
	goalY := FieldHeight / 2

	// Distance from goal (most important factor)
	distanceToGoal := math.Sqrt(math.Pow(ball.X-goalX, 2) + math.Pow(ball.Y-goalY, 2))

	// Position-based multiplier (natural ability)
	positionMultiplier := 1.0
	switch player.Position {
	case PosST:
		positionMultiplier = 1.8
	case PosCAM, PosLW, PosRW:
		positionMultiplier = 1.4
	case PosCM:
		positionMultiplier = 1.0
	case PosCDM:
		positionMultiplier = 0.7
	case PosLB, PosRB:
		positionMultiplier = 0.5
	case PosCB:
		positionMultiplier = 0.4
	case PosGK:
		positionMultiplier = 0.02
	}

	// Location-based multiplier (opportunity)
	locationMultiplier := 1.0
	if distanceToGoal < 8 { // In the box
		locationMultiplier = 2.5
	} else if distanceToGoal < 15 { // Close to box
		locationMultiplier = 1.8
	} else if distanceToGoal < 25 { // Edge of area
		locationMultiplier = 1.2
	} else if distanceToGoal > 40 { // Very far
		locationMultiplier = 0.3
	}

	// Angle to goal (center is better)
	angleToCenterGoal := math.Abs(ball.Y - goalY)
	if angleToCenterGoal < 5 { // Central position
		locationMultiplier *= 1.3
	} else if angleToCenterGoal > 15 { // Wide angle
		locationMultiplier *= 0.7
	}

	// If player is in unusual attacking position, boost chance
	isInAttackingPosition := false
	if player.TeamID == getHomeTeamID() && ball.X > FieldWidth*0.6 {
		isInAttackingPosition = true
	} else if player.TeamID != getHomeTeamID() && ball.X < FieldWidth*0.4 {
		isInAttackingPosition = true
	}

	if isInAttackingPosition && (player.Position == PosCM || player.Position == PosCDM ||
		player.Position == PosLB || player.Position == PosRB) {
		locationMultiplier *= 1.5 // Reward unexpected attackers
	}

	finalProbability := baseProbability * positionMultiplier * locationMultiplier
	return math.Min(0.9, math.Max(0.01, finalProbability))
}

func getHomeTeamID() int {
	// Helper to identify home team (assumes first match)
	for _, match := range matches {
		return match.HomeTeam.ID
	}
	return 1
}

func findNearestAttackingPlayer(matchID int, ball *BallPosition) *Player {
	var nearestPlayer *Player
	minDistance := math.Inf(1)

	// Get the match to determine which team is attacking
	match, exists := matches[matchID]
	if !exists {
		return nil
	}

	// Determine which team is attacking based on ball position
	// If ball is in the right half (x > 50), home team is attacking
	// If ball is in the left half (x < 50), away team is attacking
	var attackingTeamID int
	if ball.X > 50 {
		attackingTeamID = match.HomeTeam.ID
	} else {
		attackingTeamID = match.AwayTeam.ID
	}

	if locations, exists := playerLocations[matchID]; exists {
		for playerID, location := range locations {
			if player, exists := players[playerID]; exists {
				// Only consider attacking players from the attacking team
				if player.TeamID == attackingTeamID &&
					(player.Position == PosST || player.Position == PosCAM ||
						player.Position == PosLW || player.Position == PosRW) {
					distance := math.Sqrt(math.Pow(location.X-ball.X, 2) + math.Pow(location.Y-ball.Y, 2))
					if distance < minDistance && distance < 15 {
						minDistance = distance
						nearestPlayer = player
					}
				}
			}
		}
	}

	return nearestPlayer
}

func handleCardEvent(matchID int, match *Match) {
	// Only select from available players
	availablePlayers := getAvailablePlayersForMatch(matchID)
	if len(availablePlayers) == 0 {
		return
	}

	player := availablePlayers[rand.Intn(len(availablePlayers))]
	isHomePlayer := player.TeamID == match.HomeTeam.ID

	cardType := "yellow"
	if rand.Float32() < 0.1 { // 10% chance for red card
		cardType = "red"
		player.RedCards++
		player.SeasonStats.RedCardsThisSeason++
		player.CurrentRating -= 2.0

		// Mark player as unavailable due to red card
		setPlayerUnavailable(matchID, player.ID, PlayerRedCard, match.Minute, "Direct red card")

		// Update momentum - red card affects team morale
		updateMatchMomentum(matchID, match, "red_card", player.TeamID)

		// Recalculate match probabilities
		recalculateMatchProbabilities(matchID, match)

		logInfo("🟥 RED CARD! %s receives a red card and is sent off!", player.Name)

		addLiveCommentary(matchID, match.Minute,
			fmt.Sprintf("RED CARD! %s is sent off! %s down to 10 men!",
				player.Name,
				getTeamName(player.TeamID)),
			EventCard, player)
	} else {
		player.YellowCards++
		player.SeasonStats.YellowCardsThisSeason++
		player.CurrentRating -= 0.5
		logInfo("🟨 YELLOW CARD! %s receives a yellow card", player.Name)

		addLiveCommentary(matchID, match.Minute,
			fmt.Sprintf("Yellow card shown to %s", player.Name),
			EventCard, player)
	}

	// Update match stats
	if isHomePlayer {
		if cardType == "red" {
			matchStats[matchID].HomeRedCards++
		} else {
			matchStats[matchID].HomeYellowCards++
		}
	} else {
		if cardType == "red" {
			matchStats[matchID].AwayRedCards++
		} else {
			matchStats[matchID].AwayYellowCards++
		}
	}
}

// Player availability management
func getAvailablePlayersForMatch(matchID int) []*Player {
	var available []*Player
	match := matches[matchID]
	if match == nil {
		return available
	}

	// Get all players from both teams
	homePlayers := getPlayersFromTeam(match.HomeTeam.ID)
	awayPlayers := getPlayersFromTeam(match.AwayTeam.ID)
	allPlayers := append(homePlayers, awayPlayers...)

	// Filter out unavailable players
	for _, player := range allPlayers {
		if isPlayerAvailable(matchID, player.ID) {
			available = append(available, player)
		}
	}
	return available
}

func isPlayerAvailable(matchID, playerID int) bool {
	if playerAvailability[matchID] == nil {
		return true
	}
	availability := playerAvailability[matchID][playerID]
	return availability == nil || availability.Status == PlayerAvailable
}

func setPlayerUnavailable(matchID, playerID int, status string, minute int, reason string) {
	if playerAvailability[matchID] == nil {
		playerAvailability[matchID] = make(map[int]*PlayerAvailability)
	}

	playerAvailability[matchID][playerID] = &PlayerAvailability{
		PlayerID:        playerID,
		Status:          status,
		UnavailableFrom: minute,
		Reason:          reason,
	}
}

func getTeamName(teamID int) string {
	if team := teams[teamID]; team != nil {
		return team.ShortName
	}
	return "Unknown"
}

// Match momentum management
func updateMatchMomentum(matchID int, match *Match, eventType string, affectedTeamID int) {
	if matchMomentum[matchID] == nil {
		matchMomentum[matchID] = &MatchMomentum{
			HomeTeamMomentum: 0.0,
			AwayTeamMomentum: 0.0,
			LastUpdateTime:   time.Now(),
		}
	}

	momentum := matchMomentum[matchID]
	isHomeTeam := affectedTeamID == match.HomeTeam.ID

	switch eventType {
	case "goal":
		if isHomeTeam {
			momentum.HomeTeamMomentum += 0.3
			momentum.AwayTeamMomentum -= 0.2
		} else {
			momentum.AwayTeamMomentum += 0.3
			momentum.HomeTeamMomentum -= 0.2
		}
		momentum.LastGoalTime = match.Minute
		momentum.LastGoalTeam = affectedTeamID
		momentum.ConsecutiveGoals++

	case "red_card":
		if isHomeTeam {
			momentum.HomeTeamMomentum -= 0.4
			momentum.AwayTeamMomentum += 0.2
		} else {
			momentum.AwayTeamMomentum -= 0.4
			momentum.HomeTeamMomentum += 0.2
		}
		momentum.LastRedCardTime = match.Minute
		momentum.LastRedCardTeam = affectedTeamID

	case "corner":
		adjustment := 0.1
		if isHomeTeam {
			momentum.HomeTeamMomentum += adjustment
		} else {
			momentum.AwayTeamMomentum += adjustment
		}

	case "foul":
		adjustment := 0.05
		if isHomeTeam {
			momentum.HomeTeamMomentum -= adjustment
		} else {
			momentum.AwayTeamMomentum -= adjustment
		}
	}

	// Clamp momentum values between -1.0 and 1.0
	momentum.HomeTeamMomentum = math.Max(-1.0, math.Min(1.0, momentum.HomeTeamMomentum))
	momentum.AwayTeamMomentum = math.Max(-1.0, math.Min(1.0, momentum.AwayTeamMomentum))
	momentum.LastUpdateTime = time.Now()
}

// Dynamic probability calculation
func recalculateMatchProbabilities(matchID int, match *Match) {
	if dynamicProbabilities[matchID] == nil {
		dynamicProbabilities[matchID] = &DynamicMatchProbabilities{
			MatchID: matchID,
			Factors: make(map[string]float64),
		}
	}

	probs := dynamicProbabilities[matchID]
	momentum := matchMomentum[matchID]

	// Base probabilities from pre-match calculation
	baseHomeWin, _, baseAwayWin := calculateMatchProbabilities(&match.HomeTeam, &match.AwayTeam)

	// Adjust for current score
	scoreDiff := match.HomeScore - match.AwayScore
	scoreAdjustment := float64(scoreDiff) * 0.1

	// Adjust for player numbers (red cards)
	homePlayerCount := 11 - getRedCardCount(matchID, match.HomeTeam.ID)
	awayPlayerCount := 11 - getRedCardCount(matchID, match.AwayTeam.ID)
	playerDiff := float64(homePlayerCount-awayPlayerCount) * 0.15

	// Adjust for momentum
	momentumAdjustment := 0.0
	if momentum != nil {
		momentumAdjustment = (momentum.HomeTeamMomentum - momentum.AwayTeamMomentum) * 0.2
	}

	// Adjust for time remaining - more conservative as time runs out
	timeRemaining := 90 - match.Minute
	timeMultiplier := 1.0
	if timeRemaining < 15 {
		timeMultiplier = 0.5 + (float64(timeRemaining) / 30.0)
	}

	// Calculate adjusted probabilities
	totalAdjustment := (scoreAdjustment + playerDiff + momentumAdjustment) * timeMultiplier

	probs.HomeWinProbability = math.Max(0.05, math.Min(0.9, baseHomeWin+totalAdjustment))
	probs.AwayWinProbability = math.Max(0.05, math.Min(0.9, baseAwayWin-totalAdjustment))
	probs.DrawProbability = math.Max(0.05, 1.0-probs.HomeWinProbability-probs.AwayWinProbability)

	// Normalize
	total := probs.HomeWinProbability + probs.DrawProbability + probs.AwayWinProbability
	probs.HomeWinProbability /= total
	probs.DrawProbability /= total
	probs.AwayWinProbability /= total

	// Calculate next goal probabilities
	probs.HomeNextGoalProb = calculateNextGoalProbability(match, true, momentum)
	probs.AwayNextGoalProb = calculateNextGoalProbability(match, false, momentum)
	probs.NextGoalProbability = probs.HomeNextGoalProb + probs.AwayNextGoalProb

	// Store contributing factors
	probs.Factors["score_diff"] = scoreAdjustment
	probs.Factors["player_diff"] = playerDiff
	probs.Factors["momentum"] = momentumAdjustment
	probs.Factors["time_multiplier"] = timeMultiplier

	probs.LastUpdate = time.Now()
}

func getRedCardCount(matchID, teamID int) int {
	count := 0
	if playerAvailability[matchID] != nil {
		for playerID, availability := range playerAvailability[matchID] {
			if availability.Status == PlayerRedCard {
				player := getPlayerByID(playerID)
				if player != nil && player.TeamID == teamID {
					count++
				}
			}
		}
	}
	return count
}

func getPlayerByID(playerID int) *Player {
	return players[playerID]
}

func calculateNextGoalProbability(match *Match, isHome bool, momentum *MatchMomentum) float64 {
	baseProb := 0.02 // 2% base chance per minute

	// Adjust for team attacking strength
	var team *TeamInfo
	if isHome {
		team = &match.HomeTeam
	} else {
		team = &match.AwayTeam
	}

	attackStrength := calculateAttackStrength(team)
	baseProb *= attackStrength * 2.0

	// Adjust for momentum
	if momentum != nil {
		if isHome {
			baseProb *= (1.0 + momentum.HomeTeamMomentum*0.5)
		} else {
			baseProb *= (1.0 + momentum.AwayTeamMomentum*0.5)
		}
	}

	// Adjust for player count
	playerCount := 11 - getRedCardCount(match.ID, team.ID)
	if playerCount < 11 {
		baseProb *= (float64(playerCount) / 11.0)
	}

	return math.Max(0.001, math.Min(0.1, baseProb))
}

// Cooldown and next match creation
func startCooldownAndCreateNext(finishedMatchID int) {
	logInfo("⏳ Starting %d-second post-match break for match %d...", PostMatchBreakSeconds, finishedMatchID)
	time.Sleep(PostMatchBreakSeconds * time.Second)

	mutex.Lock()
	// Move finished match to finishedMatches map instead of deleting
	if match, exists := matches[finishedMatchID]; exists {
		finishedMatches[finishedMatchID] = match
		delete(matches, finishedMatchID)
		logInfo("📦 Moved finished match %d to finished matches storage", finishedMatchID)
	}

	// Create next match
	logInfo("🆕 Creating next match after post-match break...")
	createNextMatch()
	mutex.Unlock()
}

func createNextMatch() {
	// Use separate mutex for scheduling to prevent race conditions
	schedulingMutex.Lock()
	defer schedulingMutex.Unlock()

	// Get next scheduled match
	scheduledMatch := getNextUnplayedMatch()
	if scheduledMatch == nil {
		log.Printf("⚠️  No more scheduled matches available")
		return
	}

	// Double-check the match isn't already being played (race condition fix)
	if scheduledMatch.IsPlayed {
		log.Printf("⚠️  Match already marked as played: %s vs %s",
			scheduledMatch.HomeTeam.ShortName, scheduledMatch.AwayTeam.ShortName)
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

	// Mark schedule as being played IMMEDIATELY to prevent race conditions
	scheduledMatch.IsPlayed = true
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

	// Update schedule with match ID
	scheduledMatch.MatchID = matchCounter

	matches[matchCounter] = match
	matchStats[matchCounter] = generateInitialMatchStats(matchCounter)
	liveCommentary[matchCounter] = []*LiveCommentary{}
	playerLocations[matchCounter] = make(map[int]*PlayerLocation)

	// Initialize enhanced simulation data
	matchMomentum[matchCounter] = &MatchMomentum{
		HomeTeamMomentum: 0.0,
		AwayTeamMomentum: 0.0,
		LastUpdateTime:   time.Now(),
	}

	dynamicProbabilities[matchCounter] = &DynamicMatchProbabilities{
		MatchID:             matchCounter,
		HomeWinProbability:  homeWin,
		DrawProbability:     draw,
		AwayWinProbability:  awayWin,
		NextGoalProbability: 0.02,
		HomeNextGoalProb:    0.01,
		AwayNextGoalProb:    0.01,
		LastUpdate:          time.Now(),
		Factors:             make(map[string]float64),
	}

	playerAvailability[matchCounter] = make(map[int]*PlayerAvailability)

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
	logInfo("🏆 Season manager started - checking every 5 minutes")
	// Check more frequently for season end
	seasonTicker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer seasonTicker.Stop()

	for {
		select {
		case <-seasonTicker.C:
			logInfo("🗓️  Season check: Season %d, Week %d", currentSeason, currentMatchweek)
			if shouldEndSeason() {
				logInfo("🏁 Ending season %d...", currentSeason)
				endSeason()
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
	if match.Status == StatusFinished {
		// Update matchweek only when all matches for current matchweek are finished
		allMatchesFinished := true
		for _, league := range []string{LeaguePremier, LeagueCommunityLeague} {
			if schedules := getScheduledMatches(league, currentMatchweek); len(schedules) > 0 {
				for _, schedule := range schedules {
					if !schedule.IsPlayed {
						allMatchesFinished = false
						break
					}
				}
			}
		}

		if allMatchesFinished {
			currentMatchweek++
			logInfo("📅 Advancing to matchweek %d", currentMatchweek)

			// Check if season should end after advancing matchweek
			if shouldEndSeason() {
				logInfo("🏁 Season complete! Starting season transition...")
				endSeason()
				startNewSeason()
			}
		}
	}

	// Update team stats
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

// HTTP Handlers
func getAllMatches(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	status := r.URL.Query().Get("status")
	league := r.URL.Query().Get("league")
	teamIDStr := r.URL.Query().Get("team_id")

	mutex.RLock()
	matchList := make([]*Match, 0, len(matches)+len(finishedMatches))

	// Helper function to check if match should be included
	shouldInclude := func(match *Match) bool {
		if status != "" {
			// Special handling for "finished" status
			if status == "finished" {
				if match.Status != StatusFinished {
					return false
				}
			} else if status == "live" {
				if match.Status != StatusLive {
					return false
				}
			} else if match.Status != status {
				return false
			}
		}
		if league != "" && match.Competition != league {
			return false
		}
		if teamIDStr != "" {
			teamID, err := strconv.Atoi(teamIDStr)
			if err == nil && match.HomeTeam.ID != teamID && match.AwayTeam.ID != teamID {
				return false
			}
		}
		return true
	}

	// Add active matches
	for _, match := range matches {
		if shouldInclude(match) {
			matchList = append(matchList, match)
		}
	}

	// Add finished matches
	for _, match := range finishedMatches {
		if shouldInclude(match) {
			matchList = append(matchList, match)
		}
	}
	mutex.RUnlock()

	// Sort matches by ID for consistent ordering
	sort.Slice(matchList, func(i, j int) bool {
		return matchList[i].ID < matchList[j].ID
	})

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

	// Get only the first 11 players from each team
	homeTeamID := matches[id].HomeTeam.ID
	awayTeamID := matches[id].AwayTeam.ID

	homeCount := 0
	awayCount := 0
	locationList := make([]*PlayerLocation, 0, 22) // Pre-allocate for 22 players

	for _, location := range locations {
		if player, exists := players[location.PlayerID]; exists {
			if player.TeamID == homeTeamID && homeCount < 11 {
				locationList = append(locationList, location)
				homeCount++
			} else if player.TeamID == awayTeamID && awayCount < 11 {
				locationList = append(locationList, location)
				awayCount++
			}
		}
	}

	// Get ball position
	ball := ballPositions[id]
	mutex.RUnlock()

	response := map[string]interface{}{
		"locations": locationList,
		"match_id":  id,
		"count":     len(locationList),
		"timestamp": time.Now(),
	}

	// Add ball position if it exists
	if ball != nil {
		response["ball"] = ball
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getMatchMomentum(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	mutex.RLock()
	momentum := matchMomentum[id]
	mutex.RUnlock()

	response := map[string]interface{}{
		"match_id":  id,
		"timestamp": time.Now(),
	}

	if momentum != nil {
		response["momentum"] = momentum
	} else {
		response["momentum"] = nil
		response["message"] = "No momentum data available for this match"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getMatchProbabilities(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	mutex.RLock()
	probs := dynamicProbabilities[id]
	match := matches[id]
	mutex.RUnlock()

	response := map[string]interface{}{
		"match_id":  id,
		"timestamp": time.Now(),
	}

	if probs != nil {
		response["probabilities"] = probs
	} else if match != nil {
		// Calculate initial probabilities if none exist
		homeWin, draw, awayWin := calculateMatchProbabilities(&match.HomeTeam, &match.AwayTeam)
		response["probabilities"] = map[string]interface{}{
			"home_win_prob": homeWin,
			"draw_prob":     draw,
			"away_win_prob": awayWin,
			"calculated_at": time.Now(),
			"note":          "Initial pre-match probabilities",
		}
	} else {
		response["probabilities"] = nil
		response["message"] = "Match not found"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getMatchAvailability(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	mutex.RLock()
	availability := playerAvailability[id]
	match := matches[id]
	mutex.RUnlock()

	if match == nil {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}

	// Get available and unavailable players
	availablePlayers := getAvailablePlayersForMatch(id)
	var unavailablePlayers []map[string]interface{}

	for _, avail := range availability {
		if avail.Status != PlayerAvailable {
			player := getPlayerByID(avail.PlayerID)
			if player != nil {
				unavailablePlayers = append(unavailablePlayers, map[string]interface{}{
					"player":           player,
					"status":           avail.Status,
					"unavailable_from": avail.UnavailableFrom,
					"reason":           avail.Reason,
				})
			}
		}
	}

	response := map[string]interface{}{
		"match_id":            id,
		"available_count":     len(availablePlayers),
		"unavailable_count":   len(unavailablePlayers),
		"unavailable_players": unavailablePlayers,
		"home_player_count":   11 - getRedCardCount(id, match.HomeTeam.ID),
		"away_player_count":   11 - getRedCardCount(id, match.AwayTeam.ID),
		"timestamp":           time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

// For goroutine monitoring - can be hooked to a grafana dashboard
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
		"name":            "MatchPulse API",
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	mutex.RLock()
	// Find first live match ID
	var firstLiveMatchID int
	for id, match := range matches {
		if match.Status == StatusLive {
			firstLiveMatchID = id
			break
		}
	}

	templateData := struct {
		ActiveMatches    int
		TotalPlayers     int
		TotalTeams       int
		CurrentSeason    int
		CurrentMatchweek int
		LastUpdated      string
		Version          string
		FirstLiveMatchID int
	}{
		ActiveMatches:    len(matches),
		TotalPlayers:     len(players),
		TotalTeams:       len(teams),
		CurrentSeason:    currentSeason,
		CurrentMatchweek: currentMatchweek,
		LastUpdated:      time.Now().Format("2006-01-02 15:04:05"),
		Version:          version,
		FirstLiveMatchID: firstLiveMatchID,
	}
	mutex.RUnlock()

	const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>MatchPulse API v{{.Version}}</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            box-sizing: border-box;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 12px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            min-height: calc(100vh - 40px);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 2px solid #f1f3f4;
        }
        .header h1 {
            font-size: 2.5rem;
            margin: 0 0 10px 0;
            color: #333;
        }
        .header p {
            color: #6c757d;
            font-size: 1.1rem;
            margin: 0;
        }
        .status {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin: 25px 0;
        }
        .status-item {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
            box-shadow: 0 4px 15px rgba(0,0,0,0.1);
        }
        .status-value {
            font-size: 2rem;
            font-weight: bold;
            margin-bottom: 5px;
        }
        .status-label {
            font-size: 0.9rem;
            opacity: 0.9;
        }
        .content {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 25px;
        }
        .section {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.05);
        }
        .section h2 {
            margin: 0 0 20px 0;
            color: #495057;
            font-size: 1.4rem;
            padding-bottom: 10px;
            border-bottom: 2px solid #dee2e6;
        }
        .endpoint-group {
            margin-bottom: 20px;
        }
        .endpoint-group h3 {
            margin: 0 0 12px 0;
            font-size: 1.1rem;
            color: #6c757d;
            background: #e9ecef;
            padding: 8px 12px;
            border-radius: 5px;
            border-left: 4px solid #007bff;
        }
        ul {
            margin: 0;
            padding: 0;
            list-style: none;
        }
        li {
            margin-bottom: 8px;
            padding: 8px 12px;
            background: white;
            border-radius: 5px;
            border-left: 3px solid #007bff;
            transition: all 0.2s ease;
        }
        li:hover {
            background: #f0f8ff;
            border-left-color: #0056b3;
            transform: translateX(5px);
        }
        a {
            color: #007bff;
            text-decoration: none;
            font-weight: 500;
            display: block;
        }
        a:hover {
            color: #0056b3;
        }
        .new-feature {
            position: relative;
        }
        .new-feature::after {
            content: "NEW";
            background: #28a745;
            color: white;
            font-size: 0.7rem;
            padding: 2px 6px;
            border-radius: 10px;
            position: absolute;
            right: 10px;
            top: 50%;
            transform: translateY(-50%);
        }
        .enhanced-feature {
            position: relative;
        }
        .enhanced-feature::after {
            content: "ENHANCED";
            background: #ffc107;
            color: #212529;
            font-size: 0.7rem;
            padding: 2px 6px;
            border-radius: 10px;
            position: absolute;
            right: 10px;
            top: 50%;
            transform: translateY(-50%);
        }
        .footer {
            text-align: center;
            margin-top: 30px;
            padding-top: 20px;
            border-top: 2px solid #f1f3f4;
            color: #6c757d;
        }
        .api-schema-link {
            display: inline-block;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 12px 24px;
            border-radius: 25px;
            text-decoration: none;
            font-weight: bold;
            margin-top: 15px;
            transition: transform 0.2s ease;
        }
        .api-schema-link:hover {
            transform: translateY(-2px);
            color: white;
        }
        .github-link {
            display: inline-block;
            background: #24292e;
            color: white;
            padding: 12px 24px;
            border-radius: 25px;
            text-decoration: none;
            font-weight: bold;
            margin-top: 15px;
            margin-left: 10px;
            transition: transform 0.2s ease;
        }
        .github-link:hover {
            transform: translateY(-2px);
            color: white;
        }
        .fixtures-link {
            display: inline-block;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 12px 24px;
            border-radius: 25px;
            text-decoration: none;
            font-weight: bold;
            margin-top: 15px;
            margin-right: 10px;
            transition: transform 0.2s ease;
        }
        .fixtures-link:hover {
            transform: translateY(-2px);
            color: white;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚽ MatchPulse API v{{.Version}}</h1>
            <p>Real-time Football Simulation with Enhanced Match Dynamics</p>
            <div class="status">
                <div class="status-item">
                    <div class="status-value">{{.ActiveMatches}}</div>
                    <div class="status-label">Active Matches</div>
                </div>
                <div class="status-item">
                    <div class="status-value">{{.TotalPlayers}}</div>
                    <div class="status-label">Total Players</div>
                </div>
                <div class="status-item">
                    <div class="status-value">{{.TotalTeams}}</div>
                    <div class="status-label">Total Teams</div>
                </div>
                <div class="status-item">
                    <div class="status-value">Season {{.CurrentSeason}}</div>
                    <div class="status-label">Current Season</div>
                </div>
                <div class="status-item">
                    <div class="status-value">Week {{.CurrentMatchweek}}</div>
                    <div class="status-label">Current Matchweek</div>
                </div>
            </div>
        </div>
        <div class="content">
            <div class="section">
                <h2>🏈 Match Endpoints</h2>
                <div class="endpoint-group">
                    <h3>Core Match Data</h3>
                    <ul>
                        <li><a href="/api/v1/matches">All Matches</a></li>
                        <li><a href="/api/v1/matches?status=live">Live Matches</a></li>
                        <li><a href="/api/v1/matches?status=finished">Finished Matches</a></li>
                        <li><a href="/api/v1/matches?league=Premier%20League">Premier League</a></li>
                        <li><a href="/api/v1/matches?league=Community%20League">Community League</a></li>
                        <li><a href="/api/v1/matches/1">Match Details</a></li>
                        <li><a href="/api/v1/matches/1/stats">Match Statistics</a></li>
                        <li><a href="/api/v1/matches/1/commentary">Live Commentary</a></li>
                    </ul>
                </div>
                <div class="endpoint-group">
                    <h3>Enhanced Match Features</h3>
                    <ul>
                        <li class="new-feature"><a href="/api/v1/matches/{{.FirstLiveMatchID}}/momentum">Match Momentum</a></li>
                        <li class="new-feature"><a href="/api/v1/matches/{{.FirstLiveMatchID}}/probabilities">Win Probabilities</a></li>
                        <li class="new-feature"><a href="/api/v1/matches/{{.FirstLiveMatchID}}/availability">Player Availability</a></li>
                        <li class="enhanced-feature"><a href="/api/v1/matches/{{.FirstLiveMatchID}}/players">Player Positions</a></li>
                    </ul>
                </div>
            </div>
            <div class="section">
                <h2>👥 Players & Teams</h2>
                <div class="endpoint-group">
                    <h3>Player Data</h3>
                    <ul>
                        <li><a href="/api/v1/players">All Players</a></li>
                        <li><a href="/api/v1/players?position=ST">Strikers</a></li>
                        <li><a href="/api/v1/players?position=GK">Goalkeepers</a></li>
                        <li><a href="/api/v1/players?team=1">Team Players</a></li>
                        <li><a href="/api/v1/players/1">Player Details</a></li>
                    </ul>
                </div>
                <div class="endpoint-group">
                    <h3>Team Data</h3>
                    <ul>
                        <li><a href="/api/v1/teams">All Teams</a></li>
                        <li><a href="/api/v1/teams?league=Premier%20League">Premier League Teams</a></li>
                        <li><a href="/api/v1/teams?league=La%20Liga">Community League Teams</a></li>
                        <li><a href="/api/v1/teams/1">Team Details</a></li>
                        <li><a href="/api/v1/teams/1/form">Team Form</a></li>
                    </ul>
                </div>
            </div>
            <div class="section">
                <h2>🏆 League & Season</h2>
                <div class="endpoint-group">
                    <h3>League Tables</h3>
                    <ul>
                        <li><a href="/api/v1/leagues/Premier%20League/table" class="league-link">Premier League</a></li>
                        <li><a href="/api/v1/leagues/Community%20League/table" class="league-link">Community League</a></li>
                        <li><a href="/api/v1/leagues/Premier%20League/form">Premier League Form</a></li>
                        <li><a href="/api/v1/leagues/Community%20League/form">Community League Form</a></li>
                        <li><a href="/api/v1/leagues/Premier%20League/schedule">Premier League Schedule</a></li>
                    </ul>
                </div>
                <div class="endpoint-group">
                    <h3>Season Management</h3>
                    <ul>
                        <li><a href="/api/v1/seasons/current">Current Season</a></li>
                        <li><a href="/api/v1/seasons/history">Season History</a></li>
                        <li><a href="/api/v1/seasons/current/matchdays/1">Matchday 1</a></li>
                        <li><a href="/api/v1/seasons/current/matchdays/{{.CurrentMatchweek}}">Current Matchday</a></li>
                    </ul>
                </div>
            </div>
            <div class="section">
                <h2>🔧 System & Utilities</h2>
                <div class="endpoint-group">
                    <h3>System Status</h3>
                    <ul>
                        <li><a href="/api/v1/health">Health Check</a></li>
                        <li><a href="/api/v1/stats">Global Statistics</a></li>
                        <li><a href="/api/v1/search?q=Johnson">Search API</a></li>
                    </ul>
                </div>
                <div class="endpoint-group">
                    <h3>Data Tables (HTML)</h3>
                    <ul>
                        <li><a href="/tables?type=matches">Matches Table</a></li>
                        <li><a href="/tables?type=teams">Teams Table</a></li>
                        <li><a href="/tables?type=players">Players Table</a></li>
                        <li><a href="/tables?type=league-tables">League Tables</a></li>
                        <li><a href="/tables?type=season-stats">Season Stats</a></li>
                    </ul>
                </div>
            </div>
            <div class="section">
                <h2>📅 Fixtures & Schedule</h2>
                <div class="endpoint-group">
                    <h3>Match Schedule</h3>
                    <ul>
                        <li><a href="/fixtures">View All Fixtures</a></li>
                        <li><a href="/api/v1/fixtures">All Fixtures (API)</a></li>
                        <li><a href="/api/v1/fixtures/Premier%20League">Premier League Fixtures</a></li>
                        <li><a href="/api/v1/fixtures/Community%20League">Community League Fixtures</a></li>
                        <li><a href="/api/v1/seasons/current/matchdays/{{.CurrentMatchweek}}">Current Matchday</a></li>
                    </ul>
                </div>
            </div>
        </div>
        <div class="footer">
            <p><strong>🚀 New Features:</strong> Real-time momentum tracking • Dynamic match probabilities • Player availability system • Enhanced red card handling</p>
            <p><strong>API Version:</strong> v{{.Version}} | <strong>Last Updated:</strong> {{.LastUpdated}}</p>
            <a href="/fixtures" class="fixtures-link">View Fixtures</a>
            <a href="/api-schema.txt" class="api-schema-link">Download API Schema</a>
            <a href="https://github.com/tobimadehin/matchpulse-api" class="github-link">View on GitHub</a>
        </div>
    </div>
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
	// Check if all matches in all leagues are finished
	for _, league := range []string{LeaguePremier, LeagueCommunityLeague} {
		if schedules, exists := seasonSchedules[league]; exists {
			for _, schedule := range schedules {
				if !schedule.IsPlayed {
					return false
				}
			}
		}
	}
	return true
}

func startNewSeason() {
	logInfo("🎬 Starting new season %d...", currentSeason+1)
	currentSeason++
	currentMatchweek = 1

	// Clear finished matches at the start of new season
	mutex.Lock()
	finishedMatches = make(map[int]*Match)
	mutex.Unlock()

	// Reset other season-related data
	// ... existing code ...
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

// Get all fixtures for all leagues - allows viewing the complete season schedule upfront
func getAllFixtures(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	defer mutex.RUnlock()

	allFixtures := make(map[string][]*SeasonSchedule)
	totalFixtures := 0

	// Get all fixtures for each league
	for league, schedules := range seasonSchedules {
		allFixtures[league] = schedules
		totalFixtures += len(schedules)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"fixtures":       allFixtures,
		"total_fixtures": totalFixtures,
		"current_season": currentSeason,
		"timestamp":      time.Now(),
	})
}

// Get all fixtures for a specific league
func getLeagueFixtures(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	league := vars["league"]

	mutex.RLock()
	schedules, exists := seasonSchedules[league]
	mutex.RUnlock()

	if !exists {
		http.Error(w, "League not found", http.StatusNotFound)
		return
	}

	// Optional filtering
	statusFilter := r.URL.Query().Get("status") // "played", "unplayed", "all"
	matchdayStr := r.URL.Query().Get("matchday")

	var filteredSchedules []*SeasonSchedule

	for _, schedule := range schedules {
		// Filter by status if requested
		if statusFilter != "" && statusFilter != "all" {
			if statusFilter == "played" && !schedule.IsPlayed {
				continue
			}
			if statusFilter == "unplayed" && schedule.IsPlayed {
				continue
			}
		}

		// Filter by matchday if requested
		if matchdayStr != "" {
			matchday, err := strconv.Atoi(matchdayStr)
			if err == nil && schedule.Matchday != matchday {
				continue
			}
		}

		filteredSchedules = append(filteredSchedules, schedule)
	}

	// Count played and unplayed
	playedCount := 0
	unplayedCount := 0
	for _, schedule := range schedules {
		if schedule.IsPlayed {
			playedCount++
		} else {
			unplayedCount++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"league":         league,
		"fixtures":       filteredSchedules,
		"total_fixtures": len(schedules),
		"played_count":   playedCount,
		"unplayed_count": unplayedCount,
		"filtered_count": len(filteredSchedules),
		"current_season": currentSeason,
		"timestamp":      time.Now(),
	})
}

func getMatchTactics(matchID int) *MatchTactics {
	if tactics, exists := matchTactics[matchID]; exists {
		return tactics
	}

	// Generate random tactics if not exists
	offensiveTactics := []string{TacticTikiTaka, TacticCounterAttack, TacticDirectPlay, TacticWingPlay, TacticPressing}
	defensiveTactics := []string{TacticCompactDefense, TacticManMarking, TacticZonalMarking, TacticOffside, TacticLowBlock}

	tactics := &MatchTactics{
		HomeOffensive: offensiveTactics[rand.Intn(len(offensiveTactics))],
		HomeDefensive: defensiveTactics[rand.Intn(len(defensiveTactics))],
		AwayOffensive: offensiveTactics[rand.Intn(len(offensiveTactics))],
		AwayDefensive: defensiveTactics[rand.Intn(len(defensiveTactics))],
	}

	matchTactics[matchID] = tactics
	return tactics
}

func setBallEvent(matchID int, eventType string, x, y float64, possessorID int) {
	if ball := ballPositions[matchID]; ball != nil {
		ball.X = x
		ball.Y = y
		ball.PossessorID = possessorID
		ball.EventType = eventType
		ball.EventStarted = time.Now()
		ball.Speed = 0
		ball.Timestamp = time.Now()

		// Reposition players for the event
		repositionPlayersForEvent(matchID, eventType, x, y)
	}
}

func repositionPlayersForEvent(matchID int, eventType string, ballX, ballY float64) {
	match := matches[matchID]
	if match == nil {
		return
	}

	switch eventType {
	case BallEventKickoff:
		repositionForKickoff(matchID, match)
	case BallEventCorner:
		repositionForCorner(matchID, match, ballX, ballY)
	case BallEventFreekick:
		repositionForFreekick(matchID, match, ballX, ballY)
	case BallEventThrowIn:
		repositionForThrowIn(matchID, match, ballX, ballY)
	case BallEventPenalty:
		repositionForPenalty(matchID, match, ballX, ballY)
	case BallEventGoalkick:
		repositionForGoalkick(matchID, match, ballX, ballY)
	}
}

func repositionForKickoff(matchID int, match *Match) {
	// Reset to formation positions
	homePositions := getFormationPositions(match.HomeFormation, true)
	awayPositions := getFormationPositions(match.AwayFormation, false)

	homePlayers := getPlayersFromTeam(match.HomeTeam.ID)[:11]
	awayPlayers := getPlayersFromTeam(match.AwayTeam.ID)[:11]

	// Position home team
	for i, player := range homePlayers {
		if i < len(homePositions) {
			playerLocations[matchID][player.ID] = &PlayerLocation{
				PlayerID:  player.ID,
				X:         homePositions[i].X,
				Y:         homePositions[i].Y,
				Timestamp: time.Now(),
			}
		}
	}

	// Position away team
	for i, player := range awayPlayers {
		if i < len(awayPositions) {
			playerLocations[matchID][player.ID] = &PlayerLocation{
				PlayerID:  player.ID,
				X:         awayPositions[i].X,
				Y:         awayPositions[i].Y,
				Timestamp: time.Now(),
			}
		}
	}
}

func repositionForCorner(matchID int, match *Match, ballX, ballY float64) {
	isHomeCorner := ballX > FieldWidth/2

	if isHomeCorner {
		// Home team attacking corner
		repositionTeamForAttackingCorner(matchID, match.HomeTeam.ID, ballX, ballY)
		repositionTeamForDefendingCorner(matchID, match.AwayTeam.ID, ballX, ballY)
	} else {
		// Away team attacking corner
		repositionTeamForAttackingCorner(matchID, match.AwayTeam.ID, ballX, ballY)
		repositionTeamForDefendingCorner(matchID, match.HomeTeam.ID, ballX, ballY)
	}
}

func repositionTeamForAttackingCorner(matchID int, teamID int, ballX, ballY float64) {
	players := getPlayersFromTeam(teamID)[:11]
	goalX := FieldWidth
	if ballX < FieldWidth/2 {
		goalX = 0
	}

	for i, player := range players {
		var x, y float64

		switch player.Position {
		case PosLW, PosRW, PosCM:
			if i == 0 { // Corner taker
				x, y = ballX, ballY
			} else {
				// Position in penalty area
				x = goalX - 10 + rand.Float64()*15
				y = FieldHeight/2 - 10 + rand.Float64()*20
			}
		case PosST, PosCAM:
			// Position near goal
			x = goalX - 8 + rand.Float64()*12
			y = FieldHeight/2 - 8 + rand.Float64()*16
		case PosCDM, PosCB:
			// Stay back for defensive cover
			x = FieldWidth/2 - 10 + rand.Float64()*20
			y = FieldHeight/2 - 15 + rand.Float64()*30
		case PosGK:
			// Stay in goal
			x = goalX - FieldWidth + 5
			y = FieldHeight / 2
		default:
			// Default positioning
			x = goalX - 15 + rand.Float64()*20
			y = FieldHeight/2 - 12 + rand.Float64()*24
		}

		playerLocations[matchID][player.ID] = &PlayerLocation{
			PlayerID:  player.ID,
			X:         math.Max(0, math.Min(FieldWidth, x)),
			Y:         math.Max(0, math.Min(FieldHeight, y)),
			Timestamp: time.Now(),
		}
	}
}

func repositionTeamForDefendingCorner(matchID int, teamID int, ballX, ballY float64) {
	players := getPlayersFromTeam(teamID)[:11]
	goalX := 0.0
	if ballX < FieldWidth/2 {
		goalX = FieldWidth
	}

	for _, player := range players {
		var x, y float64

		switch player.Position {
		case PosGK:
			// Stay in goal
			x = goalX + 5
			if goalX == 0 {
				x = 5
			} else {
				x = FieldWidth - 5
			}
			y = FieldHeight / 2
		case PosCB, PosLB, PosRB:
			// Mark attackers in penalty area
			x = goalX + 8 + rand.Float64()*8
			if goalX == 0 {
				x = 8 + rand.Float64()*8
			} else {
				x = FieldWidth - 16 + rand.Float64()*8
			}
			y = FieldHeight/2 - 12 + rand.Float64()*24
		case PosCDM, PosCM:
			// Cover edge of penalty area
			x = goalX + 18 + rand.Float64()*8
			if goalX == 0 {
				x = 18 + rand.Float64()*8
			} else {
				x = FieldWidth - 26 + rand.Float64()*8
			}
			y = FieldHeight/2 - 15 + rand.Float64()*30
		default:
			// Stay back defensively
			x = goalX + 15 + rand.Float64()*10
			if goalX == 0 {
				x = 15 + rand.Float64()*10
			} else {
				x = FieldWidth - 25 + rand.Float64()*10
			}
			y = FieldHeight/2 - 20 + rand.Float64()*40
		}

		playerLocations[matchID][player.ID] = &PlayerLocation{
			PlayerID:  player.ID,
			X:         math.Max(0, math.Min(FieldWidth, x)),
			Y:         math.Max(0, math.Min(FieldHeight, y)),
			Timestamp: time.Now(),
		}
	}
}

func repositionForFreekick(matchID int, match *Match, ballX, ballY float64) {
	// Determine attacking/defending teams based on ball position
	isHomeAttacking := ballX > FieldWidth/2
	var attackingTeamID, defendingTeamID int

	if isHomeAttacking {
		attackingTeamID = match.HomeTeam.ID
		defendingTeamID = match.AwayTeam.ID
	} else {
		attackingTeamID = match.AwayTeam.ID
		defendingTeamID = match.HomeTeam.ID
	}

	// Position attacking team
	attackingPlayers := getPlayersFromTeam(attackingTeamID)[:11]
	for i, player := range attackingPlayers {
		var x, y float64

		if i == 0 { // Free kick taker
			x, y = ballX, ballY
		} else if player.Position == PosST || player.Position == PosCAM {
			// Position for potential shot/cross
			goalX := FieldWidth
			if ballX < FieldWidth/2 {
				goalX = 0
			}
			x = goalX - 15 + rand.Float64()*10
			y = FieldHeight/2 - 8 + rand.Float64()*16
		} else {
			// Support positions
			x = ballX - 20 + rand.Float64()*15
			y = ballY - 10 + rand.Float64()*20
		}

		playerLocations[matchID][player.ID] = &PlayerLocation{
			PlayerID:  player.ID,
			X:         math.Max(0, math.Min(FieldWidth, x)),
			Y:         math.Max(0, math.Min(FieldHeight, y)),
			Timestamp: time.Now(),
		}
	}

	// Position defending team (wall + coverage)
	defendingPlayers := getPlayersFromTeam(defendingTeamID)[:11]
	wallDistance := 9.15 // FIFA regulation 10 yards

	for i, player := range defendingPlayers {
		var x, y float64

		if player.Position == PosGK {
			// Goalkeeper positioning
			goalX := 0.0
			if ballX < FieldWidth/2 {
				goalX = FieldWidth
			}
			x = goalX + 5.0
			if goalX == 0 {
				x = 5
			} else {
				x = FieldWidth - 5
			}
			y = FieldHeight / 2
		} else if i < 4 { // Wall players
			angle := math.Atan2(FieldHeight/2-ballY, (FieldWidth/2)-ballX)
			x = ballX + math.Cos(angle)*wallDistance
			y = ballY + math.Sin(angle)*wallDistance + float64(i-2)*2
		} else {
			// Mark attackers
			x = ballX + 10 + rand.Float64()*20
			y = ballY - 15 + rand.Float64()*30
		}

		playerLocations[matchID][player.ID] = &PlayerLocation{
			PlayerID:  player.ID,
			X:         math.Max(0, math.Min(FieldWidth, x)),
			Y:         math.Max(0, math.Min(FieldHeight, y)),
			Timestamp: time.Now(),
		}
	}
}

func repositionForThrowIn(matchID int, match *Match, ballX, ballY float64) {
	// Simple repositioning - players spread out along the line
	allPlayers := append(getPlayersFromTeam(match.HomeTeam.ID)[:11], getPlayersFromTeam(match.AwayTeam.ID)[:11]...)

	for i, player := range allPlayers {
		x := ballX - 10 + rand.Float64()*20
		y := ballY + float64(i-11)*3 // Spread along the line

		playerLocations[matchID][player.ID] = &PlayerLocation{
			PlayerID:  player.ID,
			X:         math.Max(0, math.Min(FieldWidth, x)),
			Y:         math.Max(0, math.Min(FieldHeight, y)),
			Timestamp: time.Now(),
		}
	}
}

func repositionForPenalty(matchID int, match *Match, ballX, ballY float64) {
	// Position all players outside penalty area except penalty taker and goalkeeper
	allPlayers := append(getPlayersFromTeam(match.HomeTeam.ID)[:11], getPlayersFromTeam(match.AwayTeam.ID)[:11]...)

	for i, player := range allPlayers {
		var x, y float64

		if player.Position == PosGK {
			// Goalkeeper on goal line
			goalX := 0.0
			if ballX > FieldWidth/2 {
				goalX = FieldWidth
			}
			x = goalX + 2.0
			if goalX == 0 {
				x = 2
			} else {
				x = FieldWidth - 2
			}
			y = FieldHeight / 2
		} else if i == 0 { // Penalty taker
			x, y = ballX, ballY
		} else {
			// Outside penalty area
			x = ballX - 20 + rand.Float64()*40
			y = ballY - 20 + rand.Float64()*40
		}

		playerLocations[matchID][player.ID] = &PlayerLocation{
			PlayerID:  player.ID,
			X:         math.Max(0, math.Min(FieldWidth, x)),
			Y:         math.Max(0, math.Min(FieldHeight, y)),
			Timestamp: time.Now(),
		}
	}
}

func repositionForGoalkick(matchID int, match *Match, ballX, ballY float64) {
	// Players spread out to receive goal kick
	allPlayers := append(getPlayersFromTeam(match.HomeTeam.ID)[:11], getPlayersFromTeam(match.AwayTeam.ID)[:11]...)

	for _, player := range allPlayers {
		var x, y float64

		if player.Position == PosGK {
			x, y = ballX, ballY
		} else {
			// Spread across the field
			x = 20 + rand.Float64()*(FieldWidth-40)
			y = 10 + rand.Float64()*(FieldHeight-20)
		}

		playerLocations[matchID][player.ID] = &PlayerLocation{
			PlayerID:  player.ID,
			X:         x,
			Y:         y,
			Timestamp: time.Now(),
		}
	}
}

func updateBallPhysics(matchID int, ball *BallPosition) {
	if ball == nil {
		return
	}

	// Handle ball movement based on current event type
	switch ball.EventType {
	case BallEventPlay:
		updateBallInPlay(matchID, ball)
	case BallEventKickoff:
		handleKickoffBallEvent(matchID, ball)
	case BallEventFreekick:
		handleFreekickBallEvent(matchID, ball)
	case BallEventCorner:
		handleCornerBallEvent(matchID, ball)
	case BallEventThrowIn:
		handleThrowInBallEvent(matchID, ball)
	case BallEventPenalty:
		handlePenaltyBallEvent(matchID, ball)
	case BallEventGoalkick:
		handleGoalkickBallEvent(matchID, ball)
	default:
		updateBallInPlay(matchID, ball)
	}

	ball.Timestamp = time.Now()
}

func updateBallInPlay(matchID int, ball *BallPosition) {
	if ball.PossessorID > 0 {
		// Ball follows player with possession
		if location, exists := playerLocations[matchID][ball.PossessorID]; exists {
			ball.X = location.X + (rand.Float64()-0.5)*3
			ball.Y = location.Y + (rand.Float64()-0.5)*3

			// Simulate passing - change possession occasionally
			if rand.Float64() < 0.1 { // 10% chance per update
				simulatePass(matchID, ball)
			}
		}
	} else {
		// Ball moves with physics when loose
		ball.Speed *= 0.92 // Friction
		ball.X += math.Cos(ball.Direction) * ball.Speed
		ball.Y += math.Sin(ball.Direction) * ball.Speed

		// Check if any player can pick up the ball
		if ball.Speed < 1.0 {
			nearestPlayer := findNearestPlayerToBall(matchID, ball)
			if nearestPlayer != nil {
				ball.LastTouchID = ball.PossessorID
				ball.PossessorID = nearestPlayer.ID
			}
		}
	}

	// Keep within bounds
	ball.X = math.Max(0, math.Min(FieldWidth, ball.X))
	ball.Y = math.Max(0, math.Min(FieldHeight, ball.Y))

	// Check for out of bounds events
	checkBallOutOfBounds(matchID, ball)
}

func simulatePass(matchID int, ball *BallPosition) {
	if ball.PossessorID == 0 {
		return
	}

	possessor, exists := players[ball.PossessorID]
	if !exists {
		return
	}

	// Find teammates for passing
	teammates := getTeammates(matchID, possessor.TeamID, ball.PossessorID)
	if len(teammates) == 0 {
		return
	}

	// Simple pass to random teammate
	target := teammates[rand.Intn(len(teammates))]
	if location, exists := playerLocations[matchID][target.ID]; exists {
		// Set ball direction toward target
		ball.Direction = math.Atan2(location.Y-ball.Y, location.X-ball.X)
		ball.Speed = 8.0 + rand.Float64()*4.0
		ball.LastTouchID = ball.PossessorID
		ball.PossessorID = 0 // Ball is in the air
	}
}

func findNearestPlayerToBall(matchID int, ball *BallPosition) *Player {
	var nearestPlayer *Player
	minDistance := 5.0 // Must be within 5 units

	if locations, exists := playerLocations[matchID]; exists {
		for playerID, location := range locations {
			distance := math.Sqrt(math.Pow(location.X-ball.X, 2) + math.Pow(location.Y-ball.Y, 2))
			if distance < minDistance {
				if player, exists := players[playerID]; exists {
					minDistance = distance
					nearestPlayer = player
				}
			}
		}
	}

	return nearestPlayer
}

func getTeammates(matchID int, teamID, excludePlayerID int) []*Player {
	var teammates []*Player

	if locations, exists := playerLocations[matchID]; exists {
		for playerID := range locations {
			if player, exists := players[playerID]; exists {
				if player.TeamID == teamID && player.ID != excludePlayerID {
					teammates = append(teammates, player)
				}
			}
		}
	}

	return teammates
}

func checkBallOutOfBounds(matchID int, ball *BallPosition) {
	match := matches[matchID]
	if match == nil {
		return
	}

	// Check for corner kicks
	if (ball.X <= 0 || ball.X >= FieldWidth) && (ball.Y >= 0 && ball.Y <= FieldHeight) {
		// Determine which team gets the corner/goal kick
		isHomeAttack := ball.X >= FieldWidth/2

		if ball.Y <= 5 || ball.Y >= FieldHeight-5 { // Near goal line
			if isHomeAttack {
				setBallEvent(matchID, BallEventCorner, FieldWidth-1, ball.Y, 0)
				addLiveCommentary(matchID, match.Minute, "Corner kick!", EventCorner, nil)
			} else {
				setBallEvent(matchID, BallEventGoalkick, 6, FieldHeight/2, 0)
				addLiveCommentary(matchID, match.Minute, "Goal kick", EventCommentary, nil)
			}
		}
	}

	// Check for throw-ins
	if ball.Y <= 0 || ball.Y >= FieldHeight {
		throwY := math.Max(1, math.Min(FieldHeight-1, ball.Y))
		setBallEvent(matchID, BallEventThrowIn, ball.X, throwY, 0)
		addLiveCommentary(matchID, match.Minute, "Throw-in", EventCommentary, nil)
	}
}

func handleKickoffBallEvent(matchID int, ball *BallPosition) {
	// Wait 3 seconds then start play
	if time.Since(ball.EventStarted) > 3*time.Second {
		// Find center midfielder to take kickoff
		match := matches[matchID]
		if match != nil {
			teamPlayers := getPlayersFromTeam(match.HomeTeam.ID)
			for _, player := range teamPlayers {
				if player.Position == PosCM {
					ball.PossessorID = player.ID
					ball.EventType = BallEventPlay
					break
				}
			}
		}
	}
}

func handleFreekickBallEvent(matchID int, ball *BallPosition) {
	// Wait 2 seconds then take free kick
	if time.Since(ball.EventStarted) > 2*time.Second {
		nearestPlayer := findNearestPlayerToBall(matchID, ball)
		if nearestPlayer != nil {
			ball.PossessorID = nearestPlayer.ID
			ball.EventType = BallEventPlay

			// Simulate free kick
			ball.Direction = rand.Float64() * 2 * math.Pi
			ball.Speed = 6.0 + rand.Float64()*8.0
		}
	}
}

func handleCornerBallEvent(matchID int, ball *BallPosition) {
	// Wait 3 seconds then take corner
	if time.Since(ball.EventStarted) > 3*time.Second {
		// Find winger or midfielder to take corner
		match := matches[matchID]
		if match != nil {
			isHomeCorner := ball.X > FieldWidth/2
			var teamID int
			if isHomeCorner {
				teamID = match.HomeTeam.ID
			} else {
				teamID = match.AwayTeam.ID
			}

			teamPlayers := getPlayersFromTeam(teamID)
			for _, player := range teamPlayers {
				if player.Position == PosLW || player.Position == PosRW || player.Position == PosCM {
					ball.PossessorID = player.ID
					ball.EventType = BallEventPlay

					// Aim toward goal area
					goalY := FieldHeight / 2
					ball.Direction = math.Atan2(goalY-ball.Y, (FieldWidth/2)-ball.X)
					ball.Speed = 8.0 + rand.Float64()*4.0
					break
				}
			}
		}
	}
}

func handleThrowInBallEvent(matchID int, ball *BallPosition) {
	// Wait 2 seconds then take throw-in
	if time.Since(ball.EventStarted) > 2*time.Second {
		nearestPlayer := findNearestPlayerToBall(matchID, ball)
		if nearestPlayer != nil {
			ball.PossessorID = nearestPlayer.ID
			ball.EventType = BallEventPlay
		}
	}
}

func handlePenaltyBallEvent(matchID int, ball *BallPosition) {
	// Wait 5 seconds then take penalty
	if time.Since(ball.EventStarted) > 5*time.Second {
		// Find striker to take penalty
		match := matches[matchID]
		if match != nil {
			isHomePenalty := ball.X > FieldWidth/2
			var teamID int
			if isHomePenalty {
				teamID = match.HomeTeam.ID
			} else {
				teamID = match.AwayTeam.ID
			}

			teamPlayers := getPlayersFromTeam(teamID)
			for _, player := range teamPlayers {
				if player.Position == PosST {
					ball.PossessorID = player.ID
					ball.EventType = BallEventPlay

					// High chance of goal on penalty
					if rand.Float64() < 0.8 {
						// Simulate goal
						handleGoalEvent(matchID, match)
					} else {
						// Miss - ball goes to keeper
						ball.Direction = math.Atan2((FieldHeight/2)-ball.Y, 0-ball.X)
						ball.Speed = 10.0
					}
					break
				}
			}
		}
	}
}

func handleGoalkickBallEvent(matchID int, ball *BallPosition) {
	// Wait 2 seconds then take goal kick
	if time.Since(ball.EventStarted) > 2*time.Second {
		match := matches[matchID]
		if match != nil {
			// Find goalkeeper
			isHomeGoalkick := ball.X < FieldWidth/2
			var teamID int
			if isHomeGoalkick {
				teamID = match.HomeTeam.ID
			} else {
				teamID = match.AwayTeam.ID
			}

			teamPlayers := getPlayersFromTeam(teamID)
			for _, player := range teamPlayers {
				if player.Position == PosGK {
					ball.PossessorID = player.ID
					ball.EventType = BallEventPlay

					// Long kick upfield
					ball.Direction = math.Atan2(0, FieldWidth-ball.X)
					ball.Speed = 12.0 + rand.Float64()*8.0
					break
				}
			}
		}
	}
}

func getFormationPositions(formation string, isHome bool) []FormationPosition {
	var positions []FormationPosition

	switch formation {
	case Formation442:
		positions = get442Formation(isHome)
	case Formation433:
		positions = get433Formation(isHome)
	case Formation352:
		positions = get352Formation(isHome)
	case Formation4231:
		positions = get4231Formation(isHome)
	case Formation532:
		positions = get532Formation(isHome)
	default:
		positions = get442Formation(isHome)
	}

	return positions
}

func get442Formation(isHome bool) []FormationPosition {
	baseX := 20.0
	if !isHome {
		baseX = 80.0
	}

	return []FormationPosition{
		{baseX - 15, FieldHeight / 2},    // GK
		{baseX, FieldHeight * 0.2},       // CB
		{baseX, FieldHeight * 0.8},       // CB
		{baseX + 5, FieldHeight * 0.1},   // LB
		{baseX + 5, FieldHeight * 0.9},   // RB
		{baseX + 20, FieldHeight * 0.25}, // CM
		{baseX + 20, FieldHeight * 0.45}, // CM
		{baseX + 20, FieldHeight * 0.55}, // CM
		{baseX + 20, FieldHeight * 0.75}, // CM
		{baseX + 35, FieldHeight * 0.35}, // ST
		{baseX + 35, FieldHeight * 0.65}, // ST
	}
}

func get433Formation(isHome bool) []FormationPosition {
	baseX := 20.0
	if !isHome {
		baseX = 80.0
	}

	return []FormationPosition{
		{baseX - 15, FieldHeight / 2},   // GK
		{baseX, FieldHeight * 0.2},      // CB
		{baseX, FieldHeight * 0.5},      // CB
		{baseX, FieldHeight * 0.8},      // CB
		{baseX + 5, FieldHeight * 0.1},  // LB
		{baseX + 5, FieldHeight * 0.9},  // RB
		{baseX + 20, FieldHeight * 0.3}, // CM
		{baseX + 20, FieldHeight * 0.5}, // CM
		{baseX + 20, FieldHeight * 0.7}, // CM
		{baseX + 35, FieldHeight * 0.2}, // LW
		{baseX + 35, FieldHeight * 0.5}, // ST
	}
}

func get352Formation(isHome bool) []FormationPosition {
	baseX := 20.0
	if !isHome {
		baseX = 80.0
	}

	return []FormationPosition{
		{baseX - 15, FieldHeight / 2},   // GK
		{baseX, FieldHeight * 0.25},     // CB
		{baseX, FieldHeight * 0.5},      // CB
		{baseX, FieldHeight * 0.75},     // CB
		{baseX + 15, FieldHeight * 0.1}, // LWB
		{baseX + 15, FieldHeight * 0.3}, // CM
		{baseX + 15, FieldHeight * 0.5}, // CM
		{baseX + 15, FieldHeight * 0.7}, // CM
		{baseX + 15, FieldHeight * 0.9}, // RWB
		{baseX + 35, FieldHeight * 0.4}, // ST
		{baseX + 35, FieldHeight * 0.6}, // ST
	}
}

func get4231Formation(isHome bool) []FormationPosition {
	baseX := 20.0
	if !isHome {
		baseX = 80.0
	}

	return []FormationPosition{
		{baseX - 15, FieldHeight / 2},   // GK
		{baseX, FieldHeight * 0.2},      // CB
		{baseX, FieldHeight * 0.8},      // CB
		{baseX + 5, FieldHeight * 0.1},  // LB
		{baseX + 5, FieldHeight * 0.9},  // RB
		{baseX + 20, FieldHeight * 0.4}, // CDM
		{baseX + 20, FieldHeight * 0.6}, // CDM
		{baseX + 30, FieldHeight * 0.2}, // LW
		{baseX + 30, FieldHeight * 0.5}, // CAM
		{baseX + 30, FieldHeight * 0.8}, // RW
		{baseX + 40, FieldHeight * 0.5}, // ST
	}
}

func get532Formation(isHome bool) []FormationPosition {
	baseX := 20.0
	if !isHome {
		baseX = 80.0
	}

	return []FormationPosition{
		{baseX - 15, FieldHeight / 2},   // GK
		{baseX, FieldHeight * 0.15},     // CB
		{baseX, FieldHeight * 0.35},     // CB
		{baseX, FieldHeight * 0.5},      // CB
		{baseX, FieldHeight * 0.65},     // CB
		{baseX, FieldHeight * 0.85},     // CB
		{baseX + 20, FieldHeight * 0.3}, // CM
		{baseX + 20, FieldHeight * 0.5}, // CM
		{baseX + 20, FieldHeight * 0.7}, // CM
		{baseX + 35, FieldHeight * 0.4}, // ST
		{baseX + 35, FieldHeight * 0.6}, // ST
	}
}

func main() {
	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://0.0.0.0:%s", port)
	}

	// Create router
	router := mux.NewRouter()

	// Apply CORS
	router.Use(corsMiddleware)

	// Apply application middleware
	router.Use(applicationMiddleware)

	// Home page route
	router.HandleFunc("/", serveHomepage).Methods("GET")

	// Fixtures page route
	router.HandleFunc("/fixtures", serveFixturesPage).Methods("GET")

	// Tables route
	router.HandleFunc("/tables", getTableData).Methods("GET")

	// Downloadable API schema file
	router.PathPrefix("/api-schema.txt").Handler(http.StripPrefix("/", http.FileServer(http.Dir("."))))

	// API routes - RESTful structure
	apiRouter := router.PathPrefix("/api/v1").Subrouter()

	// System endpoints
	apiRouter.HandleFunc("/health", healthCheck).Methods("GET")
	apiRouter.HandleFunc("/stats", getGlobalStats).Methods("GET")
	apiRouter.HandleFunc("/search", searchAPI).Methods("GET")

	// Match endpoints
	apiRouter.HandleFunc("/matches", getAllMatches).Methods("GET")
	apiRouter.HandleFunc("/matches/{id:[0-9]+}", getMatch).Methods("GET")
	apiRouter.HandleFunc("/matches/{id:[0-9]+}/stats", getMatchStats).Methods("GET")
	apiRouter.HandleFunc("/matches/{id:[0-9]+}/commentary", getMatchCommentary).Methods("GET")
	apiRouter.HandleFunc("/matches/{id:[0-9]+}/players", getMatchLocations).Methods("GET")
	apiRouter.HandleFunc("/matches/{id:[0-9]+}/momentum", getMatchMomentum).Methods("GET")
	apiRouter.HandleFunc("/matches/{id:[0-9]+}/probabilities", getMatchProbabilities).Methods("GET")
	apiRouter.HandleFunc("/matches/{id:[0-9]+}/availability", getMatchAvailability).Methods("GET")

	// Player endpoints
	apiRouter.HandleFunc("/players", getAllPlayers).Methods("GET")
	apiRouter.HandleFunc("/players/{id:[0-9]+}", getPlayer).Methods("GET")

	// Team endpoints
	apiRouter.HandleFunc("/teams", getAllTeams).Methods("GET")
	apiRouter.HandleFunc("/teams/{id:[0-9]+}", getTeam).Methods("GET")
	apiRouter.HandleFunc("/teams/{id:[0-9]+}/form", getTeamForm).Methods("GET")

	// League endpoints
	apiRouter.HandleFunc("/leagues/{league}/table", getLeagueTable).Methods("GET")
	apiRouter.HandleFunc("/leagues/{league}/form", getLeagueForm).Methods("GET")
	apiRouter.HandleFunc("/leagues/{league}/schedule", getSeasonSchedule).Methods("GET")

	// Season endpoints
	apiRouter.HandleFunc("/seasons/current", getSeasonStats).Methods("GET")
	apiRouter.HandleFunc("/seasons/history", getSeasonHistory).Methods("GET")
	apiRouter.HandleFunc("/seasons/current/matchdays/{matchday:[0-9]+}", getMatchdaySchedule).Methods("GET")

	// Fixture endpoints - view all season fixtures upfront
	apiRouter.HandleFunc("/fixtures", getAllFixtures).Methods("GET")
	apiRouter.HandleFunc("/fixtures/{league}", getLeagueFixtures).Methods("GET")

	// Print startup information
	fmt.Printf("🚀 MatchPulse API v%s starting on port %s\n", version, port)
	fmt.Printf("📚 API Documentation: %s/\n", baseURL)
	fmt.Printf("🏥 Health Check: %s/api/v1/health\n", baseURL)
	fmt.Printf("⚽ Live Matches: %s/api/v1/matches\n", baseURL)
	fmt.Printf("📊 Match Details: %s/api/v1/matches/1\n", baseURL)
	fmt.Printf("⚡ Match Momentum: %s/api/v1/matches/1/momentum\n", baseURL)
	fmt.Printf("🎲 Match Probabilities: %s/api/v1/matches/1/probabilities\n", baseURL)
	fmt.Printf("👥 Player Availability: %s/api/v1/matches/1/availability\n", baseURL)
	fmt.Printf("🏆 Season History: %s/api/v1/seasons/history\n", baseURL)
	fmt.Printf("📈 Current Season: %s/api/v1/seasons/current\n", baseURL)
	fmt.Printf("🏅 League Table: %s/api/v1/leagues/Premier%%20League/table\n", baseURL)
	fmt.Printf("📅 All Fixtures: %s/api/v1/fixtures\n", baseURL)
	fmt.Printf("📅 League Fixtures: %s/api/v1/fixtures/Premier%%20League\n", baseURL)

	// Start server
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, router))
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
	var teamName string
	var teamID int

	if isHomeCorner {
		teamName = match.HomeTeam.Name
		teamID = match.HomeTeam.ID
		matchStats[matchID].HomeCorners++
	} else {
		teamName = match.AwayTeam.Name
		teamID = match.AwayTeam.ID
		matchStats[matchID].AwayCorners++
	}

	// Update momentum - corners create pressure
	updateMatchMomentum(matchID, match, "corner", teamID)

	// Recalculate probabilities as corners can lead to goals
	recalculateMatchProbabilities(matchID, match)

	log.Printf("⚽ Corner kick for %s in match %d", teamName, matchID)
	addLiveCommentary(matchID, match.Minute,
		fmt.Sprintf("Corner kick for %s", teamName),
		EventCorner, nil)
}

func handleFoulEvent(matchID int, match *Match) {
	ball := ballPositions[matchID]
	if ball == nil {
		return
	}

	// Find players involved in the action (near the ball)
	playersNearBall := getPlayersNearAction(matchID, ball, 8.0)
	if len(playersNearBall) < 2 {
		return // Need at least 2 players for a foul
	}

	// Determine foul context
	foulContext := determineFoulContext(ball, match)

	// Select fouler based on context and proximity
	fouler := selectFouler(playersNearBall, ball, foulContext)
	if fouler == nil {
		return
	}

	// Determine foul severity based on context
	severity := determineFoulSeverity(fouler, ball, foulContext)

	// Apply foul consequences
	applyFoulConsequences(matchID, match, fouler, ball, severity, foulContext)
}

func getPlayersNearAction(matchID int, ball *BallPosition, radius float64) []*Player {
	var nearPlayers []*Player

	if locations, exists := playerLocations[matchID]; exists {
		for playerID, location := range locations {
			if player, exists := players[playerID]; exists {
				distance := math.Sqrt(math.Pow(location.X-ball.X, 2) + math.Pow(location.Y-ball.Y, 2))
				if distance <= radius {
					nearPlayers = append(nearPlayers, player)
				}
			}
		}
	}

	return nearPlayers
}

type FoulContext struct {
	IsInPenaltyArea   bool
	IsNearGoal        bool
	IsCounterAttack   bool
	IsDangerousPlay   bool
	BallPossessorTeam int
}

func determineFoulContext(ball *BallPosition, match *Match) FoulContext {
	context := FoulContext{}

	// Check if in penalty area
	if (ball.X <= 18 && ball.Y >= 20 && ball.Y <= 44) ||
		(ball.X >= FieldWidth-18 && ball.Y >= 20 && ball.Y <= 44) {
		context.IsInPenaltyArea = true
	}

	// Check if near goal
	distanceToGoal := math.Min(
		math.Sqrt(math.Pow(ball.X, 2)+math.Pow(ball.Y-FieldHeight/2, 2)),
		math.Sqrt(math.Pow(ball.X-FieldWidth, 2)+math.Pow(ball.Y-FieldHeight/2, 2)))
	context.IsNearGoal = distanceToGoal < 25

	// Check for dangerous play (high speed, attacking third)
	context.IsDangerousPlay = ball.Speed > 8.0 && (ball.X < 30 || ball.X > 70)

	// Determine ball possessor team
	if ball.PossessorID > 0 {
		if player, exists := players[ball.PossessorID]; exists {
			context.BallPossessorTeam = player.TeamID
		}
	}

	return context
}

func selectFouler(candidates []*Player, ball *BallPosition, context FoulContext) *Player {
	if len(candidates) == 0 {
		return nil
	}

	// Weight players by foul likelihood
	weights := make([]float64, len(candidates))

	for i, player := range candidates {
		weight := 1.0

		// Defending players more likely to foul
		if context.BallPossessorTeam != 0 && player.TeamID != context.BallPossessorTeam {
			weight *= 2.0
		}

		// Position-based likelihood
		switch player.Position {
		case PosCB, PosCDM:
			weight *= 1.5 // Defenders more likely to foul
		case PosLB, PosRB:
			weight *= 1.3
		case PosCM:
			weight *= 1.0
		case PosCAM, PosLW, PosRW:
			weight *= 0.8
		case PosST:
			weight *= 0.6
		case PosGK:
			if context.IsInPenaltyArea {
				weight *= 1.8 // GK fouls in penalty area
			} else {
				weight *= 0.1 // GK rarely fouls outside area
			}
		}

		// Context-based adjustments
		if context.IsNearGoal {
			weight *= 1.4 // More fouls near goal
		}
		if context.IsDangerousPlay {
			weight *= 1.6 // More fouls in dangerous situations
		}

		weights[i] = weight
	}

	// Select based on weights
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	r := rand.Float64() * totalWeight
	cumulative := 0.0

	for i, weight := range weights {
		cumulative += weight
		if r <= cumulative {
			return candidates[i]
		}
	}

	return candidates[len(candidates)-1]
}

func determineFoulSeverity(fouler *Player, ball *BallPosition, context FoulContext) string {
	baseSeverity := "yellow"

	// Penalty area fouls more severe
	if context.IsInPenaltyArea {
		if rand.Float64() < 0.3 {
			baseSeverity = "red"
		}
	}

	// Goalkeeper handling outside penalty area
	if fouler.Position == PosGK && !context.IsInPenaltyArea && ball.Speed > 5.0 {
		if rand.Float64() < 0.6 {
			baseSeverity = "red"
		}
	}

	// Last man fouls
	if context.IsNearGoal && fouler.Position == PosCB {
		if rand.Float64() < 0.4 {
			baseSeverity = "red"
		}
	}

	// Dangerous play
	if context.IsDangerousPlay && rand.Float64() < 0.25 {
		baseSeverity = "red"
	}

	return baseSeverity
}

func applyFoulConsequences(matchID int, match *Match, fouler *Player, ball *BallPosition, severity string, context FoulContext) {
	// Update statistics
	if fouler.TeamID == match.HomeTeam.ID {
		matchStats[matchID].HomeFouls++
	} else {
		matchStats[matchID].AwayFouls++
	}

	// Apply card
	if severity == "yellow" {
		fouler.YellowCards++
		fouler.SeasonStats.YellowCardsThisSeason++
		fouler.CurrentRating -= 0.3

		if fouler.TeamID == match.HomeTeam.ID {
			matchStats[matchID].HomeYellowCards++
		} else {
			matchStats[matchID].AwayYellowCards++
		}

		logInfo("🟨 Yellow card for %s (foul)", fouler.Name)
		addLiveCommentary(matchID, match.Minute,
			fmt.Sprintf("Yellow card! %s commits a foul", fouler.Name),
			EventCard, fouler)

	} else if severity == "red" {
		fouler.RedCards++
		fouler.SeasonStats.RedCardsThisSeason++
		fouler.CurrentRating -= 1.5

		if fouler.TeamID == match.HomeTeam.ID {
			matchStats[matchID].HomeRedCards++
		} else {
			matchStats[matchID].AwayRedCards++
		}

		logInfo("🟥 Red card for %s (serious foul)", fouler.Name)
		addLiveCommentary(matchID, match.Minute,
			fmt.Sprintf("RED CARD! %s sent off for serious foul play!", fouler.Name),
			EventCard, fouler)
	}

	// Determine restart type
	if context.IsInPenaltyArea && fouler.TeamID != context.BallPossessorTeam {
		// Penalty
		penaltyX := 11.0
		if ball.X < FieldWidth/2 {
			penaltyX = FieldWidth - 11.0
		}
		setBallEvent(matchID, BallEventPenalty, penaltyX, FieldHeight/2, 0)
		addLiveCommentary(matchID, match.Minute, "PENALTY!", EventPenalty, nil)
	} else {
		// Free kick
		setBallEvent(matchID, BallEventFreekick, ball.X, ball.Y, 0)
		addLiveCommentary(matchID, match.Minute, "Free kick awarded", EventFreekick, nil)
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
	if match == nil || match.Status != StatusLive {
		return
	}

	if playerLocations[matchID] == nil {
		playerLocations[matchID] = make(map[int]*PlayerLocation)
	}

	// Initialize ball if not exists
	if ballPositions[matchID] == nil {
		ballPositions[matchID] = &BallPosition{
			X:            FieldWidth / 2,
			Y:            FieldHeight / 2,
			EventType:    BallEventKickoff,
			EventStarted: time.Now(),
			Timestamp:    time.Now(),
		}
	}

	ball := ballPositions[matchID]

	// Get match tactics
	tactics := getMatchTactics(matchID)

	// Determine possession (home team has ball if possessor is from home team)
	homePossession := false
	if ball.PossessorID > 0 {
		if player, exists := players[ball.PossessorID]; exists {
			homePossession = player.TeamID == match.HomeTeam.ID
		}
	}

	// Get starting 11 from each team
	homePlayers := getPlayersFromTeam(match.HomeTeam.ID)[:11]
	awayPlayers := getPlayersFromTeam(match.AwayTeam.ID)[:11]

	// Update positions based on tactics and ball position
	updateTeamWithTactics(matchID, homePlayers, true, homePossession, tactics.HomeOffensive, tactics.HomeDefensive, ball, match)
	updateTeamWithTactics(matchID, awayPlayers, false, !homePossession, tactics.AwayOffensive, tactics.AwayDefensive, ball, match)

	// Update ball position
	updateBallPhysics(matchID, ball)
}

func updateTeamWithTactics(matchID int, teamPlayers []*Player, isHome, hasPossession bool,
	offensiveTactic, defensiveTactic string, ball *BallPosition, match *Match) {

	formation := match.HomeFormation
	if !isHome {
		formation = match.AwayFormation
	}

	// Get formation positions
	positions := getFormationPositions(formation, isHome)

	for i, player := range teamPlayers {
		if player == nil || i >= len(positions) {
			continue
		}

		basePos := positions[i]
		var x, y float64

		if hasPossession {
			// Offensive positioning based on tactic
			x, y = calculateOffensivePosition(player, basePos, offensiveTactic, ball, match.Minute)
		} else {
			// Defensive positioning based on tactic
			x, y = calculateDefensivePosition(player, basePos, defensiveTactic, ball, match.Minute)
		}

		// Add slight natural movement
		x += math.Sin(float64(time.Now().Unix())+float64(player.ID)) * 1.5
		y += math.Cos(float64(time.Now().Unix())+float64(player.ID)*1.5) * 1.5

		// Keep within bounds
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

func calculateOffensivePosition(player *Player, basePos FormationPosition, tactic string, ball *BallPosition, minute int) (float64, float64) {
	x, y := basePos.X, basePos.Y

	switch tactic {
	case TacticTikiTaka:
		// Players form triangles around the ball
		if distance(x, y, ball.X, ball.Y) < 20 {
			angle := math.Atan2(y-ball.Y, x-ball.X)
			x = ball.X + math.Cos(angle)*12
			y = ball.Y + math.Sin(angle)*12
		}
		// Slight forward push
		x += 5

	case TacticCounterAttack:
		// Quick forward movement
		if player.Position == PosST || player.Position == PosLW || player.Position == PosRW {
			x += 15
		}

	case TacticWingPlay:
		// Wingers stay wide
		if player.Position == PosLW {
			y = math.Min(y+10, FieldHeight-5)
		} else if player.Position == PosRW {
			y = math.Max(y-10, 5)
		}
	}

	return x, y
}

func calculateDefensivePosition(player *Player, basePos FormationPosition, tactic string, ball *BallPosition, minute int) (float64, float64) {
	x, y := basePos.X, basePos.Y

	switch tactic {
	case TacticCompactDefense:
		// Move towards center and back
		y = y*0.7 + (FieldHeight/2)*0.3
		x -= 10

	case TacticManMarking:
		// Move closer to nearest opponent
		// This would need opponent tracking
		x = x*0.8 + ball.X*0.2
		y = y*0.8 + ball.Y*0.2

	case TacticLowBlock:
		// Deep defensive line
		if player.Position != PosGK {
			x = math.Min(x, 30)
		}
	}

	return x, y
}

// Helper function for distance calculation
func distance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
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

	// Performance during match (accumulated from events)
	performanceBonus := player.CurrentRating - 6.0
	baseRating += math.Max(-2.5, math.Min(4.0, performanceBonus))

	// Position-specific bonuses
	switch player.Position {
	case PosGK:
		// GK gets bonus for saves, penalty for goals conceded
		oppScore := match.AwayScore
		if !isHome {
			oppScore = match.HomeScore
		}
		if oppScore == 0 {
			baseRating += 1.0 // Clean sheet bonus
		} else if oppScore >= 3 {
			baseRating -= 0.5 // Heavy defeat penalty
		}

	case PosST, PosLW, PosRW:
		// Attackers get bonus for goals and shots
		if player.SeasonStats.GoalsThisSeason > 0 && rand.Float64() < 0.3 {
			baseRating += 0.8 // Goal bonus
		}

	case PosCB, PosLB, PosRB:
		// Defenders get clean sheet bonus
		oppScore := match.AwayScore
		if !isHome {
			oppScore = match.HomeScore
		}
		if oppScore == 0 {
			baseRating += 0.7
		} else if oppScore >= 2 {
			baseRating -= 0.4
		}

	case PosCM, PosCAM, PosCDM:
		// Midfielders get bonus for assists and all-round play
		if player.SeasonStats.AssistsThisSeason > 0 && rand.Float64() < 0.2 {
			baseRating += 0.5 // Assist bonus
		}
	}

	// Team result impact
	teamScore := match.HomeScore
	oppScore := match.AwayScore
	if !isHome {
		teamScore = match.AwayScore
		oppScore = match.HomeScore
	}

	if teamScore > oppScore {
		baseRating += 0.4 // Win bonus
	} else if teamScore < oppScore {
		baseRating -= 0.4 // Loss penalty
	} else {
		baseRating += 0.1 // Small draw bonus
	}

	// Card penalties
	if player.YellowCards > 0 {
		baseRating -= 0.3
	}
	if player.RedCards > 0 {
		baseRating -= 1.5
	}

	// Reset for next match
	player.CurrentRating = 6.0

	return math.Max(3.0, math.Min(10.0, baseRating))
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
	leagues := []string{LeaguePremier, LeagueCommunityLeague}

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

	// Generate proper round-robin tournament schedule
	// Each team plays every other team twice (home and away)
	totalRounds := len(leagueTeams) - 1

	// Generate first leg (teams play each other once)
	for round := 0; round < totalRounds; round++ {
		roundMatches := generateRoundMatches(leagueTeams, round)
		for _, match := range roundMatches {
			schedule := &SeasonSchedule{
				Matchday:    matchday,
				League:      league,
				HomeTeam:    match.Home,
				AwayTeam:    match.Away,
				IsPlayed:    false,
				ScheduledAt: time.Now().Add(time.Duration(matchday) * 24 * time.Hour),
			}
			schedules = append(schedules, schedule)
		}
		matchday++
	}

	// Generate return leg (teams play each other again with reversed home/away)
	for round := 0; round < totalRounds; round++ {
		roundMatches := generateRoundMatches(leagueTeams, round)
		for _, match := range roundMatches {
			// Reverse home and away for return leg
			schedule := &SeasonSchedule{
				Matchday:    matchday,
				League:      league,
				HomeTeam:    match.Away, // Swapped
				AwayTeam:    match.Home, // Swapped
				IsPlayed:    false,
				ScheduledAt: time.Now().Add(time.Duration(matchday) * 24 * time.Hour),
			}
			schedules = append(schedules, schedule)
		}
		matchday++
	}

	seasonSchedules[league] = schedules
	log.Printf("📅 Generated complete season schedule for %s: %d matches across %d matchdays (%d teams)",
		league, len(schedules), matchday-1, len(leagueTeams))
}

// Helper struct for round generation
type RoundMatch struct {
	Home *TeamInfo
	Away *TeamInfo
}

// Generate matches for a specific round using round-robin algorithm
func generateRoundMatches(teams []*TeamInfo, round int) []RoundMatch {
	var matches []RoundMatch
	n := len(teams)

	if n%2 != 0 {
		log.Printf("⚠️ Odd number of teams (%d), scheduling may be unbalanced", n)
		return matches
	}

	// Use round-robin tournament algorithm
	// Fix one team and rotate others
	for i := 0; i < n/2; i++ {
		team1Idx := i
		team2Idx := (n - 1 - i + round) % (n - 1)

		// Ensure we don't go out of bounds
		if team2Idx >= team1Idx {
			team2Idx++
		}

		if team1Idx < len(teams) && team2Idx < len(teams) && team1Idx != team2Idx {
			matches = append(matches, RoundMatch{
				Home: teams[team1Idx],
				Away: teams[team2Idx],
			})
		}
	}

	return matches
}

// Log a summary of all generated fixtures
func logFixtureSummary() {
	totalFixtures := 0
	for league, schedules := range seasonSchedules {
		totalFixtures += len(schedules)
		log.Printf("📅 %s: %d fixtures generated (Matchdays 1-%d)",
			league, len(schedules), len(schedules)/(len(getTeamsByLeague(league))/2))
	}
	log.Printf("🏆 Total season fixtures: %d matches across all leagues", totalFixtures)
	log.Printf("🔍 View all fixtures: GET /api/v1/fixtures")
	log.Printf("🔍 View league fixtures: GET /api/v1/fixtures/{league}")
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
	// This function is called within schedulingMutex, so no additional locking needed

	// Get the current minimum matches played across all teams for balanced scheduling
	minMatchesPlayed := getMinimumMatchesPlayed()
	maxMatchesPlayed := getMaximumMatchesPlayed()

	// If difference is too large, try to find any valid match
	if maxMatchesPlayed-minMatchesPlayed > 2 {
		// First try to find a match between least played teams
		if match := getMatchForLeastPlayedTeams(); match != nil {
			return match
		}

		// If that fails, fall back to any unplayed match
		for _, league := range []string{LeaguePremier, LeagueCommunityLeague} {
			if schedules, exists := seasonSchedules[league]; exists {
				for _, schedule := range schedules {
					if !schedule.IsPlayed {
						logInfo("🎯 Selected fallback match: %s vs %s (Home: %d, Away: %d played)",
							schedule.HomeTeam.ShortName, schedule.AwayTeam.ShortName,
							getTeamMatchesPlayed(schedule.HomeTeam.ID),
							getTeamMatchesPlayed(schedule.AwayTeam.ID))
						return schedule
					}
				}
			}
		}
		return nil
	}

	// Otherwise, get next chronological match that's fair
	for _, league := range []string{LeaguePremier, LeagueCommunityLeague} {
		if schedules, exists := seasonSchedules[league]; exists {
			// Sort by matchday to ensure chronological order
			var sortedSchedules []*SeasonSchedule
			for _, schedule := range schedules {
				if !schedule.IsPlayed {
					sortedSchedules = append(sortedSchedules, schedule)
				}
			}

			// Sort by matchday
			sort.Slice(sortedSchedules, func(i, j int) bool {
				return sortedSchedules[i].Matchday < sortedSchedules[j].Matchday
			})

			for _, schedule := range sortedSchedules {
				homeMatchesPlayed := getTeamMatchesPlayed(schedule.HomeTeam.ID)
				awayMatchesPlayed := getTeamMatchesPlayed(schedule.AwayTeam.ID)

				// More flexible criteria: allow matches if teams haven't played too many more than minimum
				if homeMatchesPlayed <= minMatchesPlayed+2 && awayMatchesPlayed <= minMatchesPlayed+2 {
					logInfo("🎯 Selected balanced match: %s vs %s (Home: %d, Away: %d played, Min: %d)",
						schedule.HomeTeam.ShortName, schedule.AwayTeam.ShortName,
						homeMatchesPlayed, awayMatchesPlayed, minMatchesPlayed)
					return schedule
				}
			}
		}
	}

	return nil
}

func getMatchForLeastPlayedTeams() *SeasonSchedule {
	minMatchesPlayed := getMinimumMatchesPlayed()

	for _, league := range []string{LeaguePremier, LeagueCommunityLeague} {
		if schedules, exists := seasonSchedules[league]; exists {
			for _, schedule := range schedules {
				if !schedule.IsPlayed {
					homeMatchesPlayed := getTeamMatchesPlayed(schedule.HomeTeam.ID)
					awayMatchesPlayed := getTeamMatchesPlayed(schedule.AwayTeam.ID)

					// More flexible: allow teams that have played minimum or minimum+1 matches
					if (homeMatchesPlayed == minMatchesPlayed || homeMatchesPlayed == minMatchesPlayed+1) &&
						(awayMatchesPlayed == minMatchesPlayed || awayMatchesPlayed == minMatchesPlayed+1) {
						logInfo("🎯 Selected catch-up match: %s vs %s (Home: %d, Away: %d played, Min: %d)",
							schedule.HomeTeam.ShortName, schedule.AwayTeam.ShortName,
							homeMatchesPlayed, awayMatchesPlayed, minMatchesPlayed)
						return schedule
					}
				}
			}
		}
	}

	return nil
}

func getMaximumMatchesPlayed() int {
	maxMatches := 0
	for _, team := range teams {
		matchesPlayed := getTeamMatchesPlayed(team.ID)
		if matchesPlayed > maxMatches {
			maxMatches = matchesPlayed
		}
	}
	return maxMatches
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
	for _, league := range []string{LeaguePremier, LeagueCommunityLeague} {
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
	// Base strength (0.4 to ensure minimum reasonable strength)
	baseStrength := 0.4

	// Form strength (0-15 points possible, scaled to 0.0-0.3)
	formStrength := float64(team.FormPoints) / 15.0 * 0.3

	// Calculate final strength (0.4 to 0.7)
	strength := baseStrength + formStrength

	// Ensure strength is within bounds
	if strength < 0.4 {
		strength = 0.4
	} else if strength > 0.7 {
		strength = 0.7
	}

	return strength
}

func updateTeamForm(team *TeamInfo, result string, isHome bool) {
	// Update form array (last 5 results)
	if len(team.Form) >= 5 {
		team.Form = team.Form[1:]
	}
	team.Form = append(team.Form, result)

	// Calculate form points
	team.FormPoints = 0
	for _, r := range team.Form {
		switch r {
		case "W":
			team.FormPoints += 3
		case "D":
			team.FormPoints += 1
		}
	}

	logInfo("Updated team form: %s - Form: %v, Points: %d",
		team.ShortName, team.Form, team.FormPoints)
}

// Calculate team strength based on player ratings and form
func calculateTeamStrength(team *TeamInfo) float64 {
	// Base strength (0.4 to ensure minimum reasonable strength)
	baseStrength := 0.4

	// Form strength (0-15 points possible, scaled to 0.0-0.3)
	formStrength := float64(team.FormPoints) / 15.0 * 0.3

	// Calculate final strength (0.4 to 0.7)
	strength := baseStrength + formStrength

	// Ensure strength is within bounds
	if strength < 0.4 {
		strength = 0.4
	} else if strength > 0.7 {
		strength = 0.7
	}

	return strength
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
					<option value="Community League"%s>Community League</option>
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
			map[bool]string{true: " selected", false: ""}[selectedLeague == "Community League"],
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
	start := (page - 1) * itemsPerPage
	end := start + itemsPerPage

	// Get all teams
	allTeams := make([]*TeamInfo, 0, len(teams))
	for _, team := range teams {
		allTeams = append(allTeams, team)
	}

	// Sort teams by name
	sort.Slice(allTeams, func(i, j int) bool {
		return allTeams[i].Name < allTeams[j].Name
	})

	// Calculate total pages
	totalPages := (len(allTeams) + itemsPerPage - 1) / itemsPerPage

	// Generate table
	html.WriteString("<table class='table table-striped'>")
	html.WriteString("<thead><tr>")
	html.WriteString("<th>ID</th>")
	html.WriteString("<th>Name</th>")
	html.WriteString("<th>League</th>")
	html.WriteString("<th>Form</th>")
	html.WriteString("<th>Form Points</th>")
	html.WriteString("</tr></thead><tbody>")

	for i := start; i < end && i < len(allTeams); i++ {
		team := allTeams[i]
		html.WriteString("<tr>")
		html.WriteString(fmt.Sprintf("<td>%d</td>", team.ID))
		html.WriteString(fmt.Sprintf("<td>%s</td>", team.Name))
		html.WriteString(fmt.Sprintf("<td>%s</td>", team.League))
		html.WriteString(fmt.Sprintf("<td>%v</td>", team.Form))
		html.WriteString(fmt.Sprintf("<td>%d</td>", team.FormPoints))
		html.WriteString("</tr>")
	}

	html.WriteString("</tbody></table>")
	return html.String(), totalPages
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

func serveFixturesPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	mutex.RLock()
	templateData := struct {
		CurrentSeason    int
		CurrentMatchweek int
		LastUpdated      string
		Version          string
	}{
		CurrentSeason:    currentSeason,
		CurrentMatchweek: currentMatchweek,
		LastUpdated:      time.Now().Format("2006-01-02 15:04:05"),
		Version:          version,
	}
	mutex.RUnlock()

	const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>MatchPulse Fixtures - v{{.Version}}</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            box-sizing: border-box;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 12px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            min-height: calc(100vh - 40px);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 2px solid #f1f3f4;
        }
        .header h1 {
            font-size: 2.5rem;
            margin: 0 0 10px 0;
            color: #333;
        }
        .header p {
            color: #6c757d;
            font-size: 1.1rem;
            margin: 0;
        }
        .fixtures-container {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 25px;
            margin-top: 30px;
        }
        .fixture-card {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.05);
        }
        .fixture-card h2 {
            margin: 0 0 20px 0;
            color: #495057;
            font-size: 1.4rem;
            padding-bottom: 10px;
            border-bottom: 2px solid #dee2e6;
        }
        .match-list {
            list-style: none;
            padding: 0;
            margin: 0;
        }
        .match-item {
            padding: 15px;
            border-bottom: 1px solid #dee2e6;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .match-item:last-child {
            border-bottom: none;
        }
        .team {
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .team img {
            width: 30px;
            height: 30px;
            border-radius: 50%;
        }
        .match-time {
            color: #6c757d;
            font-size: 0.9rem;
        }
        .match-status {
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 0.8rem;
            font-weight: bold;
        }
        .status-live {
            background: #28a745;
            color: white;
        }
        .status-finished {
            background: #6c757d;
            color: white;
        }
        .status-scheduled {
            background: #ffc107;
            color: #212529;
        }
        .back-link {
            display: inline-block;
            margin-top: 20px;
            color: #007bff;
            text-decoration: none;
            font-weight: 500;
        }
        .back-link:hover {
            color: #0056b3;
        }
        .league-selector {
            margin-bottom: 20px;
            text-align: center;
        }
        .league-selector select {
            padding: 8px 16px;
            border: 1px solid #dee2e6;
            border-radius: 4px;
            font-size: 1rem;
            margin-right: 10px;
        }
        .league-selector button {
            padding: 8px 16px;
            background: #007bff;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 1rem;
        }
        .league-selector button:hover {
            background: #0056b3;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📅 MatchPulse Fixtures</h1>
            <p>Season {{.CurrentSeason}} - Matchweek {{.CurrentMatchweek}}</p>
            <div class="league-selector">
                <select id="league-select" onchange="updateFixtures()">
                    <option value="Premier League">Premier League</option>
                    <option value="Community League">Community League</option>
                </select>
                <button onclick="updateFixtures()">View Fixtures</button>
            </div>
        </div>
        <div class="fixtures-container" id="fixtures-container">
            <!-- Fixtures will be loaded here via JavaScript -->
        </div>
        <a href="/" class="back-link">← Back to Home</a>
    </div>
    <script>
        function updateFixtures() {
            const league = document.getElementById('league-select').value;
            fetch('/api/v1/fixtures/' + encodeURIComponent(league))
                .then(response => response.json())
                .then(data => {
                    const container = document.getElementById('fixtures-container');
                    container.innerHTML = '';
                    
                    // Group fixtures by matchday
                    const matchdays = {};
                    data.fixtures.forEach(fixture => {
                        if (!matchdays[fixture.matchday]) {
                            matchdays[fixture.matchday] = [];
                        }
                        matchdays[fixture.matchday].push(fixture);
                    });

                    // Sort matchdays
                    const sortedMatchdays = Object.keys(matchdays).sort((a, b) => a - b);

                    // Create fixture cards for each matchday
                    sortedMatchdays.forEach(matchday => {
                        const fixtures = matchdays[matchday];
                        const card = document.createElement('div');
                        card.className = 'fixture-card';
                        
                        let html = '<h2>Matchday ' + matchday + '</h2>';
                        html += '<ul class="match-list">';
                        
                        fixtures.forEach(fixture => {
                            const statusClass = fixture.is_played ? 'status-finished' : 
                                             fixture.match_id ? 'status-live' : 'status-scheduled';
                            const statusText = fixture.is_played ? 'Finished' : 
                                             fixture.match_id ? 'Live' : 'Scheduled';
                            
                            html += "<li class=\"match-item\">" +
                                "<div class=\"team\">" +
                                "<img src=\"" + fixture.home_team.logo_url + "\" alt=\"" + fixture.home_team.name + "\">" +
                                "<span>" + fixture.home_team.name + "</span>" +
                                "</div>" +
                                "<div class=\"match-time\">" +
                                new Date(fixture.scheduled_at).toLocaleString() +
                                "</div>" +
                                "<div class=\"team\">" +
                                "<span>" + fixture.away_team.name + "</span>" +
                                "<img src=\"" + fixture.away_team.logo_url + "\" alt=\"" + fixture.away_team.name + "\">" +
                                "</div>" +
                                "<span class=\"match-status " + statusClass + "\">" + statusText + "</span>" +
                                "</li>";
                        });
                        
                        html += '</ul>';
                        card.innerHTML = html;
                        container.appendChild(card);
                    });
                })
                .catch(error => {
                    console.error('Error loading fixtures:', error);
                    document.getElementById('fixtures-container').innerHTML = 
                        '<div class="error">Error loading fixtures. Please try again later.</div>';
                });
        }

        // Load fixtures when page loads
        document.addEventListener('DOMContentLoaded', updateFixtures);
    </script>
</body>
</html>`

	tmpl, err := template.New("fixtures").Parse(htmlTemplate)
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
