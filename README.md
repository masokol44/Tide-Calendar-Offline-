🌊 Tide Calendar (Offline) — Multi‑Language Offline Tide Predictor
8 languages, one reliable offline tide calculator – predict tides without internet using harmonic analysis – right from your terminal.

✨ Features
🌊 Offline prediction – uses harmonic constituents (M2, S2, N2, K1, O1)

📅 Any date/time – forecast for today or any past/future date

📊 ASCII tide graph – visualize the tide curve in your terminal

📋 Table output – high/low tide times and heights

💾 Persistent config – save location phase offsets for your area

⚡ No API calls – fully self‑contained, works anywhere

🚀 Quick Start
All implementations share the same CLI pattern:

bash
# Show today's tide (24‑hour forecast)
<command>

# Forecast for a specific date
<command> --date 2026-08-25

# Forecast for 48 hours
<command> --hours 48

# Show only the ASCII graph
<command> --graph

# Use a custom configuration file with harmonic constituents
<command> --config my_location.json

# Save current settings as a default config
<command> --save-config
Arguments:

--date YYYY-MM-DD – start date (default: today)

--hours N – forecast duration in hours (default: 24)

--graph – show ASCII graph instead of table

--list – show high/low tides (default)

--config FILE – load harmonic constituents from JSON

--save-config – save current parameters to default config

📸 Example Output
text
🌊 Offline Tide Calendar
Date: 2026-08-25 00:00 – 24h forecast

High/Low Tides:
2026-08-25 02:15  High  1.8 m
2026-08-25 08:45  Low   0.2 m
2026-08-25 14:30  High  1.7 m
2026-08-25 20:55  Low   0.3 m

Level (m)
 2.0 |                                            *
 1.5 |       *        *           *        *       *
 1.0 |    *     *  *     *     *     *  *     *    *
 0.5 | *          *          *          *          *
 0.0 |*          *          *          *          *
-0.5 |          *          *          *          *
-1.0 |     *     *     *     *     *     *     *
-1.5 |  *        *        *        *        *
-2.0 |*          *          *          *
      ----------------------------------------------
      00:00  03:00  06:00  09:00  12:00  15:00  18:00  21:00  24:00
📁 Repository Structure
text
.
├── README.md
├── python/
│   └── tide_offline.py
├── go/
│   └── tide_offline.go
├── javascript/
│   └── tide_offline.js
├── ruby/
│   └── tide_offline.rb
├── php/
│   └── tide_offline.php
├── java/
│   └── TideOffline.java
├── csharp/
│   └── TideOffline.cs
└── cpp/
    └── tide_offline.cpp
