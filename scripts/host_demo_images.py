#!/usr/bin/env python3
"""Download demo shop/news photos into clap_api local storage."""
import subprocess
import sys

UA = "Mozilla/5.0 (compatible; SmartKlapDemo/1.0)"
IMAGES = {
    "food/double-burger.jpg": "https://images.unsplash.com/photo-1568901346375-23c9450c58cd?auto=format&fit=crop&w=800&h=800&q=80",
    "food/club-hot-dog.jpg": "https://images.pexels.com/photos/4518656/pexels-photo-4518656.jpeg?auto=compress&cs=tinysrgb&w=800",
    "food/loaded-nachos.jpg": "https://images.unsplash.com/photo-1513456852971-30c0b8199d4d?auto=format&fit=crop&w=800&h=800&q=80",
    "food/salted-popcorn.jpg": "https://images.pexels.com/photos/33129/popcorn-movie-party-entertainment.jpg?auto=compress&cs=tinysrgb&w=800",
    "food/cola-zero.jpg": "https://images.pexels.com/photos/50593/coca-cola-cold-drink-soft-drink-coke-50593.jpeg?auto=compress&cs=tinysrgb&w=800",
    "food/mineral-water.jpg": "https://images.unsplash.com/photo-1548839140-29a749e1cf4d?auto=format&fit=crop&w=800&h=800&q=80",
    "food/chicken-wrap.jpg": "https://images.unsplash.com/photo-1626700051175-6818013e1d4f?auto=format&fit=crop&w=800&h=800&q=80",
    "food/veggie-pizza.jpg": "https://images.unsplash.com/photo-1565299624946-b28f40a0ae38?auto=format&fit=crop&w=800&h=800&q=80",
    "food/french-fries.jpg": "https://images.unsplash.com/photo-1573080496219-bb080dd4f877?auto=format&fit=crop&w=800&h=800&q=80",
    "food/energy-drink.jpg": "https://images.unsplash.com/photo-1622543925917-763c34d1a86e?auto=format&fit=crop&w=800&h=800&q=80",
    "food/orange-juice.jpg": "https://images.unsplash.com/photo-1600271886742-f049cd451bba?auto=format&fit=crop&w=800&h=800&q=80",
    "food/club-sandwich.jpg": "https://images.unsplash.com/photo-1528735602780-2552fd46c7af?auto=format&fit=crop&w=800&h=800&q=80",
    "food/bbq-wings.jpg": "https://images.unsplash.com/photo-1527477396000-e27163b481c2?auto=format&fit=crop&w=800&h=800&q=80",
    "food/chocolate-muffin.jpg": "https://images.unsplash.com/photo-1607958996333-41aef7caefaa?auto=format&fit=crop&w=800&h=800&q=80",
    "food/iced-coffee.jpg": "https://images.unsplash.com/photo-1461023058943-07fcbe16d735?auto=format&fit=crop&w=800&h=800&q=80",
    "food/pretzel.jpg": "https://images.unsplash.com/photo-1555507036-ab1f4038808a?auto=format&fit=crop&w=800&h=800&q=80",
    "food/fish-chips.jpg": "https://images.pexels.com/photos/566345/pexels-photo-566345.jpeg?auto=compress&cs=tinysrgb&w=800",
    "food/sparkling-water.jpg": "https://images.unsplash.com/photo-1523362628745-0c100150b504?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/sport-tshirt.jpg": "https://images.unsplash.com/photo-1576566588028-4147f3842f27?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/away-tshirt.jpg": "https://images.unsplash.com/photo-1522778119026-d647f0596c20?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/match-ball.jpg": "https://images.pexels.com/photos/46798/the-ball-stadion-football-the-pitch-46798.jpeg?auto=compress&cs=tinysrgb&w=800",
    "merch/sticker-pack.jpg": "https://images.pexels.com/photos/4226806/pexels-photo-4226806.jpeg?auto=compress&cs=tinysrgb&w=800",
    "merch/training-suit.jpg": "https://images.unsplash.com/photo-1556821840-3a63f95609a7?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/winter-hoodie.jpg": "https://images.unsplash.com/photo-1509942774463-acf339cf87d5?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/mini-ball.jpg": "https://images.unsplash.com/photo-1575361204480-aadea25e6e68?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/scarf.jpg": "https://images.unsplash.com/photo-1483721310020-03333e577078?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/gk-gloves.jpg": "https://images.unsplash.com/photo-1579952363873-27f3bade9f55?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/captain-armband.jpg": "https://images.unsplash.com/photo-1574629810360-7efbbe195018?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/stadium-cap.jpg": "https://images.unsplash.com/photo-1521369909029-2afed882baee?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/fan-flag.jpg": "https://images.unsplash.com/photo-1577223625816-7546f13df25d?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/training-shorts.jpg": "https://images.unsplash.com/photo-1562183241-b937e95585b6?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/socks-pack.jpg": "https://images.unsplash.com/photo-1586350977771-b3b0abd50c82?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/water-bottle.jpg": "https://images.unsplash.com/photo-1523362628745-0c100150b504?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/retro-jersey.jpg": "https://images.unsplash.com/photo-1517466787929-bc90951d0974?auto=format&fit=crop&w=800&h=800&q=80",
    "merch/pump-ball.jpg": "https://images.unsplash.com/photo-1431324155629-1a6deb1dec8d?auto=format&fit=crop&w=800&h=800&q=80",
    "news/welcome-season.jpg": "https://images.unsplash.com/photo-1546519638-68e109498ffc?auto=format&fit=crop&w=1200&h=675&q=80",
    "news/match-preview.jpg": "https://images.unsplash.com/photo-1504450758481-7338eba7524a?auto=format&fit=crop&w=1200&h=675&q=80",
    "news/home-kit.jpg": "https://images.unsplash.com/photo-1574629810360-7efbbe195018?auto=format&fit=crop&w=1200&h=675&q=80",
    "news/fan-zone.jpg": "https://images.unsplash.com/photo-1517466787929-bc90951d0974?auto=format&fit=crop&w=1200&h=675&q=80",
    "news/player-month.jpg": "https://images.pexels.com/photos/358042/pexels-photo-358042.jpeg?auto=compress&cs=tinysrgb&w=1200",
    "news/derby-tickets.jpg": "https://images.pexels.com/photos/114296/pexels-photo-114296.jpeg?auto=compress&cs=tinysrgb&w=1200",
    "news/academy-night.jpg": "https://images.pexels.com/photos/1618269/pexels-photo-1618269.jpeg?auto=compress&cs=tinysrgb&w=1200",
    "news/squad-update.jpg": "https://images.unsplash.com/photo-1571019613454-1cb2f99b2d8b?auto=format&fit=crop&w=1200&h=675&q=80",
    "news/community.jpg": "https://images.unsplash.com/photo-1526232761682-d26e03ac148e?auto=format&fit=crop&w=1200&h=675&q=80",
    "news/chant-week.jpg": "https://images.unsplash.com/photo-1514525253161-7a46d19cd819?auto=format&fit=crop&w=1200&h=675&q=80",
    "news/stadium-food.jpg": "https://images.unsplash.com/photo-1555939594-58d7cb561ad1?auto=format&fit=crop&w=1200&h=675&q=80",
    "news/membership.jpg": "https://images.unsplash.com/photo-1577223625816-7546f13df25d?auto=format&fit=crop&w=1200&h=675&q=80",
}


def run(args):
    return subprocess.run(args, capture_output=True, text=True)


def main():
    run(["docker", "exec", "clap_api", "mkdir", "-p",
         "/data/uploads/demo/food", "/data/uploads/demo/merch", "/data/uploads/demo/news"])
    failed = []
    for rel, url in IMAGES.items():
        dest = f"/data/uploads/demo/{rel}"
        print(f"GET {rel}")
        r = run(["docker", "exec", "clap_api", "wget", "-q", "-T", "30", "-U", UA, "-O", dest, url])
        if r.returncode != 0:
            print(f"  FAIL wget: {(r.stderr or r.stdout or '').strip()[:200]}")
            failed.append(rel)
            continue
        size = run(["docker", "exec", "clap_api", "wc", "-c", dest])
        try:
            n = int(size.stdout.strip().split()[0])
        except Exception:
            n = 0
        if n < 4000:
            print(f"  FAIL size={n}")
            failed.append(rel)
            continue
        print(f"  OK {n} bytes")
    if failed:
        print("FAILED:", ", ".join(failed))
        sys.exit(1)
    print("All downloads ok.")


if __name__ == "__main__":
    main()
