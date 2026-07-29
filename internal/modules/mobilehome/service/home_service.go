package service

import (
	"context"
	"fmt"
	"time"

	authrepo "clap/internal/modules/auth/repository"
	chantsvc "clap/internal/modules/chant/service"
	clubrepo "clap/internal/modules/club/repository"
	matchrepo "clap/internal/modules/match/repository"
	"clap/internal/modules/mobilehome/dto"
	newssvc "clap/internal/modules/news/service"
	shopsvc "clap/internal/modules/shop/service"
	"clap/internal/shared/logger"

	"github.com/google/uuid"
)

const previewLimit = 3

// HomeService aggregates existing module data for the two mobile Home screens
// (Mobile API Contract §3). It owns no persistence of its own.
type HomeService interface {
	Stadium(ctx context.Context, userID uuid.UUID) (*dto.StadiumHomeResponse, error)
	Club(ctx context.Context) (*dto.ClubHomeResponse, error)
}

type homeService struct {
	userRepo  authrepo.UserRepository
	matchRepo matchrepo.MatchRepository
	clubRepo  clubrepo.ClubRepository
	chantSvc  chantsvc.ChantService
	shopSvc   shopsvc.ShopService
	newsSvc   newssvc.NewsService
}

func NewHomeService(
	userRepo authrepo.UserRepository,
	matchRepo matchrepo.MatchRepository,
	clubRepo clubrepo.ClubRepository,
	chantSvc chantsvc.ChantService,
	shopSvc shopsvc.ShopService,
	newsSvc newssvc.NewsService,
) HomeService {
	return &homeService{
		userRepo:  userRepo,
		matchRepo: matchRepo,
		clubRepo:  clubRepo,
		chantSvc:  chantSvc,
		shopSvc:   shopSvc,
		newsSvc:   newsSvc,
	}
}

func (s *homeService) Stadium(ctx context.Context, userID uuid.UUID) (*dto.StadiumHomeResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	cartCount, err := s.shopSvc.CartCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := &dto.StadiumHomeResponse{
		UserSummary: dto.UserSummary{Points: user.Points, CartCount: cartCount},
	}

	// live_match is null (not 404) when no match is live — contract §3.1.
	if liveMatch, liveErr := s.liveMatch(ctx); liveErr != nil {
		logger.Warn().Err(liveErr).Msg("home stadium: live match lookup failed")
	} else {
		resp.LiveMatch = liveMatch
	}

	todayPoints, todayTarget, recent, chants, err := s.chantSvc.TodayProgram(ctx, userID, previewLimit)
	if err != nil {
		return nil, err
	}
	program := dto.ChantProgram{
		TodayPoints: todayPoints,
		TodayTarget: todayTarget,
		RecentItems: make([]dto.ChantProgramItem, 0, len(recent)),
	}
	now := time.Now()
	for _, completion := range recent {
		title := "Chant"
		if chant, ok := chants[completion.ChantID]; ok {
			title = chant.Title
		}
		minutesAgo := int(now.Sub(completion.CreatedAt).Minutes())
		if minutesAgo < 0 {
			minutesAgo = 0
		}
		program.RecentItems = append(program.RecentItems, dto.ChantProgramItem{
			ID:         completion.ID,
			Title:      title,
			Subtitle:   fmt.Sprintf("You've gotten %d point", completion.PointsEarned),
			MinutesAgo: minutesAgo,
			Status:     "done",
		})
	}
	resp.ChantProgram = program

	foods, err := s.shopSvc.SnacksPreview(ctx, previewLimit)
	if err != nil {
		return nil, err
	}
	resp.Foods = foods

	return resp, nil
}

func (s *homeService) liveMatch(ctx context.Context) (*dto.LiveMatch, error) {
	matches, err := s.matchRepo.FindLive(ctx)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}

	match := matches[0]
	live := &dto.LiveMatch{
		ID:     match.ID,
		Status: match.Status,
		Minute: match.CurrentMinute,
	}
	if match.HomeScore != nil {
		live.HomeTeam.Score = *match.HomeScore
	}
	if match.AwayScore != nil {
		live.AwayTeam.Score = *match.AwayScore
	}
	clubs, clubErr := s.clubRepo.FindByIDs(ctx, []uuid.UUID{match.HomeClubID, match.AwayClubID})
	if clubErr != nil {
		return nil, clubErr
	}
	if home, ok := clubs[match.HomeClubID]; ok {
		live.HomeTeam.Name = home.Name
		live.HomeTeam.LogoURL = home.LogoURL
	}
	if away, ok := clubs[match.AwayClubID]; ok {
		live.AwayTeam.Name = away.Name
		live.AwayTeam.LogoURL = away.LogoURL
	}
	return live, nil
}

func (s *homeService) Club(ctx context.Context) (*dto.ClubHomeResponse, error) {
	matches, _, err := s.matchRepo.FindUpcoming(ctx, 1, previewLimit)
	if err != nil {
		return nil, err
	}

	clubIDs := make([]uuid.UUID, 0, len(matches)*2)
	for _, match := range matches {
		clubIDs = append(clubIDs, match.HomeClubID, match.AwayClubID)
	}
	clubs, err := s.clubRepo.FindByIDs(ctx, clubIDs)
	if err != nil {
		return nil, err
	}

	upcoming := make([]dto.UpcomingMatch, 0, len(matches))
	now := time.Now()
	for _, match := range matches {
		item := dto.UpcomingMatch{
			ID:     match.ID,
			Date:   match.MatchDateTime.Format("2006-01-02"),
			Time:   match.MatchDateTime.Format("15:04"),
			Status: mapMatchStatus(match.Status),
		}
		if secs := int64(match.MatchDateTime.Sub(now).Seconds()); secs > 0 {
			item.CountdownSeconds = secs
		}
		if match.Status == "finished" && match.HomeScore != nil && match.AwayScore != nil {
			score := fmt.Sprintf("%d : %d", *match.HomeScore, *match.AwayScore)
			item.Score = &score
		}
		if home, ok := clubs[match.HomeClubID]; ok {
			item.HomeName = home.Name
			item.HomeLogoURL = home.LogoURL
		}
		if away, ok := clubs[match.AwayClubID]; ok {
			item.AwayName = away.Name
			item.AwayLogoURL = away.LogoURL
		}
		upcoming = append(upcoming, item)
	}

	store, err := s.shopSvc.ProductsPreview(ctx, previewLimit)
	if err != nil {
		return nil, err
	}

	newsItems, err := s.newsSvc.Preview(ctx, previewLimit)
	if err != nil {
		return nil, err
	}

	return &dto.ClubHomeResponse{
		UpcomingMatches: upcoming,
		ClubStore:       store,
		ClubNews:        newsItems,
	}, nil
}

// mapMatchStatus converts internal match statuses to the contract's
// upcoming/live/finished vocabulary (§3.2).
func mapMatchStatus(status string) string {
	switch status {
	case "live", "halftime":
		return "live"
	case "finished":
		return "finished"
	default:
		return "upcoming"
	}
}
