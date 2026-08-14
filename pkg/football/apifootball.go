package football

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
)

const defaultAPIFootballBase = "https://v3.football.api-sports.io"

type apiFootballClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// NewAPIFootball builds the API-Sports football provider. An empty key disables it.
func NewAPIFootball(apiKey, baseURL string) Provider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultAPIFootballBase
	}
	return &apiFootballClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     strings.TrimSpace(apiKey),
	}
}

func (c *apiFootballClient) Name() string { return ProviderAPIFootball }

func (c *apiFootballClient) Enabled() bool { return c.apiKey != "" }

type apiEnvelope struct {
	Errors   json.RawMessage `json:"errors"`
	Response json.RawMessage `json:"response"`
}

type teamRow struct {
	Team struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Code    string `json:"code"`
		Country string `json:"country"`
		Logo    string `json:"logo"`
	} `json:"team"`
	Venue struct {
		Name string `json:"name"`
	} `json:"venue"`
}

type fixtureRow struct {
	Fixture struct {
		ID    int    `json:"id"`
		Date  string `json:"date"`
		Venue struct {
			Name string `json:"name"`
		} `json:"venue"`
		Status struct {
			Short   string `json:"short"`
			Elapsed *int   `json:"elapsed"`
			Extra   *int   `json:"extra"`
		} `json:"status"`
	} `json:"fixture"`
	League struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Country string `json:"country"`
		Logo    string `json:"logo"`
		Season  int    `json:"season"`
	} `json:"league"`
	Teams struct {
		Home teamInfo `json:"home"`
		Away teamInfo `json:"away"`
	} `json:"teams"`
	Goals struct {
		Home *int `json:"home"`
		Away *int `json:"away"`
	} `json:"goals"`
}

type teamInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Logo string `json:"logo"`
}

type statsRow struct {
	Team struct {
		ID int `json:"id"`
	} `json:"team"`
	Statistics []struct {
		Type  string `json:"type"`
		Value any    `json:"value"`
	} `json:"statistics"`
}

type eventRow struct {
	Time struct {
		Elapsed *int `json:"elapsed"`
		Extra   *int `json:"extra"`
	} `json:"time"`
	Team struct {
		ID int `json:"id"`
	} `json:"team"`
	Player struct {
		Name string `json:"name"`
	} `json:"player"`
	Assist struct {
		Name string `json:"name"`
	} `json:"assist"`
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

type lineupRow struct {
	Team struct {
		ID int `json:"id"`
	} `json:"team"`
	Formation string `json:"formation"`
	StartXI   []struct {
		Player lineupPlayer `json:"player"`
	} `json:"startXI"`
	Substitutes []struct {
		Player lineupPlayer `json:"player"`
	} `json:"substitutes"`
}

type lineupPlayer struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Number int    `json:"number"`
	Pos    string `json:"pos"`
	Photo  string `json:"photo"`
}

type playerRow struct {
	Player struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Age         int    `json:"age"`
		Nationality string `json:"nationality"`
		Height      string `json:"height"`
		Weight      string `json:"weight"`
		Photo       string `json:"photo"`
	} `json:"player"`
	Statistics []struct {
		Team struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
		Games struct {
			Number   *int    `json:"number"`
			Position string  `json:"position"`
			Rating   *string `json:"rating"`
		} `json:"games"`
		Shots struct {
			Total *int `json:"total"`
			On    *int `json:"on"`
		} `json:"shots"`
		Passes struct {
			Accuracy any  `json:"accuracy"`
			Key      *int `json:"key"`
		} `json:"passes"`
		Tackles struct {
			Total *int `json:"total"`
		} `json:"tackles"`
		Dribbles struct {
			Success *int `json:"success"`
		} `json:"dribbles"`
	} `json:"statistics"`
}

func (c *apiFootballClient) SearchTeams(ctx context.Context, query string) ([]Team, error) {
	if err := c.requireEnabled(); err != nil {
		return nil, err
	}
	q := strings.TrimSpace(query)
	if len(q) < 2 {
		return nil, errors.NewBadRequest("search query must be at least 2 characters", nil)
	}
	var rows []teamRow
	if err := c.get(ctx, "/teams", url.Values{"search": {q}}, &rows); err != nil {
		return nil, err
	}
	out := make([]Team, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapTeamRow(row))
	}
	return out, nil
}

func (c *apiFootballClient) GetTeam(ctx context.Context, providerTeamID string) (*Team, error) {
	if err := c.requireEnabled(); err != nil {
		return nil, err
	}
	var rows []teamRow
	if err := c.get(ctx, "/teams", url.Values{"id": {providerTeamID}}, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.NewNotFound("Team not found", nil)
	}
	team := mapTeamRow(rows[0])
	return &team, nil
}

func (c *apiFootballClient) ListFixtures(ctx context.Context, providerTeamID string, next, last int) ([]Fixture, error) {
	if err := c.requireEnabled(); err != nil {
		return nil, err
	}
	seen := map[int]struct{}{}
	var out []Fixture
	if next > 0 {
		var rows []fixtureRow
		if err := c.get(ctx, "/fixtures", url.Values{
			"team": {providerTeamID},
			"next": {strconv.Itoa(next)},
		}, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			if _, ok := seen[row.Fixture.ID]; ok {
				continue
			}
			seen[row.Fixture.ID] = struct{}{}
			out = append(out, mapFixtureRow(row))
		}
	}
	if last > 0 {
		var rows []fixtureRow
		if err := c.get(ctx, "/fixtures", url.Values{
			"team": {providerTeamID},
			"last": {strconv.Itoa(last)},
		}, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			if _, ok := seen[row.Fixture.ID]; ok {
				continue
			}
			seen[row.Fixture.ID] = struct{}{}
			out = append(out, mapFixtureRow(row))
		}
	}
	return out, nil
}

func (c *apiFootballClient) ListLiveFixtures(ctx context.Context, providerTeamID string) ([]Fixture, error) {
	if err := c.requireEnabled(); err != nil {
		return nil, err
	}
	var rows []fixtureRow
	if err := c.get(ctx, "/fixtures", url.Values{
		"team": {providerTeamID},
		"live": {"all"},
	}, &rows); err != nil {
		return nil, err
	}
	out := make([]Fixture, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapFixtureRow(row))
	}
	return out, nil
}

func (c *apiFootballClient) GetFixtureDetails(ctx context.Context, providerMatchID string) (*FixtureDetails, error) {
	if err := c.requireEnabled(); err != nil {
		return nil, err
	}
	var fixtures []fixtureRow
	if err := c.get(ctx, "/fixtures", url.Values{"id": {providerMatchID}}, &fixtures); err != nil {
		return nil, err
	}
	if len(fixtures) == 0 {
		return nil, errors.NewNotFound("Match not found", nil)
	}

	details := &FixtureDetails{Fixture: mapFixtureRow(fixtures[0])}

	var stats []statsRow
	if err := c.get(ctx, "/fixtures/statistics", url.Values{"fixture": {providerMatchID}}, &stats); err != nil {
		logger.Warn().Err(err).Str("fixture", providerMatchID).Msg("football statistics fetch failed")
	} else {
		details.Stats = mergeStats(stats, details.Fixture.HomeTeam.ProviderID)
	}

	var events []eventRow
	if err := c.get(ctx, "/fixtures/events", url.Values{"fixture": {providerMatchID}}, &events); err != nil {
		logger.Warn().Err(err).Str("fixture", providerMatchID).Msg("football events fetch failed")
	} else {
		details.Events = make([]Event, 0, len(events))
		for _, row := range events {
			details.Events = append(details.Events, Event{
				Elapsed:    derefInt(row.Time.Elapsed),
				Extra:      derefInt(row.Time.Extra),
				TeamID:     strconv.Itoa(row.Team.ID),
				Type:       row.Type,
				PlayerName: row.Player.Name,
				AssistName: row.Assist.Name,
				Detail:     row.Detail,
			})
		}
	}

	var lineups []lineupRow
	if err := c.get(ctx, "/fixtures/lineups", url.Values{"fixture": {providerMatchID}}, &lineups); err != nil {
		logger.Warn().Err(err).Str("fixture", providerMatchID).Msg("football lineups fetch failed")
	} else {
		details.Lineups = make([]Lineup, 0, len(lineups))
		for _, row := range lineups {
			lineup := Lineup{
				TeamID:    strconv.Itoa(row.Team.ID),
				Formation: row.Formation,
			}
			for i, item := range row.StartXI {
				lineup.Players = append(lineup.Players, mapLineupPlayer(item.Player, true, i))
			}
			for i, item := range row.Substitutes {
				lineup.Players = append(lineup.Players, mapLineupPlayer(item.Player, false, 100+i))
			}
			details.Lineups = append(details.Lineups, lineup)
		}
	}

	return details, nil
}

func (c *apiFootballClient) GetPlayer(ctx context.Context, providerPlayerID, season string) (*PlayerProfile, error) {
	if err := c.requireEnabled(); err != nil {
		return nil, err
	}
	params := url.Values{"id": {providerPlayerID}}
	if strings.TrimSpace(season) != "" {
		params.Set("season", season)
	}
	var rows []playerRow
	if err := c.get(ctx, "/players", params, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.NewNotFound("Player not found", nil)
	}
	profile := mapPlayerRow(rows[0])
	return &profile, nil
}

func (c *apiFootballClient) requireEnabled() error {
	if !c.Enabled() {
		return errors.NewServiceUnavailable("Football data provider is not configured", nil)
	}
	return nil
}

func (c *apiFootballClient) get(ctx context.Context, path string, query url.Values, dest any) error {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return errors.NewInternal("Failed to build football provider request", err)
	}
	req.Header.Set("x-apisports-key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.NewServiceUnavailable("Football data provider is unreachable", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return errors.NewInternal("Failed to read football provider response", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return errors.NewTooManyRequests("Football data provider rate limit reached", nil)
	}
	if resp.StatusCode >= 400 {
		return errors.NewServiceUnavailable(fmt.Sprintf("Football data provider returned %d", resp.StatusCode), nil)
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return errors.NewInternal("Failed to decode football provider response", err)
	}
	if dest == nil || len(envelope.Response) == 0 || string(envelope.Response) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Response, dest); err != nil {
		return errors.NewInternal("Failed to decode football provider payload", err)
	}
	return nil
}

func mapTeamRow(row teamRow) Team {
	return Team{
		ProviderID: strconv.Itoa(row.Team.ID),
		Name:       row.Team.Name,
		Code:       row.Team.Code,
		Country:    row.Team.Country,
		LogoURL:    row.Team.Logo,
		VenueName:  row.Venue.Name,
	}
}

func mapFixtureRow(row fixtureRow) Fixture {
	status := MapStatus(row.Fixture.Status.Short)
	return Fixture{
		ProviderID:       strconv.Itoa(row.Fixture.ID),
		Kickoff:          row.Fixture.Date,
		Status:           status,
		Elapsed:          derefInt(row.Fixture.Status.Elapsed),
		Extra:            derefInt(row.Fixture.Status.Extra),
		StadiumName:      row.Fixture.Venue.Name,
		LeagueProviderID: strconv.Itoa(row.League.ID),
		LeagueName:       row.League.Name,
		LeagueCountry:    row.League.Country,
		LeagueLogoURL:    row.League.Logo,
		Season:           strconv.Itoa(row.League.Season),
		HomeTeam:         mapTeamInfo(row.Teams.Home, ""),
		AwayTeam:         mapTeamInfo(row.Teams.Away, ""),
		HomeScore:        row.Goals.Home,
		AwayScore:        row.Goals.Away,
	}
}

func mapTeamInfo(info teamInfo, venue string) Team {
	return Team{
		ProviderID: strconv.Itoa(info.ID),
		Name:       info.Name,
		LogoURL:    info.Logo,
		VenueName:  venue,
	}
}

func mapLineupPlayer(p lineupPlayer, starter bool, order int) LineupPlayer {
	photo := p.Photo
	if photo == "" && p.ID > 0 {
		photo = fmt.Sprintf("https://media.api-sports.io/football/players/%d.png", p.ID)
	}
	_ = order
	id := ""
	if p.ID > 0 {
		id = strconv.Itoa(p.ID)
	}
	return LineupPlayer{
		ProviderPlayerID: id,
		Name:             p.Name,
		Number:           p.Number,
		Position:         MapPosition(p.Pos),
		PhotoURL:         photo,
		Starter:          starter,
	}
}

func mergeStats(rows []statsRow, homeTeamID string) []StatPair {
	home := map[string]int{}
	away := map[string]int{}
	order := []string{}
	seen := map[string]struct{}{}

	for _, row := range rows {
		target := away
		if strconv.Itoa(row.Team.ID) == homeTeamID {
			target = home
		}
		for _, item := range row.Statistics {
			label := MapStatLabel(item.Type)
			if label == "" {
				continue
			}
			target[label] = ParseStatValue(item.Value)
			if _, ok := seen[label]; !ok {
				seen[label] = struct{}{}
				order = append(order, label)
			}
		}
	}

	preferred := PreferredStatOrder()
	out := make([]StatPair, 0, len(preferred)+len(order))
	used := map[string]struct{}{}
	for _, label := range preferred {
		if _, ok := seen[label]; !ok {
			continue
		}
		out = append(out, StatPair{Label: label, Home: home[label], Away: away[label]})
		used[label] = struct{}{}
	}
	for _, label := range order {
		if _, ok := used[label]; ok {
			continue
		}
		out = append(out, StatPair{Label: label, Home: home[label], Away: away[label]})
	}
	return out
}

func mapPlayerRow(row playerRow) PlayerProfile {
	profile := PlayerProfile{
		ProviderID:    strconv.Itoa(row.Player.ID),
		Name:          row.Player.Name,
		Age:           row.Player.Age,
		Nationality:   row.Player.Nationality,
		HeightCM:      parseMeasure(row.Player.Height),
		WeightKG:      parseMeasure(row.Player.Weight),
		PhotoURL:      row.Player.Photo,
		PreferredFoot: "",
	}
	if len(row.Statistics) > 0 {
		stat := row.Statistics[0]
		profile.ClubName = stat.Team.Name
		profile.ClubProviderID = strconv.Itoa(stat.Team.ID)
		profile.Position = MapPosition(stat.Games.Position)
		if stat.Games.Number != nil {
			profile.Number = *stat.Games.Number
		}
		profile.Radar = []RadarStat{
			{Label: "Attack", Value: clampStat(derefInt(stat.Shots.Total) * 8)},
			{Label: "Skill", Value: clampStat(derefInt(stat.Dribbles.Success) * 10)},
			{Label: "Defence", Value: clampStat(derefInt(stat.Tackles.Total) * 8)},
			{Label: "Tactic", Value: clampStat(ParseStatValue(stat.Passes.Accuracy))},
			{Label: "Creativity", Value: clampStat(derefInt(stat.Passes.Key) * 12)},
		}
	}
	return profile
}

func parseMeasure(raw string) int {
	s := strings.TrimSpace(raw)
	for i, r := range s {
		if r < '0' || r > '9' {
			s = s[:i]
			break
		}
	}
	n, _ := strconv.Atoi(s)
	return n
}

func clampStat(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
