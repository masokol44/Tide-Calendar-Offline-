# tide_offline.rb
#!/usr/bin/env ruby
require 'json'
require 'optparse'
require 'date'

DEFAULT_CONSTITUENTS = [
  [1.2, 12.42, 0.0],
  [0.4, 12.00, 0.0],
  [0.2, 12.66, 0.0],
  [0.1, 23.93, 0.0],
  [0.08, 25.82, 0.0],
]
MEAN_LEVEL = 0.5

class TideModel
  def initialize(constituents = DEFAULT_CONSTITUENTS, mean_level = MEAN_LEVEL)
    @constituents = constituents
    @mean_level = mean_level
  end

  def level(hours)
    level = @mean_level
    @constituents.each do |amp, period, phase|
      phase_rad = phase * Math::PI / 180.0
      omega = 2 * Math::PI / period
      level += amp * Math.sin(omega * hours + phase_rad)
    end
    level
  end

  def find_extrema(start, duration, step = 0.05)
    times = []
    levels = []
    t = start
    while t <= start + duration
      times << t
      levels << level(t)
      t += step
    end
    highs = []
    lows = []
    (1...times.length-1).each do |i|
      if levels[i] > levels[i-1] && levels[i] > levels[i+1]
        highs << { t: times[i], lvl: levels[i] }
      elsif levels[i] < levels[i-1] && levels[i] < levels[i+1]
        lows << { t: times[i], lvl: levels[i] }
      end
    end
    [highs, lows]
  end
end

def hours_since_epoch(dt)
  epoch = Time.utc(2000, 1, 1)
  (dt - epoch) / 3600.0
end

def format_time(hours)
  h = hours.to_i % 24
  m = ((hours - h) * 60).to_i
  sprintf("%02d:%02d", h, m)
end

def draw_graph(levels, start, duration, width = 50, height = 20)
  return "No data" if levels.empty?
  min_lvl = levels.min
  max_lvl = levels.max
  range_lvl = max_lvl - min_lvl
  range_lvl = 1 if range_lvl == 0
  cols = width
  step = duration / cols
  sampled = []
  cols.times do |i|
    t = start + i * step
    idx = ((t - start) / step).floor
    idx = levels.length - 1 if idx >= levels.length
    sampled << levels[idx]
  end
  min_s = sampled.min
  max_s = sampled.max
  range_s = max_s - min_s
  range_s = 1 if range_s == 0
  grid = Array.new(height) { Array.new(cols, ' ') }
  cols.times do |i|
    row = ((sampled[i] - min_s) / range_s * (height - 1)).to_i
    row = height - 1 if row >= height
    row = 0 if row < 0
    grid[row][i] = '*'
  end
  lines = []
  lines << "Level (m)"
  (height-1).downto(0) do |r|
    lvl = min_s + (max_s - min_s) * r / (height - 1)
    line = sprintf("%5.2f |", lvl)
    line += grid[r].join
    lines << line
  end
  lines << "      +" + "-" * (cols - 1)
  labels = ""
  0.step(duration.to_i, 3) do |h|
    pos = (h / duration * cols).to_i
    label = sprintf("%02d:00", h)
    labels << " " * (pos - labels.length) if labels.length < pos
    labels << label
  end
  lines << "       " + labels
  lines.join("\n")
end

options = {}
OptionParser.new do |opts|
  opts.banner = "Usage: tide_offline.rb [options]"
  opts.on("--date DATE", "Start date YYYY-MM-DD") { |v| options[:date] = v }
  opts.on("--hours N", Float, "Forecast duration (hours)") { |v| options[:hours] = v }
  opts.on("--graph", "Show ASCII graph") { options[:graph] = true }
  opts.on("--list", "Show high/low tides") { options[:list] = true }
  opts.on("--config FILE", "JSON config file") { |v| options[:config] = v }
  opts.on("--save-config FILE", "Save config") { |v| options[:save_config] = v }
end.parse!

constituents = DEFAULT_CONSTITUENTS
mean_level = MEAN_LEVEL
if options[:config]
  data = JSON.parse(File.read(options[:config]))
  constituents = data['constituents'] || DEFAULT_CONSTITUENTS
  mean_level = data['mean_level'] || MEAN_LEVEL
end

if options[:save_config]
  File.write(options[:save_config], JSON.pretty_generate({ constituents: constituents, mean_level: mean_level }))
  puts "Config saved to #{options[:save_config]}"
  exit 0
end

if options[:date]
  start_time = DateTime.parse(options[:date]).to_time.utc
else
  now = Time.now.utc
  start_time = Time.utc(now.year, now.month, now.day)
end

start_hours = hours_since_epoch(start_time)
duration = options[:hours] || 24.0
model = TideModel.new(constituents, mean_level)
highs, lows = model.find_extrema(start_hours, duration)

if options[:graph]
  levels = []
  step = 0.1
  t = start_hours
  while t <= start_hours + duration
    levels << model.level(t)
    t += step
  end
  puts draw_graph(levels, start_hours, duration)
else
  puts "\n🌊 Offline Tide Calendar"
  puts "Date: #{start_time.strftime('%Y-%m-%d %H:%M')} – #{duration}h forecast\n"
  puts "High/Low Tides:"
  events = (highs + lows).sort_by { |e| e[:t] }
  events.each do |e|
    if e[:t] >= start_hours && e[:t] <= start_hours + duration
      dt = start_time + (e[:t] - start_hours) * 3600
      typ = highs.any? { |h| (h[:lvl] - e[:lvl]).abs < 0.001 } ? "High" : "Low"
      puts "#{dt.strftime('%Y-%m-%d %H:%M')}  #{typ.ljust(4)}  #{'%.2f' % e[:lvl]} m"
    end
  end
end
