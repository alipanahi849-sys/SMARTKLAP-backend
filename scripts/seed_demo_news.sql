-- Demo seed: club news for the mobile feed (rich HTML + photos).
-- Safe to re-run: upserts fixed UUIDs and refreshes title, body, image and dates.

BEGIN;

INSERT INTO clubs (id, name, short_name, country, is_active) VALUES
  ('a3000000-0000-4000-8000-000000000001', 'SP Burgos', 'BUR', 'Spain', true),
  ('a3000000-0000-4000-8000-000000000002', 'FC Barcelona', 'BAR', 'Spain', true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO news (
    id, club_id, title, body_html, image_url, published_at, is_active, created_at, updated_at
) VALUES
  (
    'd1000000-0000-4000-8000-000000000001',
    'a3000000-0000-4000-8000-000000000001',
    'Welcome to the 2025/26 Season',
    $html$
<h2>New season, new energy</h2>
<p>SP Burgos is ready for another exciting <strong>ACB League</strong> campaign. The Coliseum will be our fortress again — packed stands, coordinated chants, and the lights that make Friday nights feel bigger than the scoreboard.</p>
<p>Pre-season finished with two home wins and a tight away draw. The coaching staff has locked the rotation, and several academy players trained with the first team throughout August. Season tickets are on sale in the fan shop and can be paid with points in the app.</p>
<h3>What to know this week</h3>
<ul>
  <li>First home game this Friday at <strong>20:30</strong></li>
  <li>Doors open at 19:00 — fan zone from 18:30</li>
  <li>Download the app to earn points on every chant and prediction</li>
  <li>New home kit is already in stock, sizes S to XXL</li>
</ul>
<blockquote>Bring your voice. Bring your light. We are Burgos.</blockquote>
<p>See you under the lights. Let’s make the Coliseum loud from the first whistle to the last.</p>
$html$,
    'https://images.unsplash.com/photo-1546519638-68e109498ffc?auto=format&fit=crop&w=1200&h=675&q=80',
    NOW() - INTERVAL '11 days',
    true,
    NOW() - INTERVAL '11 days',
    NOW()
  ),
  (
    'd1000000-0000-4000-8000-000000000002',
    'a3000000-0000-4000-8000-000000000001',
    'Match Preview: Burgos vs Barcelona',
    $html$
<h2>Big night at the Coliseum</h2>
<p>Our squad faces <em>FC Barcelona</em> in a crucial league fixture. Barcelona arrive with a deep roster and a high pick-and-roll tempo. We will answer with physical defence, extra bodies on the glass, and a full-arena chant program from tip-off.</p>
<h3>Match information</h3>
<ul>
  <li><strong>Tip-off:</strong> 20:30</li>
  <li><strong>Doors open:</strong> 19:00</li>
  <li><strong>Venue:</strong> Coliseum Burgos</li>
  <li><strong>Broadcast:</strong> Club radio + in-app live feed</li>
</ul>
<p>Keys to the game: contest every three, crash the offensive boards, and keep turnovers under 12. If the crowd stays with the team through the third quarter — historically our toughest stretch — we like our chances late.</p>
<blockquote>Arrive early. The pre-match chant starts 20 minutes before tip-off and counts toward the leaderboard.</blockquote>
<p>Limited standing tickets remain at the north gate. Members with season cards should enter through Gate B to skip the main queue.</p>
$html$,
    'https://images.unsplash.com/photo-1504450758481-7338eba7524a?auto=format&fit=crop&w=1200&h=675&q=80',
    NOW() - INTERVAL '9 days',
    true,
    NOW() - INTERVAL '9 days',
    NOW()
  ),
  (
    'd1000000-0000-4000-8000-000000000003',
    'a3000000-0000-4000-8000-000000000001',
    'New Home Kit Now Available',
    $html$
<h2>2025/26 home jersey</h2>
<p>The official home kit is in the club shop. Lightweight breathable polyester, embroidered crest, moisture-wicking finish, and supporter sizing from S to XXL. The collar uses the same contrast tape as last season’s playoff run — a small detail our ultras asked to keep.</p>
<h3>Prices</h3>
<ul>
  <li>Adult replica: <strong>€32.50</strong> (or 3250 points)</li>
  <li>Player-fit: €62.00 — limited stock</li>
  <li>Junior sizes in selected stores this weekend</li>
  <li>Pay with card or points in the app</li>
</ul>
<p>Personalisation (name + number) takes 48 hours. Collect in store or add it to a merch order with stadium pickup on match day. Training suit and winter hoodie drop next week.</p>
<p>Wear it Friday. The first 200 supporters in the home kit at Gate A get a free scarf sticker pack.</p>
$html$,
    'https://images.unsplash.com/photo-1574629810360-7efbbe195018?auto=format&fit=crop&w=1200&h=675&q=80',
    NOW() - INTERVAL '8 days',
    true,
    NOW() - INTERVAL '8 days',
    NOW()
  ),
  (
    'd1000000-0000-4000-8000-000000000004',
    'a3000000-0000-4000-8000-000000000001',
    'Fan Zone Opens Two Hours Before Tip-off',
    $html$
<h2>Pre-match fan zone</h2>
<p>The fan zone opens <strong>two hours before tip-off</strong> at the north entrance plaza. Live DJ, food trucks, photo spots with the giant crest, and a kids’ mini-court run by the academy staff.</p>
<h3>On the plaza this Friday</h3>
<ul>
  <li>Food trucks: burgers, wraps, nachos, and churros</li>
  <li>Free face paint for under-12s until 19:30</li>
  <li>Chant rehearsal with the ultras at 19:40</li>
  <li>App check-in: +50 points when you scan the plaza QR</li>
</ul>
<p>Complete chants in the app to climb the leaderboard and unlock halftime rewards. Last season’s fan-zone regulars averaged 400 extra points per home game just from check-ins and predictions.</p>
<p>Bags larger than A4 will be checked at the plaza. Bring a refillable bottle — water fountains are next to the merch stall.</p>
$html$,
    'https://images.unsplash.com/photo-1517466787929-bc90951d0974?auto=format&fit=crop&w=1200&h=675&q=80',
    NOW() - INTERVAL '6 days',
    true,
    NOW() - INTERVAL '6 days',
    NOW()
  ),
  (
    'd1000000-0000-4000-8000-000000000005',
    'a3000000-0000-4000-8000-000000000001',
    'Player of the Month: January',
    $html$
<h2>January award</h2>
<p>Congratulations to our <strong>Player of the Month</strong>. Across five league games in January he averaged 18.4 points, 7.1 rebounds and 4.2 assists, including a 27-point night against a top-four side and two game-winning stops in the last minute.</p>
<p>Watch the highlight reel in the video feed and leave a message of support. The club will pass the best comments to the dressing room on Friday.</p>
<h3>How February voting works</h3>
<ul>
  <li>Voting opens Monday 10:00 in the app</li>
  <li>Every account gets one vote — extra votes can be unlocked with points</li>
  <li>Shortlist announced after the next home game</li>
  <li>Winner is revealed on the first of the month during the chant program</li>
</ul>
<p>Last year’s February winner went on to make the ACB weekly team twice. Your vote is not just a badge — it is part of how we tell the story of this squad.</p>
$html$,
    'https://images.pexels.com/photos/358042/pexels-photo-358042.jpeg?auto=compress&cs=tinysrgb&w=1200',
    NOW() - INTERVAL '5 days',
    true,
    NOW() - INTERVAL '5 days',
    NOW()
  ),
  (
    'd1000000-0000-4000-8000-000000000006',
    'a3000000-0000-4000-8000-000000000001',
    'Derby Tickets On Sale Friday Morning',
    $html$
<h2>Burgos derby — tickets</h2>
<p>Tickets for the provincial derby go on sale <strong>Friday at 10:00</strong> in the app and at the Coliseum box office. Season members get a 30-minute head start from 09:30.</p>
<p>This fixture sold out in under two hours last season. If you are buying for a group, use the family pack (two adults + two under-16s) — it is cheaper than four singles and includes a shared snack voucher.</p>
<h3>Price bands</h3>
<ul>
  <li>North end (ultras): €18</li>
  <li>Sideline lower: €28</li>
  <li>Sideline upper: €22</li>
  <li>Family pack: €64</li>
</ul>
<p>Away allocation is limited and already held by the visiting club. Do not buy from unofficial resale sites — those barcodes are cancelled at the turnstile.</p>
<blockquote>Have your membership QR ready. Box-office queues move faster if your profile name matches the card.</blockquote>
$html$,
    'https://images.pexels.com/photos/114296/pexels-photo-114296.jpeg?auto=compress&cs=tinysrgb&w=1200',
    NOW() - INTERVAL '4 days',
    true,
    NOW() - INTERVAL '4 days',
    NOW()
  ),
  (
    'd1000000-0000-4000-8000-000000000007',
    'a3000000-0000-4000-8000-000000000001',
    'Academy Night: Next Generation Takes the Floor',
    $html$
<h2>Academy double-header</h2>
<p>Wednesday is Academy Night. U16 and U18 play back-to-back at the Coliseum, and first-team players will sit in the stands for the second half of the U18 game. Entry is free with any first-team ticket stub from this season, or €5 at the door.</p>
<p>Several U18 names have already trained with the senior squad. A strong night in front of a loud home crowd is exactly the stage they need.</p>
<h3>Schedule</h3>
<ul>
  <li>17:30 — U16 vs Valladolid</li>
  <li>19:15 — U18 vs Valladolid</li>
  <li>Half-time U18: skills challenge judged by the first-team captain</li>
</ul>
<p>Parents can collect player-of-the-game votes in the app. The two winners receive a signed home shirt and a place in Friday’s tunnel walk.</p>
$html$,
    'https://images.pexels.com/photos/1618269/pexels-photo-1618269.jpeg?auto=compress&cs=tinysrgb&w=1200',
    NOW() - INTERVAL '3 days',
    true,
    NOW() - INTERVAL '3 days',
    NOW()
  ),
  (
    'd1000000-0000-4000-8000-000000000008',
    'a3000000-0000-4000-8000-000000000001',
    'Squad Update: Fitness Ahead of Barcelona',
    $html$
<h2>Medical and training notes</h2>
<p>The medical staff released a short update after Thursday’s session. Two rotation players completed individual rehab in the morning and joined 5-on-5 in the afternoon. No new injuries were reported.</p>
<p>The starting five is expected to be available against Barcelona. Minutes for the second unit will be managed if the game gets away late — the coaching staff prefers a tighter rotation in high-tempo fixtures.</p>
<h3>What supporters should know</h3>
<ul>
  <li>Full contact training resumed on Thursday</li>
  <li>Shootaround is closed to the public on Friday morning</li>
  <li>Warm-up playlist and first chant drop at 20:10</li>
</ul>
<p>If you see rumours on social media, wait for the club app. We publish the confirmed 12-man list three hours before tip-off.</p>
$html$,
    'https://images.unsplash.com/photo-1571019613454-1cb2f99b2d8b?auto=format&fit=crop&w=1200&h=675&q=80',
    NOW() - INTERVAL '2 days 12 hours',
    true,
    NOW() - INTERVAL '2 days 12 hours',
    NOW()
  ),
  (
    'd1000000-0000-4000-8000-000000000009',
    'a3000000-0000-4000-8000-000000000001',
    'Community Visit at Hospital Universitario',
    $html$
<h2>Players on the paediatric ward</h2>
<p>Four first-team players and the club mascot spent Tuesday morning at Hospital Universitario de Burgos. They brought mini-balls, home shirts, and a portable hoop for the playroom. Photos from the visit are in the gallery below the article in the app.</p>
<p>This is the third hospital visit of the season. The foundation also funds match tickets for families who spend long weeks on the ward — 40 seats are reserved every home Friday.</p>
<h3>How you can help</h3>
<ul>
  <li>Donate points in the app — 500 points = one family ticket</li>
  <li>Drop new (unopened) colouring books at Gate C on match day</li>
  <li>Volunteer for the next visit: sign-up form is in Profile → Foundation</li>
</ul>
<blockquote>The kids asked for a win on Friday. That is the only briefing the players needed.</blockquote>
$html$,
    'https://images.unsplash.com/photo-1526232761682-d26e03ac148e?auto=format&fit=crop&w=1200&h=675&q=80',
    NOW() - INTERVAL '2 days',
    true,
    NOW() - INTERVAL '2 days',
    NOW()
  ),
  (
    'd1000000-0000-4000-8000-000000000010',
    'a3000000-0000-4000-8000-000000000001',
    'Chant of the Week: We Are Burgos',
    $html$
<h2>This week’s synced chant</h2>
<p><strong>We Are Burgos</strong> is the featured chant for Friday. It runs 75 seconds, starts 20 minutes before tip-off, and again after every home three in the first quarter if the arena is still above 80% occupancy on the live map.</p>
<p>Flash and vibration are timed to the chorus. Keep your phone unmuted and the app in the foreground — backgrounded devices miss the first downbeat and the points that come with it.</p>
<h3>How to score</h3>
<ul>
  <li>Join in the first 3 seconds: full points</li>
  <li>Join late: half points</li>
  <li>Stay until the last lyric: bonus 25 points</li>
  <li>Share a clip from the stands: extra 10 points after moderation</li>
</ul>
<p>Preview the lyrics in the Chants tab. If you are bringing someone new, sit them on the north side — that block always hits the timing window first.</p>
$html$,
    'https://images.unsplash.com/photo-1514525253161-7a46d19cd819?auto=format&fit=crop&w=1200&h=675&q=80',
    NOW() - INTERVAL '1 day 8 hours',
    true,
    NOW() - INTERVAL '1 day 8 hours',
    NOW()
  ),
  (
    'd1000000-0000-4000-8000-000000000011',
    'a3000000-0000-4000-8000-000000000001',
    'Match-Day Menu: New Stadium Food This Friday',
    $html$
<h2>New at Stadium Snacks</h2>
<p>The kiosks have a refreshed menu for the Barcelona game. Double Burger and BBQ Wings stay, and we add a grilled chicken wrap, stone-baked veggie pizza slice, and iced coffee for the late tip-off.</p>
<p>Pay with points or card in the app and skip the cash queue. Orders placed before 20:15 are ready at the north kiosk so you do not miss the opening chant.</p>
<h3>Friday specials</h3>
<ul>
  <li>Double Burger + Cola Zero: 1.00€ off</li>
  <li>Loaded Nachos to share: extra cheese at no charge until 19:45</li>
  <li>Chocolate muffin with any coffee</li>
  <li>Kids: popcorn + juice combo</li>
</ul>
<p>Allergen cards are on the counter and in the product screen. If you have a nut or gluten allergy, ask staff before you confirm the order — the wrap station shares a grill with the burger line.</p>
$html$,
    'https://images.unsplash.com/photo-1555939594-58d7cb561ad1?auto=format&fit=crop&w=1200&h=675&q=80',
    NOW() - INTERVAL '18 hours',
    true,
    NOW() - INTERVAL '18 hours',
    NOW()
  ),
  (
    'd1000000-0000-4000-8000-000000000012',
    'a3000000-0000-4000-8000-000000000001',
    'Season Membership: More Points, More Perks',
    $html$
<h2>Upgrade before the derby</h2>
<p>Season membership is still open. Members skip the Friday 09:30 ticket hold, earn <strong>1.5× points</strong> on chants, and get a 10% merch discount in the app shop.</p>
<p>Gold members also receive a reserved seat in the lower sideline for every home league game and a guest pass once a month. That guest pass is the easiest way to bring someone new into the Coliseum without buying a full season card.</p>
<h3>What you get</h3>
<ul>
  <li>Early ticket access for sold-out fixtures</li>
  <li>1.5× chant and prediction points</li>
  <li>10% off merch and 5% off stadium food</li>
  <li>Gold: reserved seat + monthly guest pass</li>
</ul>
<p>Upgrade in Profile → Membership. Existing half-season cards can be converted; the difference is charged pro-rata through the end of June. Questions: membership@spburgos.local or the fan-shop desk from 16:00 on match days.</p>
$html$,
    'https://images.unsplash.com/photo-1577223625816-7546f13df25d?auto=format&fit=crop&w=1200&h=675&q=80',
    NOW() - INTERVAL '6 hours',
    true,
    NOW() - INTERVAL '6 hours',
    NOW()
  )
ON CONFLICT (id) DO UPDATE SET
    title        = EXCLUDED.title,
    body_html    = EXCLUDED.body_html,
    image_url    = EXCLUDED.image_url,
    published_at = EXCLUDED.published_at,
    is_active    = true,
    deleted_at   = NULL,
    updated_at   = NOW();

COMMIT;
