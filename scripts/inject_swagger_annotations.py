#!/usr/bin/env python3
"""Inject swag annotations into handler methods missing @Router comments."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def ann(
    summary: str,
    method_path: str,
    tags: str,
    *,
    auth: bool = False,
    body: str | None = None,
    params: list[tuple] | None = None,
    queries: list[tuple] | None = None,
    success: str = "200",
    form: bool = False,
    form_fields: list[tuple] | None = None,
    desc: str | None = None,
    no_content: bool = False,
) -> str:
    method, path = method_path.split(" ", 1)
    lines = [f"{summary} godoc", "", f"\t@Summary\t\t{summary}"]
    if desc:
        lines.append(f"\t@Description\t{desc}")
    lines.append(f"\t@Tags\t\t\t{tags}")
    if form:
        lines.append("\t@Accept\t\t\tmpfd")
    elif body:
        lines.append("\t@Accept\t\t\tjson")
    lines.append("\t@Produce\t\tjson")
    if auth:
        lines.append("\t@Security\t\tBearerAuth")
    if params:
        for name, typ, d in params:
            lines.append(f'\t@Param\t\t\t{name}\tpath\t{typ}\ttrue\t"{d}"')
    if queries:
        for name, typ, required, d in queries:
            req = "true" if required else "false"
            lines.append(f'\t@Param\t\t\t{name}\tquery\t{typ}\t{req}\t"{d}"')
    if body:
        lines.append(f'\t@Param\t\t\tbody\tbody\t\t{body}\ttrue\t"Request body"')
    if form_fields:
        for name, typ, required, d in form_fields:
            req = "true" if required else "false"
            lines.append(f'\t@Param\t\t\t{name}\tformData\t{typ}\t{req}\t"{d}"')
    elif form:
        lines.append('\t@Param\t\t\tfile\tformData\tfile\ttrue\t"Upload file"')
    if no_content:
        lines.append("\t@Success\t\t204\t\"No Content\"")
    else:
        lines.append(f"\t@Success\t\t{success}\t{{object}}\tresponse.Response")
    if auth:
        lines.append("\t@Failure\t\t401\t{object}\tresponse.Response")
    lines.append("\t@Failure\t\t400\t{object}\tresponse.Response")
    lines.append(f"\t@Router\t\t\t{path} [{method.lower()}]")
    return "\n".join("//" if not ln else f"// {ln}" for ln in lines)


def add(catalog: dict, file: str, func: str, **kw):
    catalog.setdefault(file, {})[func] = ann(**kw)


def build_catalog() -> dict[str, dict[str, str]]:
    C: dict[str, dict[str, str]] = {}
    page_q = [
        ("sort", "string", False, "Sort field"),
        ("order", "string", False, "asc|desc"),
        ("page", "int", False, "Page number"),
        ("per_page", "int", False, "Items per page"),
    ]

    f = "internal/modules/club/handler/club_handler.go"
    add(C, f, "Create", summary="Create club", method_path="post /api/v1/clubs", tags="clubs", auth=True, body="dto.CreateClubRequest", success="201")
    add(C, f, "List", summary="List clubs", method_path="get /api/v1/clubs", tags="clubs", queries=page_q + [("q", "string", False, "Search query")])
    add(C, f, "Search", summary="Search clubs", method_path="get /api/v1/clubs/search", tags="clubs", queries=[("q", "string", True, "Search query")])
    add(C, f, "GetByID", summary="Get club", method_path="get /api/v1/clubs/{id}", tags="clubs", params=[("id", "string", "Club ID")])
    add(C, f, "Update", summary="Update club", method_path="put /api/v1/clubs/{id}", tags="clubs", auth=True, body="dto.UpdateClubRequest", params=[("id", "string", "Club ID")])
    add(C, f, "Delete", summary="Delete club", method_path="delete /api/v1/clubs/{id}", tags="clubs", auth=True, params=[("id", "string", "Club ID")])

    f = "internal/modules/league/handler/league_handler.go"
    add(C, f, "Create", summary="Create league", method_path="post /api/v1/leagues", tags="leagues", auth=True, body="dto.CreateLeagueRequest", success="201")
    add(C, f, "List", summary="List leagues", method_path="get /api/v1/leagues", tags="leagues", queries=page_q)
    add(C, f, "GetByID", summary="Get league", method_path="get /api/v1/leagues/{id}", tags="leagues", params=[("id", "string", "League ID")])
    add(C, f, "Update", summary="Update league", method_path="put /api/v1/leagues/{id}", tags="leagues", auth=True, body="dto.UpdateLeagueRequest", params=[("id", "string", "League ID")])
    add(C, f, "Delete", summary="Delete league", method_path="delete /api/v1/leagues/{id}", tags="leagues", auth=True, params=[("id", "string", "League ID")])

    f = "internal/modules/season/handler/season_handler.go"
    add(C, f, "Create", summary="Create season", method_path="post /api/v1/seasons", tags="seasons", auth=True, body="dto.CreateSeasonRequest", success="201")
    add(C, f, "List", summary="List seasons", method_path="get /api/v1/seasons", tags="seasons", queries=page_q)
    add(C, f, "GetByID", summary="Get season", method_path="get /api/v1/seasons/{id}", tags="seasons", params=[("id", "string", "Season ID")])
    add(C, f, "Update", summary="Update season", method_path="put /api/v1/seasons/{id}", tags="seasons", auth=True, body="dto.UpdateSeasonRequest", params=[("id", "string", "Season ID")])
    add(C, f, "Delete", summary="Delete season", method_path="delete /api/v1/seasons/{id}", tags="seasons", auth=True, params=[("id", "string", "Season ID")])
    add(C, f, "ListByLeagueID", summary="List seasons by league", method_path="get /api/v1/leagues/{id}/seasons", tags="seasons", params=[("id", "string", "League ID")])

    f = "internal/modules/clubseason/handler/club_season_handler.go"
    add(C, f, "AddClubToSeason", summary="Add club to season", method_path="post /api/v1/club-seasons", tags="club-seasons", auth=True, body="dto.CreateClubSeasonRequest", success="201")
    add(C, f, "UpdateStatus", summary="Update club-season status", method_path="patch /api/v1/club-seasons/{id}", tags="club-seasons", auth=True, body="dto.UpdateClubSeasonRequest", params=[("id", "string", "ClubSeason ID")])
    add(C, f, "ListClubsInSeason", summary="List clubs in season", method_path="get /api/v1/seasons/{id}/clubs", tags="club-seasons", auth=True, params=[("id", "string", "Season ID")])
    add(C, f, "RemoveClubFromSeason", summary="Remove club from season", method_path="delete /api/v1/seasons/{id}/clubs/{club_id}", tags="club-seasons", auth=True, params=[("id", "string", "Season ID"), ("club_id", "string", "Club ID")])
    add(C, f, "ListSeasonsForClub", summary="List seasons for club", method_path="get /api/v1/clubs/{id}/seasons", tags="club-seasons", auth=True, params=[("id", "string", "Club ID")])

    f = "internal/modules/match/handler/match_handler.go"
    add(C, f, "Create", summary="Create match", method_path="post /api/v1/matches", tags="matches", auth=True, body="dto.CreateMatchRequest", success="201")
    add(C, f, "List", summary="List matches", method_path="get /api/v1/matches", tags="matches", queries=page_q)
    add(C, f, "ListUpcoming", summary="List upcoming matches", method_path="get /api/v1/matches/upcoming", tags="matches")
    add(C, f, "ListLive", summary="List live matches", method_path="get /api/v1/matches/live", tags="matches")
    add(C, f, "GetByID", summary="Get match", method_path="get /api/v1/matches/{id}", tags="matches", params=[("id", "string", "Match ID")])
    add(C, f, "Update", summary="Update match", method_path="put /api/v1/matches/{id}", tags="matches", auth=True, body="dto.UpdateMatchRequest", params=[("id", "string", "Match ID")])
    add(C, f, "Delete", summary="Delete match", method_path="delete /api/v1/matches/{id}", tags="matches", auth=True, params=[("id", "string", "Match ID")])
    add(C, f, "ListBySeason", summary="List matches by season", method_path="get /api/v1/seasons/{id}/matches", tags="matches", params=[("id", "string", "Season ID")])
    add(C, f, "ListByLeague", summary="List matches by league", method_path="get /api/v1/leagues/{id}/matches", tags="matches", params=[("id", "string", "League ID")])
    add(C, f, "ListByClub", summary="List matches by club", method_path="get /api/v1/clubs/{id}/matches", tags="matches", params=[("id", "string", "Club ID")])

    f = "internal/modules/song/handler/song_handler.go"
    add(C, f, "Create", summary="Create song", method_path="post /api/v1/songs", tags="songs", auth=True, body="dto.CreateSongRequest", success="201")
    add(C, f, "List", summary="List songs", method_path="get /api/v1/songs", tags="songs", queries=page_q + [("q", "string", False, "Search query")])
    add(C, f, "Search", summary="Search songs", method_path="get /api/v1/songs/search", tags="songs", queries=[("q", "string", True, "Search query")])
    add(C, f, "GetByID", summary="Get song", method_path="get /api/v1/songs/{id}", tags="songs", params=[("id", "string", "Song ID")])
    add(C, f, "Update", summary="Update song", method_path="put /api/v1/songs/{id}", tags="songs", auth=True, body="dto.UpdateSongRequest", params=[("id", "string", "Song ID")])
    add(C, f, "Delete", summary="Delete song", method_path="delete /api/v1/songs/{id}", tags="songs", auth=True, params=[("id", "string", "Song ID")])

    f = "internal/modules/songlyric/handler/song_lyric_handler.go"
    add(C, f, "Create", summary="Create song lyric", method_path="post /api/v1/song-lyrics", tags="song-lyrics", auth=True, body="dto.CreateSongLyricRequest", success="201")
    add(C, f, "GetByID", summary="Get song lyric", method_path="get /api/v1/song-lyrics/{id}", tags="song-lyrics", params=[("id", "string", "Lyric ID")])
    add(C, f, "Update", summary="Update song lyric", method_path="put /api/v1/song-lyrics/{id}", tags="song-lyrics", auth=True, body="dto.UpdateSongLyricRequest", params=[("id", "string", "Lyric ID")])
    add(C, f, "Delete", summary="Delete song lyric", method_path="delete /api/v1/song-lyrics/{id}", tags="song-lyrics", auth=True, params=[("id", "string", "Lyric ID")])
    add(C, f, "ListBySongID", summary="List lyrics for song", method_path="get /api/v1/songs/{id}/lyrics", tags="song-lyrics", params=[("id", "string", "Song ID")])
    add(C, f, "GetBySongID", summary="Get lyrics by language", method_path="get /api/v1/songs/{id}/lyrics/{language}", tags="song-lyrics", params=[("id", "string", "Song ID"), ("language", "string", "Language code")])
    add(C, f, "ImportLyrics", summary="Import lyrics", method_path="post /api/v1/songs/{id}/lyrics/import", tags="song-lyrics", auth=True, body="dto.ImportLyricsRequest", params=[("id", "string", "Song ID")])

    f = "internal/modules/matchsongschedule/handler/match_song_schedule_handler.go"
    add(C, f, "Create", summary="Create match song schedule", method_path="post /api/v1/match-song-schedules", tags="match-song-schedules", auth=True, body="dto.CreateMatchSongScheduleRequest", success="201")
    add(C, f, "List", summary="List match song schedules", method_path="get /api/v1/match-song-schedules", tags="match-song-schedules", queries=page_q)
    add(C, f, "GetByID", summary="Get match song schedule", method_path="get /api/v1/match-song-schedules/{id}", tags="match-song-schedules", params=[("id", "string", "Schedule ID")])
    add(C, f, "Update", summary="Update match song schedule", method_path="put /api/v1/match-song-schedules/{id}", tags="match-song-schedules", auth=True, body="dto.UpdateMatchSongScheduleRequest", params=[("id", "string", "Schedule ID")])
    add(C, f, "Delete", summary="Delete match song schedule", method_path="delete /api/v1/match-song-schedules/{id}", tags="match-song-schedules", auth=True, params=[("id", "string", "Schedule ID")])
    add(C, f, "ListByMatchID", summary="List schedules by match", method_path="get /api/v1/matches/{id}/song-schedules", tags="match-song-schedules", params=[("id", "string", "Match ID")])
    add(C, f, "ListBySongID", summary="List schedules by song", method_path="get /api/v1/songs/{id}/match-schedules", tags="match-song-schedules", params=[("id", "string", "Song ID")])

    f = "internal/modules/media/handler/media_handler.go"
    add(C, f, "Upload", summary="Upload media", method_path="post /api/v1/media/upload", tags="media", auth=True, form=True, form_fields=[("file", "file", True, "Media file"), ("type", "string", False, "Media type")])
    add(C, f, "GetPlaybackURL", summary="Get playback URL", method_path="get /api/v1/media/{id}/playback-url", tags="media", auth=True, params=[("id", "string", "Media ID")])
    add(C, f, "UploadSongAudio", summary="Upload song audio", method_path="post /api/v1/songs/{id}/audio", tags="media", auth=True, form=True, params=[("id", "string", "Song ID")], form_fields=[("file", "file", True, "Audio file")])

    f = "internal/modules/chant/handler/chant_handler.go"
    add(C, f, "List", summary="List chants", method_path="get /api/v1/chants", tags="chants", auth=True, queries=[("search", "string", False, "Search query")])
    add(C, f, "Countdown", summary="Chant countdown", method_path="get /api/v1/chants/{chant_id}/countdown", tags="chants", auth=True, params=[("chant_id", "string", "Chant ID")])
    add(C, f, "Lyrics", summary="Chant lyrics", method_path="get /api/v1/chants/{chant_id}/lyrics", tags="chants", auth=True, params=[("chant_id", "string", "Chant ID")])
    add(C, f, "Complete", summary="Complete chant", method_path="post /api/v1/chants/{chant_id}/complete", tags="chants", auth=True, params=[("chant_id", "string", "Chant ID")])

    f = "internal/modules/guess/handler/guess_handler.go"
    add(C, f, "MatchOverview", summary="Guess match overview", method_path="get /api/v1/guess/matches/{match_id}", tags="guess", auth=True, params=[("match_id", "string", "Match ID")])
    add(C, f, "QuizDetail", summary="Quiz detail", method_path="get /api/v1/guess/quizzes/{quiz_id}", tags="guess", auth=True, params=[("quiz_id", "string", "Quiz ID")])
    add(C, f, "Answer", summary="Answer quiz", method_path="post /api/v1/guess/quizzes/{quiz_id}/answer", tags="guess", auth=True, body="dto.AnswerQuizRequest", params=[("quiz_id", "string", "Quiz ID")])

    f = "internal/modules/shop/handler/shop_handler.go"
    add(C, f, "ListSnacks", summary="List snacks", method_path="get /api/v1/snacks", tags="shop", auth=True, queries=[("search", "string", False, "Search"), ("category", "string", False, "Category"), ("currency", "string", False, "Currency")])
    add(C, f, "SnackDetail", summary="Snack detail", method_path="get /api/v1/snacks/{snack_id}", tags="shop", auth=True, params=[("snack_id", "string", "Snack ID")], queries=[("currency", "string", False, "Currency")])
    add(C, f, "ListProducts", summary="List products", method_path="get /api/v1/products", tags="shop", auth=True, queries=[("search", "string", False, "Search"), ("category", "string", False, "Category"), ("currency", "string", False, "Currency")])
    add(C, f, "ProductDetail", summary="Product detail", method_path="get /api/v1/products/{product_id}", tags="shop", auth=True, params=[("product_id", "string", "Product ID")], queries=[("currency", "string", False, "Currency")])
    add(C, f, "GetCart", summary="Get cart", method_path="get /api/v1/cart", tags="shop", auth=True)
    add(C, f, "AddCartItem", summary="Add cart item", method_path="post /api/v1/cart/items", tags="shop", auth=True, body="dto.AddCartItemRequest")
    add(C, f, "UpdateCartItem", summary="Update cart item", method_path="patch /api/v1/cart/items/{item_id}", tags="shop", auth=True, body="dto.UpdateCartItemRequest", params=[("item_id", "string", "Cart item ID")])
    add(C, f, "RemoveCartItem", summary="Remove cart item", method_path="delete /api/v1/cart/items/{item_id}", tags="shop", auth=True, params=[("item_id", "string", "Cart item ID")])
    add(C, f, "Checkout", summary="Checkout cart", method_path="post /api/v1/orders", tags="shop", auth=True, body="dto.CheckoutRequest", success="201")
    add(C, f, "Pay", summary="Pay order", method_path="post /api/v1/orders/{order_id}/pay", tags="shop", auth=True, body="dto.PayOrderRequest", params=[("order_id", "string", "Order ID")])

    f = "internal/modules/video/handler/video_handler.go"
    add(C, f, "Feed", summary="Video feed", method_path="get /api/v1/videos/feed", tags="videos", auth=True)
    add(C, f, "Mine", summary="My videos", method_path="get /api/v1/videos/mine", tags="videos", auth=True)
    add(C, f, "Upload", summary="Upload video", method_path="post /api/v1/videos", tags="videos", auth=True, form=True, form_fields=[("file", "file", True, "Video file"), ("caption", "string", False, "Caption")])
    add(C, f, "Like", summary="Like video", method_path="post /api/v1/videos/{video_id}/like", tags="videos", auth=True, params=[("video_id", "string", "Video ID")])
    add(C, f, "Unlike", summary="Unlike video", method_path="delete /api/v1/videos/{video_id}/like", tags="videos", auth=True, params=[("video_id", "string", "Video ID")])

    f = "internal/modules/stats/handler/player_handler.go"
    add(C, f, "PlayerDetail", summary="Player detail", method_path="get /api/v1/players/{player_id}", tags="stats", auth=True, params=[("player_id", "string", "Player ID")])

    f = "internal/modules/news/handler/news_handler.go"
    add(C, f, "List", summary="List news", method_path="get /api/v1/news", tags="news", auth=True)

    f = "internal/modules/mobilehome/handler/home_handler.go"
    add(C, f, "Stadium", summary="Stadium home", method_path="get /api/v1/mobile/home/stadium", tags="mobile-home", auth=True)
    add(C, f, "Club", summary="Club home", method_path="get /api/v1/mobile/home/club", tags="mobile-home", auth=True)

    f = "internal/modules/user/handler/profile_handler.go"
    add(C, f, "GetProfile", summary="Get profile", method_path="get /api/v1/profiles/me", tags="profiles", auth=True)
    add(C, f, "CreateProfile", summary="Create profile", method_path="post /api/v1/profiles/me", tags="profiles", auth=True, body="CreateProfileRequest", success="201")
    add(C, f, "UpdateProfile", summary="Update profile", method_path="put /api/v1/profiles/me", tags="profiles", auth=True, body="UpdateProfileRequest")
    add(C, f, "DeleteProfile", summary="Delete profile", method_path="delete /api/v1/profiles/me", tags="profiles", auth=True)

    f = "internal/modules/user/handler/mobile_profile_handler.go"
    add(C, f, "GetMe", summary="Mobile profile me", method_path="get /api/v1/profile/me", tags="mobile-profile", auth=True)
    add(C, f, "UpdateMe", summary="Update mobile profile", method_path="patch /api/v1/profile/me", tags="mobile-profile", auth=True, body="dto.UpdateMobileProfileRequest")
    add(C, f, "UploadAvatar", summary="Upload avatar", method_path="post /api/v1/profile/me/avatar", tags="mobile-profile", auth=True, form=True, form_fields=[("file", "file", True, "Avatar image")])
    add(C, f, "Leaderboard", summary="Leaderboard", method_path="get /api/v1/profile/leaderboard", tags="mobile-profile", auth=True)

    f = "internal/modules/matchruntime/handler/match_runtime_handler.go"
    add(C, f, "Start", summary="Start match runtime", method_path="post /api/v1/matches/{id}/runtime/start", tags="match-runtime", auth=True, params=[("id", "string", "Match ID")])
    add(C, f, "Pause", summary="Pause match runtime", method_path="post /api/v1/matches/{id}/runtime/pause", tags="match-runtime", auth=True, params=[("id", "string", "Match ID")])
    add(C, f, "Resume", summary="Resume match runtime", method_path="post /api/v1/matches/{id}/runtime/resume", tags="match-runtime", auth=True, params=[("id", "string", "Match ID")])
    add(C, f, "End", summary="End match runtime", method_path="post /api/v1/matches/{id}/runtime/end", tags="match-runtime", auth=True, params=[("id", "string", "Match ID")])
    add(C, f, "GetState", summary="Get match runtime state", method_path="get /api/v1/matches/{id}/runtime", tags="match-runtime", params=[("id", "string", "Match ID")])

    f = "internal/modules/playback/handler/playback_handler.go"
    add(C, f, "ScheduleSong", summary="Schedule song playback", method_path="post /api/v1/songs/schedule", tags="playback", auth=True, body="dto.ScheduleSongRequest", success="201")
    add(C, f, "CancelSong", summary="Cancel scheduled song", method_path="delete /api/v1/songs/schedule/{id}", tags="playback", auth=True, params=[("id", "string", "Schedule ID")])
    add(C, f, "GetUpcomingSongs", summary="Upcoming scheduled songs", method_path="get /api/v1/songs/schedule/upcoming", tags="playback", auth=True, queries=[("match_id", "string", True, "Match ID")])

    f = "internal/modules/realtime/handler/time_sync_handler.go"
    add(C, f, "GetServerTime", summary="Server time", method_path="get /api/v1/realtime/time", tags="realtime")
    add(C, f, "Sync", summary="Time sync", method_path="post /api/v1/realtime/time-sync", tags="realtime", body="dto.TimeSyncRequest")

    f = "internal/modules/realtime/handler/ws_handler.go"
    # Uses map responses in annotations (package does not import response).
    C.setdefault(f, {})["Connect"] = ann(
        "WebSocket connect",
        "get /api/v1/realtime/ws",
        "realtime",
        auth=True,
        desc="Upgrade to WebSocket. JWT required (query or header).",
    ).replace("response.Response", "map[string]interface{}").replace(
        '@Success\t\t200\t{object}\tmap[string]interface{}',
        '@Success\t\t101\t"Switching Protocols"',
    )

    f = "internal/modules/realtime/handler/recovery_handler.go"
    add(C, f, "GetMatchState", summary="Realtime session recovery", method_path="get /api/v1/realtime/session/{matchId}", tags="realtime", auth=True, params=[("matchId", "string", "Match ID")])

    f = "internal/modules/realtime/handler/metrics_handler.go"
    C.setdefault(f, {})["GetMetrics"] = ann(
        "Realtime metrics",
        "get /api/v1/realtime/metrics",
        "realtime",
        auth=True,
        desc="Admin only",
    ).replace("response.Response", "map[string]interface{}")

    f = "internal/modules/realtime/handler/retention_handler.go"
    add(C, f, "CleanupSchedulerEvents", summary="Cleanup scheduler events", method_path="post /api/v1/realtime/admin/cleanup/scheduler-events", tags="realtime", auth=True, desc="Admin only")
    add(C, f, "CleanupRealtimeEvents", summary="Cleanup realtime events", method_path="post /api/v1/realtime/admin/cleanup/realtime-events", tags="realtime", auth=True, desc="Admin only")
    add(C, f, "CleanupHeartbeats", summary="Cleanup heartbeats", method_path="post /api/v1/realtime/admin/cleanup/heartbeats", tags="realtime", auth=True, desc="Admin only")
    add(C, f, "CleanupAll", summary="Cleanup all retention data", method_path="post /api/v1/realtime/admin/cleanup/all", tags="realtime", auth=True, desc="Admin only")

    return C


FUNC_RE = re.compile(r"^func \(h \*\w+\) (\w+)\(c \*gin\.Context\) \{", re.M)


def ensure_response_import(text: str) -> str:
    if '"clap/internal/shared/response"' in text:
        return text
    m = re.search(r"import \(\n([\s\S]*?)\n\)", text)
    if not m:
        return text
    block = m.group(1)
    insert_at = m.start(1) + len(block)
    return text[:insert_at] + '\n\t"clap/internal/shared/response"' + text[insert_at:]


def preceding_has_router(text: str, func_start: int) -> bool:
    lines_before = text[:func_start].splitlines()
    i = len(lines_before) - 1
    while i >= 0 and lines_before[i].strip() == "":
        i -= 1
    while i >= 0 and lines_before[i].lstrip().startswith("//"):
        if "@Router" in lines_before[i]:
            return True
        i -= 1
    return False


def inject_file(path: Path, funcs: dict[str, str]) -> tuple[int, list[str]]:
    original = path.read_text()
    text = original
    inserted = 0
    matches = list(FUNC_RE.finditer(text))
    found = {m.group(1) for m in matches}
    missing = sorted(set(funcs) - found)

    for m in reversed(matches):
        name = m.group(1)
        if name not in funcs or preceding_has_router(text, m.start()):
            continue
        text = text[: m.start()] + funcs[name] + "\n" + text[m.start() :]
        inserted += 1

    if inserted:
        # Only import response if the package already references it in code
        # (swagger comments alone must not create unused imports).
        if "response." in original:
            text = ensure_response_import(text)
        path.write_text(text)
    return inserted, missing


def main():
    catalog = build_catalog()
    total = 0
    for rel, funcs in sorted(catalog.items()):
        path = ROOT / rel
        if not path.exists():
            print(f"MISSING FILE: {rel}")
            continue
        n, missing = inject_file(path, funcs)
        total += n
        print(f"{rel}: +{n} annotations" + (f" (missing funcs: {missing})" if missing else ""))
    print(f"TOTAL inserted: {total}")


if __name__ == "__main__":
    main()
