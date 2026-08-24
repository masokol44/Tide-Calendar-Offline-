# tide_offline.py
import sys
import math
import json
import argparse
from datetime import datetime, timedelta, timezone

# Default harmonic constituents: [(amplitude_m, period_hours, phase_deg)]
DEFAULT_CONSTITUENTS = [
    (1.2, 12.42, 0.0),   # M2
    (0.4, 12.00, 0.0),   # S2
    (0.2, 12.66, 0.0),   # N2
    (0.1, 23.93, 0.0),   # K1
    (0.08, 25.82, 0.0),  # O1
]
MEAN_LEVEL = 0.5  # meters

class TideModel:
    def __init__(self, constituents=None, mean_level=MEAN_LEVEL):
        self.constituents = constituents or DEFAULT_CONSTITUENTS
        self.mean_level = mean_level

    def level(self, hours_since_epoch):
        level = self.mean_level
        for amp, period, phase in self.constituents:
            phase_rad = phase * math.pi / 180.0
            omega = 2 * math.pi / period
            level += amp * math.sin(omega * hours_since_epoch + phase_rad)
        return level

    def find_extrema(self, start_hours, duration, step=0.05):
        times = []
        levels = []
        t = start_hours
        while t <= start_hours + duration:
            times.append(t)
            levels.append(self.level(t))
            t += step
        highs = []
        lows = []
        for i in range(1, len(times)-1):
            if levels[i] > levels[i-1] and levels[i] > levels[i+1]:
                highs.append((times[i], levels[i]))
            elif levels[i] < levels[i-1] and levels[i] < levels[i+1]:
                lows.append((times[i], levels[i]))
        return highs, lows

def datetime_to_epoch_hours(dt):
    # hours since 2000-01-01 00:00 UTC
    epoch = datetime(2000, 1, 1, tzinfo=timezone.utc)
    delta = dt - epoch
    return delta.total_seconds() / 3600.0

def format_time(hours):
    h = int(hours) % 24
    m = int((hours - int(hours)) * 60)
    return f"{h:02d}:{m:02d}"

def draw_graph(levels, start_hours, duration, width=50, height=20):
    if not levels:
        return "No data"
    min_lvl = min(levels)
    max_lvl = max(levels)
    range_lvl = max_lvl - min_lvl or 1
    # Sample points
    cols = width
    step = duration / cols
    data = []
    for i in range(cols):
        t = start_hours + i * step
        # find nearest level
        idx = int((t - start_hours) / step)
        if idx >= len(levels):
            idx = len(levels)-1
        data.append(levels[idx])
    min_d = min(data)
    max_d = max(data)
    range_d = max_d - min_d or 1
    grid = [[' ' for _ in range(cols)] for _ in range(height)]
    for i, val in enumerate(data):
        row = int((val - min_d) / range_d * (height-1))
        row = max(0, min(height-1, row))
        grid[row][i] = '*'
    # Add axis labels
    lines = []
    lines.append("Level (m)")
    for r in range(height-1, -1, -1):
        lvl = min_d + (max_d - min_d) * r / (height-1)
        line = f"{lvl:5.2f} |"
        line += ''.join(grid[r])
        lines.append(line)
    # Time axis
    line = "      " + "+" + "-" * (cols-1)
    lines.append(line)
    # Time labels (every 3 hours)
    labels = ""
    for h in range(0, int(duration)+1, 3):
        pos = int(h / duration * cols)
        label = f"{h:02d}:00"
        # Place label
        while len(labels) < pos:
            labels += " "
        labels += label
    lines.append("       " + labels)
    return "\n".join(lines)

def main():
    parser = argparse.ArgumentParser(description="Offline Tide Calendar")
    parser.add_argument("--date", help="Start date YYYY-MM-DD")
    parser.add_argument("--hours", type=float, default=24.0, help="Forecast duration (hours)")
    parser.add_argument("--graph", action="store_true", help="Show ASCII graph")
    parser.add_argument("--list", action="store_true", help="Show high/low tides (default)")
    parser.add_argument("--config", help="JSON file with constituents")
    parser.add_argument("--save-config", help="Save current config to file")
    args = parser.parse_args()

    # Load config
    constituents = DEFAULT_CONSTITUENTS
    mean_level = MEAN_LEVEL
    if args.config:
        with open(args.config, "r") as f:
            data = json.load(f)
            constituents = [tuple(c) for c in data.get("constituents", DEFAULT_CONSTITUENTS)]
            mean_level = data.get("mean_level", MEAN_LEVEL)

    if args.save_config:
        data = {
            "constituents": constituents,
            "mean_level": mean_level
        }
        with open(args.save_config, "w") as f:
            json.dump(data, f, indent=2)
        print(f"Config saved to {args.save_config}")
        return

    # Determine start time
    if args.date:
        dt = datetime.strptime(args.date, "%Y-%m-%d").replace(tzinfo=timezone.utc)
    else:
        dt = datetime.now(timezone.utc).replace(hour=0, minute=0, second=0, microsecond=0)

    start_hours = datetime_to_epoch_hours(dt)
    model = TideModel(constituents, mean_level)
    highs, lows = model.find_extrema(start_hours, args.hours)

    if args.graph:
        # Generate graph
        levels = []
        step = 0.1
        t = start_hours
        while t <= start_hours + args.hours:
            levels.append(model.level(t))
            t += step
        print(draw_graph(levels, start_hours, args.hours))
    else:
        # Table
        print(f"\n🌊 Offline Tide Calendar")
        print(f"Date: {dt.strftime('%Y-%m-%d %H:%M')} – {args.hours}h forecast\n")
        print("High/Low Tides:")
        if not highs and not lows:
            print("No extrema found (check range).")
        else:
            all_events = sorted(highs + lows, key=lambda x: x[0])
            for t, lvl in all_events:
                if t >= start_hours and t <= start_hours + args.hours:
                    dt_event = datetime(2000, 1, 1, tzinfo=timezone.utc) + timedelta(hours=t - datetime_to_epoch_hours(datetime(2000,1,1).replace(tzinfo=timezone.utc)))
                    # Actually we need the date from start
                    date_event = dt + timedelta(hours=t - start_hours)
                    typ = "High" if any(abs(lvl - h[1]) < 0.01 for h in highs) else "Low"
                    print(f"{date_event.strftime('%Y-%m-%d %H:%M')}  {typ:4}  {lvl:5.2f} m")

if __name__ == "__main__":
    main()
