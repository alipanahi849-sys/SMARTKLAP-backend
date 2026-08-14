package football

import "context"

const ProviderAPIFootball = "api-football"

// Team is a club as returned by the upstream football provider.
type Team struct {
	ProviderID string
	Name       string
	Code       string
	Country    string
	LogoURL    string
	VenueName  string
}

// Fixture is a match summary from the provider.
type Fixture struct {
	ProviderID       string
	Kickoff          string
	Status           string
	Elapsed          int
	Extra            int
	StadiumName      string
	LeagueProviderID string
	LeagueName       string
	LeagueCountry    string
	LeagueLogoURL    string
	Season           string
	HomeTeam         Team
	AwayTeam         Team
	HomeScore        *int
	AwayScore        *int
}

// StatPair is one labelled home/away statistic.
type StatPair struct {
	Label string
	Home  int
	Away  int
}

// Event is a timeline item (goal, card, substitution, …).
type Event struct {
	Elapsed    int
	Extra      int
	TeamID     string
	Type       string
	PlayerName string
	AssistName string
	Detail     string
}

// LineupPlayer is one player in a match squad.
type LineupPlayer struct {
	ProviderPlayerID string
	Name             string
	Number           int
	Position         string
	PhotoURL         string
	Starter          bool
}

// Lineup is one team's starting XI + bench for a fixture.
type Lineup struct {
	TeamID    string
	Formation string
	Players   []LineupPlayer
}

// PlayerProfile is a player card used on the Players screen.
type PlayerProfile struct {
	ProviderID     string
	Name           string
	Number         int
	Position       string
	Age            int
	PreferredFoot  string
	Nationality    string
	HeightCM       int
	WeightKG       int
	PhotoURL       string
	ClubName       string
	ClubProviderID string
	Radar          []RadarStat
}

// RadarStat is one spoke on the player radar chart.
type RadarStat struct {
	Label string
	Value int
}

// FixtureDetails is live stats, events, and lineups for one fixture.
type FixtureDetails struct {
	Fixture Fixture
	Stats   []StatPair
	Events  []Event
	Lineups []Lineup
}

// Provider fetches football data from an upstream API.
type Provider interface {
	Name() string
	Enabled() bool
	SearchTeams(ctx context.Context, query string) ([]Team, error)
	GetTeam(ctx context.Context, providerTeamID string) (*Team, error)
	ListFixtures(ctx context.Context, providerTeamID string, next, last int) ([]Fixture, error)
	ListLiveFixtures(ctx context.Context, providerTeamID string) ([]Fixture, error)
	GetFixtureDetails(ctx context.Context, providerMatchID string) (*FixtureDetails, error)
	GetPlayer(ctx context.Context, providerPlayerID, season string) (*PlayerProfile, error)
}
