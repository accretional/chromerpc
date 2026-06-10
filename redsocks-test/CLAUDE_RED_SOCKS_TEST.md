# Red Socks — agent-steered interactive session test

**Goal:** From a *single* live interactive browser session on the remote
`chromerpc-bidi-pool` Cloud Run service, I (the agent) navigate Amazon
iteratively — search "silly socks", look at results, scroll/select toward
**red** socks, open a product page, and push toward buying until a paywall/login
wall stops me. I take a screenshot after every move and reason about it before
the next.

**How it's driven:** a long-lived `chrome-proxy` process holds the bidi
`Session` stream open; I issue one `POST /steps` per move (a separate shell tool
call), it runs the steps on the live session and saves any screenshot to
`shots/`, which I then view and reason about. The session/Chrome state persists
across all my moves.

- Service: `chromerpc-bidi-pool` (pool-size 2), session `isess-1`
- Proxy: `http://127.0.0.1:8099`, screenshots in `redsocks-test/shots/`

---

## Log

### Move 1 — load Amazon home (`shots/0001-step.png`)
- **Did:** set 1280×900 viewport, `navigate https://www.amazon.com` (networkidle), screenshot.
- **Observed:** Standard Amazon homepage rendered fully — search bar at top, pharmacy banner, "Blue hues / Luxury Father's Day gifts / New summer styles / Deals on tech" cards. **No CAPTCHA/bot wall** — the headless session is being served normal content.
- **Next:** type "silly socks" into the search box (`#twotabsearchtextbox`) and submit by clicking the search button (a synthetic Enter often doesn't submit Amazon's form, as found earlier).

### Move 2 — search "silly socks" (`shots/0002-step.png`)
- **Did:** typed into `#twotabsearchtextbox`, clicked `#nav-search-submit-button` (returned `"clicked"`), waited for `[data-component-type=s-search-result]`, screenshot.
- **Observed:** "1-48 of over 4,000 results for silly socks". Top row: Bioworld orange-striped crew, COZYMATE animal socks (pink/white), Wraharbi ketchup-&-fries pattern, HSELL patterned dress socks. Left rail has Department/Gender/Price filters. **None are clearly red** — they're orange/multicolor/novelty patterns.
- **Thinking:** I need mostly-red socks. I'll scroll the results to scan more products for a predominantly-red pair rather than refine the query (per the task: scroll the "silly socks" results toward red ones).
- **Next:** scroll down ~900px and screenshot.

### Move 3 — scroll results (`shots/0003-step.png`)
- **Did:** `scrollBy(0,950)`, screenshot. scrollY=950.
- **Observed:** More novelty socks (Subety green, DJleed yellow chicken, LeftJett blue, cow-print, etc.) and a "More results" grid. Left rail revealed a **Color filter (swatches)** plus Theme/Material. Still nothing predominantly red.
- **Thinking:** I can keep scrolling, but a faster path to *red* socks is the left-rail Color filter. I'll scroll once more to scan; if still no red, I'll click the red color swatch (a legit "select something" decision).
- **Next:** scroll another ~950px, screenshot.

### Move 4 — scroll again (`shots/0004-step.png`)
- **Did:** `scrollBy(0,950)` (scrollY=1900), screenshot.
- **Observed:** Subety Christmas socks, Hyp SpongeBob, WHOSAY orange, MITUN multicolor. Left rail down to Lifestyle/Occasion. Still no predominantly-red pair across the first ~12 results.
- **Decision:** Plain "silly socks" is dominated by multicolor novelty prints; scrolling isn't surfacing red. I'll *refine the query* to "red silly socks" (the task allows "however you want to navigate") — stays on-theme and reliably yields red pairs.
- **Next:** set the search box to "red silly socks", submit, screenshot.

### Move 5 — refined search "red silly socks" (`shots/0005-step.png`)
- **Did:** set search box to "red silly socks", submitted, scrolled to top, screenshot. Confirmed search box value = "red silly socks", "1-48 of 1,000 results".
- **Observed:** Results are now **predominantly red** — OurSuperDeals red character/crew socks, BiliaSakti crazy red kids' socks, Jeqipotix "Funny Magnetic Holding Hands" red socks, OWLZHKBA red socks. This satisfies "mostly red ones".
- **Decision:** Pick the first product (top-left) and open its product page to attempt a purchase.
- **Next:** extract the first result's `/dp/` link + title, then navigate to it.

### Move 6 — session dropped + reconnect (keepalive fix)
- **Problem:** after move 5, the next call returned `send: EOF` — the bidi stream had idle-closed during my between-move thinking gaps (no keepalive). Server log showed the session closing and a (throttled) recycle. **Not a navigation bug.**
- **Fix:** added gRPC keepalive (`Time=30s, PermitWithoutStream`) to `chrome-proxy` so the long-lived stream survives idle gaps. Rebuilt, reconnected as **`isess-2`**, re-navigated directly to `amazon.com/s?k=red+silly+socks`.
- **Note:** shot counter reset on restart, so numbering restarts at `0001` (now the red results page). Continuing from here.
- **Observed (`shots/0001-step.png`):** red pairs — OurSuperDeals "Holding Hands" kids' crew, BiliaSakti crazy red, Jamntower red monster, Kaytlyy red cartoon-monster.
- **Next:** open the first product (OurSuperDeals red holding-hands socks).

### Move 7 — open product page (`shots/0002-step.png`)
- **Did:** found 413 `/dp/` links; first = OurSuperDeals ASIN `B0D8QBXVW3` ("Holding Hands" red socks). Navigated to `amazon.com/dp/B0D8QBXVW3`, screenshot.
- **Goal from here:** view the product, then attempt to buy (Add to Cart / Buy Now) and push toward checkout until a login/paywall stops me.

### Move 8 — product page; click "Buy Now" (`shots/0002` product, `0003` after click)
- **Observed (`shots/0002-step.png`):** Product = "Holding Hands Socks for Kids Toddler Cute Crazy Silly Funny Novelty Magnetic Tube Cotton Socks 4-11 Years", **red** socks with faces, $9.xx, In Stock, Color "Red 1 Pair", buttons **Add to cart** + **Buy Now**.
- **Decision:** click **Buy Now** (`#buy-now-button`) — most direct route to checkout, where Amazon should require sign-in (the paywall).
- **Next:** click Buy Now, wait, screenshot.

### Move 9 — PAYWALL reached (`shots/0003-step.png`)
- **Observed:** "Sign in or create account — Enter mobile number or email / Continue". URL = `https://www.amazon.com/ap/signin?openid.pape.max_auth_age=900&openid.return_to=...`, title "Amazon Sign-In", `signin=true`.
- **Conclusion:** This is the auth **paywall** — Amazon requires sign-in before completing a purchase. Without credentials this is the furthest the flow goes. **Goal achieved.**

## Outcome

Drove a single live remote browser session end-to-end, steering by screenshot at
each step:

1. Amazon home → 2. search "silly socks" → 3–4. scrolled results (multicolor
novelty, no red) → 5. refined to **"red silly socks"** (found red pairs) →
6. (session idle-dropped; added keepalive, reconnected) → 7. opened product
**OurSuperDeals red "Holding Hands" kids' socks** (`/dp/B0D8QBXVW3`) →
8. clicked **Buy Now** → 9. hit the **Amazon Sign-In paywall**.

Screenshots (`shots/`): `0001` red results · `0002` product page · `0003`
sign-in paywall. (`0004`/`0005` are leftovers from the pre-reconnect session.)

**Key learnings**
- A long-lived interactive bidi session needs **keepalive** or it idle-closes
  during operator/agent think-time. `chrome-proxy` now sets gRPC keepalive.
- The remote session **held state** (search → scroll → product → checkout) across
  many independent shell calls — exactly the interactive model the bidi service
  is for.
- Headless Chrome on the service got **normal Amazon content** (no CAPTCHA),
  including search, product pages, and the real checkout/sign-in flow.
