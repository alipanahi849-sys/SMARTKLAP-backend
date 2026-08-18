package newsfeed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEncodeDecodeID(t *testing.T) {
	id := "football/2026/jul/14/youri-tielemans-signs-manchester-united-aston-villa"
	encoded := EncodeID(id)
	if encoded == "" || strings.Contains(encoded, "/") {
		t.Fatalf("encoded=%q", encoded)
	}
	got, err := DecodeID(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if got != id {
		t.Fatalf("got %q want %q", got, id)
	}
}

func TestPickFootballTagPrefersExactClubKeyword(t *testing.T) {
	got := pickFootballTag("Manchester United", []guardianTag{
		{ID: "football/manchester-united-women", Type: "keyword", WebTitle: "Manchester United Women"},
		{ID: "football/western-united", Type: "keyword", WebTitle: "Western United"},
		{ID: "football/manchester-united", Type: "keyword", WebTitle: "Manchester United"},
	})
	if got != "football/manchester-united" {
		t.Fatalf("got %q", got)
	}
}

func TestGuardianSearchMapsPublisherFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"response": map[string]any{
					"status": "ok",
					"results": []map[string]string{
						{"id": "football/manchester-united", "type": "keyword", "webTitle": "Manchester United", "sectionId": "football"},
					},
				},
			})
			return
		case "/search":
			if r.URL.Query().Get("tag") != "football/manchester-united" {
				t.Fatalf("tag=%s", r.URL.Query().Get("tag"))
			}
			if r.URL.Query().Get("q") != "" {
				t.Fatalf("q should be empty when a club tag is used, got %s", r.URL.Query().Get("q"))
			}
			if r.URL.Query().Get("section") != "football" {
				t.Fatalf("section=%s", r.URL.Query().Get("section"))
			}
		default:
			t.Fatalf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"status":      "ok",
				"currentPage": 1,
				"pages":       3,
				"results": []map[string]any{
					{
						"id":                 "football/2026/jul/14/tielemans-united",
						"webTitle":           "fallback title",
						"webPublicationDate": "2026-07-14T18:20:15Z",
						"fields": map[string]string{
							"headline":  "Tielemans joins Manchester United",
							"trailText": "United sign midfielder",
							"body":      "<p>Manchester United have confirmed the signing.</p>",
							"thumbnail": "https://media.guim.co.uk/thumb.jpg",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewGuardian("test", server.URL)
	result, err := client.Search(context.Background(), "Manchester United", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalPages != 3 || len(result.Items) != 1 {
		t.Fatalf("%+v", result)
	}
	item := result.Items[0]
	if item.Title != "Tielemans joins Manchester United" {
		t.Fatalf("title=%q", item.Title)
	}
	if item.BodyHTML != "<p>Manchester United have confirmed the signing.</p>" {
		t.Fatalf("body=%q", item.BodyHTML)
	}
	if item.ImageURL != "https://media.guim.co.uk/thumb.jpg" {
		t.Fatalf("image=%q", item.ImageURL)
	}
	if !item.PublishedAt.Equal(time.Date(2026, 7, 14, 18, 20, 15, 0, time.UTC)) {
		t.Fatalf("published=%v", item.PublishedAt)
	}
}

func TestGuardianGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/football/2026/jul/14/tielemans-united" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"status": "ok",
				"content": map[string]any{
					"id":                 "football/2026/jul/14/tielemans-united",
					"webTitle":           "Tielemans joins Manchester United",
					"webPublicationDate": "2026-07-14T18:20:15Z",
					"fields": map[string]string{
						"headline": "Tielemans joins Manchester United",
						"body":     "<p>Full article from The Guardian.</p>",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewGuardian("test", server.URL)
	article, err := client.Get(context.Background(), "football/2026/jul/14/tielemans-united")
	if err != nil {
		t.Fatal(err)
	}
	if article.BodyHTML != "<p>Full article from The Guardian.</p>" {
		t.Fatalf("body=%q", article.BodyHTML)
	}
}
