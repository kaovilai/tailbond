# Round 10 — Android persona C (family member)

**Who I am:** someone else in the family set up the home server ("homebox") and put the Tailscale app on my phone. I use the phone for messages, photos and video. I've never installed an app from outside the Play Store. All I was told: "tailbond bonds your phone's Wi-Fi and 5G together to homebox. Type **homebox/** into your phone's browser with Tailscale on."

**What I looked at:** `task-script.md`, `screens.md`, every PNG in `web/` and `android/` (I cut the tall ones into pieces so I could read them), plus the dark-theme versions of the home, handoff and Connections screens. I didn't open anything else.

---

## Task 1: Get the tailbond app onto the phone

**Where I tap:** with Tailscale on, I type `homebox/` into Chrome and get **web-00a**: "Get tailbond on this phone", then "1 Install the app" with a big green **Download tailbond (27 MB)** button. Under it, **web-00** has five drawn pictures of Android's prompts, each with the button to press circled in orange:
① "File might be harmful … Download anyway", ② "For your security … Settings", ③ "Install unknown apps · Chrome · Allow from this source" (toggle), ④ "Do you want to install this app? Install", ⑤ "App scan recommended … Scan app". Then: "When it says App installed, tap Done and come back to this page (Chrome, this tab) for step 2."

**What went well:** the page tells me straight out that "tailbond isn't in the Play Store: it comes from homebox, your own home server, so Android asks you to confirm a few times. That's Android's normal check …, and you only go through it once." It also says "the pictures below are drawings, not buttons", and it covers the case I'd worry about most: "Chrome's Open message gone before you tapped it? … Files app → Downloads. Tap tailbond.apk." It even says "Stuck, or not comfortable with this? Ask whoever runs homebox to install it for you." If I tap step 2 before installing, **web-00b** sends me back to step 1 with "tailbond isn't on this phone yet, so step 2 brought you back here. Install it first, then tap step 2 again." That's very kind.

**Where I'd get stuck or hesitate:**
- The tip box says "Read the pictures below first (or take a screenshot): once you open the file, Android's own screens cover this page." So I have to remember five different pop-ups, two of them scary ("File might be harmful", "isn't allowed to install unknown apps"), and get through them with no help on screen. I don't know how to take a screenshot reliably. The Download button sits *above* the pictures, so my thumb will probably tap it before I've read them. I'd likely get through, but nervously, and I might tap "Cancel" on "File might be harmful" because that's what I've always been told to do. **Medium.**
- The "Is this safe?" paragraph has "SHA-256 54167117c3586c8…" and "sha256sum /usr/share/tailbond/tailbond.apk on homebox". That means nothing to me. It's grey and small, so I'd skip it. **Low.**
- There are two sets of numbers: big steps **1 2 3** and small ①–⑤ inside step 1. Later the app tells me to "tap step 2 there", and I might look at ② ("Android asks once about your browser") instead of the big step 2 button. **Low.**

**What I expected:** in the Play Store you tap Install and that's it. Nothing here could make it that simple, and the page warns me well about everything Android will do. The only hard part is having to remember it all.

---

## Task 2: Add the phone to homebox (pair), from the phone

**Where I tap:** back in Chrome, step 2's **Open tailbond and ask homebox** button. The app opens on **02a** ("Asking homebox… Looking for homebox on your Tailscale network.") and then **02b**: "Waiting for approval on homebox… Ask whoever runs homebox to approve "pixel-8" … There it shows as pixel-8 · sam@example.com · 100.101.8.8 … You can switch to another app to message them; tailbond keeps waiting for 10 minutes and carries on by itself once approved." I'd message whoever set it up. Once they approve, **02e** asks "Add this phone to homebox?" and I tap **Add phone** (the web page told me to: "Once approved, tailbond shows what it will do: tap Add phone").

**What went well:** 02b tells me exactly what to say to whoever set it up, and that I can leave the app. The errors are in plain words: **02c** "Can't find homebox — Check that the Tailscale app says Connected …" and **02d** "homebox said no — Whoever runs homebox declined this phone. Ask them, then ask again."

**Where I'd get stuck or hesitate:**
- If I tapped **Open** on Android's "App installed" screen instead of Done, which is the natural thing to do, I land on **01-onboarding** with a big **Get started** button. The hint I actually need ("Installed tailbond from your home server's page in Chrome? Easier: go back to Chrome and tap step 2 there") is small text *below* that button. I'd tap Get started and end up on **02-pairing**, where typing "homebox" and tapping **Ask to connect** still works, so I'm not stuck. It's just a longer way round. **Low.**
- The confirm dialog **02e** says "they see this phone as homebox, so homebox's Tailscale access rules apply (not your own account's)" and "It connects over the internet to home.example.net:443". I don't know what "access rules" means here, and "home.example.net" isn't a name anyone told me. Is that really our homebox? I'd tap Add phone because the web page said to, but I'd wonder. **Low.**

---

## Task 3: Connect

**Where I tap:** on the home screen (**12-home-disconnected**: "Disconnected · Stopped. Turn on to reach homebox and your Tailscale machines, or switch back to Tailscale below.") I tap the big switch. Because Tailscale is on, **04-handoff-other-vpn** appears: "Switch from Tailscale to tailbond?" → **Turn on tailbond** → Android's own "OK" (not pictured, but the dialog warns me: "Next, Android asks you to allow tailbond's connection request. Tap OK."). Then **11** "Connecting… Reaching homebox…" and **10** "Connected · Through homebox · Wi-Fi + mobile data".

**What went well:** it warns me that Tailscale will go off and promises "You still reach your Tailscale machines, like your NAS, through homebox", which matches what I was told. It also tells me ahead of time how to undo it: "To go back later, tap Switch back to Tailscale on tailbond's main screen."

**Where I'd get stuck or hesitate:**
- The dialog is a wall of five paragraphs, and some of it is words I don't know: "Your internet traffic goes out from homebox too (like an exit node; Settings changes that)" and "If tailbond switches itself off right after this, Tailscale is set as Always-on VPN on this phone: turn that off in Android Settings → VPN → Tailscale's gear". I don't know what an exit node is, so I don't realise all my browsing now goes through the house. The Always-on part at least ends with "(or ask whoever set up the phone)". **Low.**

---

## Task 4: Find and copy the NAS's MagicDNS name or Tailscale IP

**Where I tap:** on the connected home screen I scroll down to **Machines**, type "nas" into **Search machines** (**18**), and tap the copy icon on the "nas" row. The message says "Copied 100.101.2.2". Or I tap the row and get **19-machine-detail**: "Tap an address to copy it" with **MagicDNS name** nas.tail1234.ts.net, **Short name**, **Tailscale IPv4** 100.101.2.2 and **Tailscale IPv6**, each with its own copy icon. The message says "Copied nas.tail1234.ts.net". Either way the task is done.

**What went well:** the detail page labels every address and says "With tailbond on, apps on this phone can use these addresses just like with Tailscale." The NAS shows a green "Online" dot.

**Where I'd get stuck or hesitate:**
- In the Tailscale app the machine list is right there. Here I have to scroll past Connections, Save mobile data and **This device** first. That section shows "10.77.0.2" with a long note ("Not a Tailscale address: to your Tailscale machines this phone's traffic comes from homebox, under its access rules …") and the **first copy icon on the screen**, and it's for my own phone, not the NAS. If I'm hurrying I might copy that one. **Low.**
- On the list row the name line ("nas.tail1234.ts.net") is bigger than the IP line, but the copy icon copies the IP. I only find out which one I got from the "Copied 100.101.2.2" message. That's fine for this task, but I'd have guessed it copied the name. **Low.**

---

## Task 5: Tell whether Wi-Fi and 5G are both being used right now

**Where I look:** the home screen (**10**). The top card says "Connected · Through homebox · Wi-Fi + mobile data", and under **Connections** it says "Using both Wi-Fi and mobile data", with two tiles, **Wi-Fi ● In use** and **Mobile data ● In use**, both with green dots. **20-connections** says the same, with graphs.

**Result:** yes, both are in use, and I could tell in a second. It says "mobile data" rather than "5G", but I know that's the same thing. When one of them breaks, **15** says "Using Wi-Fi only" with a red dot and "Not working right now" on Mobile data, which is just as clear. **No problems.**

---

## Task 6: Make it save mobile data

**Where I tap:** the **Save mobile data** switch on the home screen ("Use mobile data only when Wi-Fi stops working"). **16** then shows "Using Wi-Fi · saving mobile data (backup only)", "Mobile data · Not in use · saving data", and "Mode: Save mobile data". When Wi-Fi drops (**17**) it says "Wi-Fi isn't working · using mobile data" and "In use · taking over", so I know when my data is being used.

**What went well:** it's one switch on the main screen, with a plain-English line under it.

**Where I'd get stuck or hesitate:**
- If I go into **More options** (**20/21**), there's "Save mobile data" again as one of four choices, *plus* a separate **Mobile data limit** ("Monthly limit in GB (empty = no limit)" … "Separate from Save mobile data"), *plus* "Faster switch to backup (experimental) — Keeps mobile data warmed up so switching from Wi-Fi is quicker. Uses a little mobile data." Three data-saving things on one page made me unsure which one I'd actually turned on. The page does say "The Save mobile data switch on the home screen picks "Save mobile data" here", which helps. **Low.**

---

## Task 7: Hotel Wi-Fi with a sign-in page; get it working and back into the bond

**Where I tap:** the home screen shows a card (**16c**): "Wi-Fi needs sign-in — This Wi-Fi shows a sign-in page before it lets you online. tailbond keeps you connected over mobile data meanwhile, and adds Wi-Fi by itself once you're signed in." Then a big **Sign in to Wi-Fi** button. Further down: "Wi-Fi needs sign-in · using mobile data", and the Wi-Fi tile says "Needs sign-in · not used yet".

**What went well:** it explains what's going on, that I'm still connected, and that I *don't* need to turn tailbond off ("Android's sign-in page uses the Wi-Fi directly, so you don't need to turn tailbond off"). That's exactly the worry I'd have. It also says Wi-Fi comes back "by itself", so I don't have to do anything afterwards. (There's no screenshot of after I've signed in, so I can't check that part.)

**Where I'd get stuck or hesitate:**
- The small grey text *above* the button says "Tap the Wi-Fi network (it says "Sign in to network") to open the page." It describes a screen I haven't seen yet, and it doesn't say whether the blue button opens the hotel page itself or a list of Wi-Fi networks. I'd tap the button, and if a Wi-Fi list opened, I'd have to remember to look for the line saying "Sign in to network". **Low.**

---

## Task 8: Turn tailbond off and go back to the Tailscale app

**Where I tap:** **Switch back to Tailscale**. It's a full-width button under the status card on every home screen (10, 11, 13–17). **12b** then says "Disconnected · Stopped. Back to Tailscale: turn it on in its app.", and a card says "Last step: turn Tailscale on — tailbond is off. Open the Tailscale app and check that it says Connected; if it says Disconnected, tap its switch. Until then, your Tailscale machines aren't reachable." with an **Open Tailscale** button.

**What went well:** the button is always in the same place, and its name says exactly what it does. The "Last step" card warns me my NAS won't work until Tailscale is back on. If I just flip the big switch off instead, **12** says "or switch back to Tailscale below", so I can't get lost.

**Where I'd get stuck or hesitate:**
- In **30-settings**, the same option says "Turns tailbond off and opens the Tailscale app." But 12b shows me still inside tailbond with an **Open Tailscale** button. So did Tailscale open, or do I need to tap that? I'd tap Open Tailscale to be sure, which does no harm. **Low.**

---

## Across all tasks

- "tailnet" shows up in places a family member would read: "Still on your tailnet" (**01**), "This phone reaches your tailnet through homebox" (**11/12**), "only your tailnet goes through home" (**30**). Nobody ever told me that word. Most of the app says "Tailscale machines", which I do understand. **Low.**
- Dark theme (10, 04, 20 dark): everything was readable. No problems.

---

## Findings

| # | Task | Finding | Severity |
|---|---|---|---|
| 1 | 1 Install | "Read the pictures below first (or take a screenshot): once you open the file, Android's own screens cover this page." I have to memorise 5 Android pop-ups (incl. "File might be harmful → Download anyway" and turning on "Allow from this source") with no help on screen. The Download button sits above the pictures. A first-time sideloader may tap Cancel on the scary one. | medium |
| 2 | 1 Install | The "Is this safe?" paragraph with "SHA-256 5416…" and "sha256sum /usr/share/tailbond/tailbond.apk" is gibberish to a non-technical user (it's grey and skippable). | low |
| 3 | 1/2 | Two numbering schemes (big steps 1–3 and ①–⑤ inside step 1). "tap step 2" can be read as ② "Android asks once about your browser". | low |
| 4 | 2 Pair | If I tap Open (not Done) after installing, 01-onboarding's big "Get started" leads to the type-the-name route. "Easier: go back to Chrome and tap step 2 there" is small text below the button. Typing "homebox" still works. | low |
| 5 | 2 Pair | 02e dialog: "homebox's Tailscale access rules apply (not your own account's)" and "home.example.net:443", an address nobody told me about. Makes me hesitate before tapping Add phone. | low |
| 6 | 3 Connect | 04 dialog is five paragraphs, with "like an exit node" and "Always-on VPN … Tailscale's gear". Unclear to me that all my browsing now goes via home. | low |
| 7 | 4 NAS | Machines list is below the fold. The first copy icon I meet is next to my own phone's "10.77.0.2" ("Not a Tailscale address: …"), not the NAS. | low |
| 8 | 4 NAS | List-row copy icon copies the IP ("Copied 100.101.2.2") even though the name line is shown bigger. I only learn which one from the message. | low |
| 9 | 6 Save data | More options page has the "Save mobile data" mode, a separate "Mobile data limit" ("Separate from Save mobile data") and "Faster switch to backup (experimental) … warmed up". Three data-saving controls on one page. | low |
| 10 | 7 Hotel Wi-Fi | "Tap the Wi-Fi network (it says "Sign in to network") to open the page" sits above the "Sign in to Wi-Fi" button and describes a screen I haven't seen. Unclear whether the button opens the page itself or the Wi-Fi list. | low |
| 11 | 8 Back to Tailscale | Settings says Switch back "opens the Tailscale app", but 12b shows me still in tailbond with an "Open Tailscale" button. Did it open or not? | low |
| 12 | All | The word "tailnet" ("Still on your tailnet", "reaches your tailnet through homebox", "only your tailnet goes through home") is never explained. Elsewhere the app says "Tailscale machines". | low |

Totals: 0 high / 1 medium / 11 low
