// TideOffline.cs
using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text.Json;
using System.Text.Json.Serialization;

class Config
{
    [JsonPropertyName("constituents")]
    public List<List<double>> Constituents { get; set; }
    [JsonPropertyName("mean_level")]
    public double MeanLevel { get; set; }
}

class TideModel
{
    private List<List<double>> constituents;
    private double meanLevel;

    public TideModel(List<List<double>> constituents, double meanLevel)
    {
        this.constituents = constituents;
        this.meanLevel = meanLevel;
    }

    public double Level(double hours)
    {
        double lvl = meanLevel;
        foreach (var c in constituents)
        {
            double amp = c[0], period = c[1], phase = c[2];
            double phaseRad = phase * Math.PI / 180.0;
            double omega = 2 * Math.PI / period;
            lvl += amp * Math.Sin(omega * hours + phaseRad);
        }
        return lvl;
    }

    public List<(double t, double lvl)> FindExtrema(double start, double duration)
    {
        double step = 0.05;
        var times = new List<double>();
        var levels = new List<double>();
        for (double t = start; t <= start + duration; t += step)
        {
            times.Add(t);
            levels.Add(Level(t));
        }
        var extrema = new List<(double, double)>();
        for (int i = 1; i < times.Count - 1; i++)
        {
            if (levels[i] > levels[i-1] && levels[i] > levels[i+1])
                extrema.Add((times[i], levels[i]));
            else if (levels[i] < levels[i-1] && levels[i] < levels[i+1])
                extrema.Add((times[i], levels[i]));
        }
        return extrema;
    }
}

class Program
{
    private static readonly List<List<double>> DefaultConstituents = new List<List<double>>
    {
        new List<double>{1.2, 12.42, 0.0},
        new List<double>{0.4, 12.00, 0.0},
        new List<double>{0.2, 12.66, 0.0},
        new List<double>{0.1, 23.93, 0.0},
        new List<double>{0.08, 25.82, 0.0},
    };
    private const double MeanLevel = 0.5;

    static double HoursSinceEpoch(DateTime dt)
    {
        var epoch = new DateTime(2000, 1, 1, 0, 0, 0, DateTimeKind.Utc);
        return (dt - epoch).TotalHours;
    }

    static string FormatTime(double hours)
    {
        int h = (int)hours % 24;
        int m = (int)((hours - h) * 60);
        return $"{h:D2}:{m:D2}";
    }

    static string DrawGraph(List<double> levels, double start, double duration)
    {
        if (levels.Count == 0) return "No data";
        double minLvl = levels.Min();
        double maxLvl = levels.Max();
        double rangeLvl = maxLvl - minLvl;
        if (rangeLvl == 0) rangeLvl = 1;
        int width = 50, height = 20;
        int cols = width;
        double step = duration / cols;
        var sampled = new List<double>();
        for (int i = 0; i < cols; i++)
        {
            double t = start + i * step;
            int idx = (int)((t - start) / step);
            if (idx >= levels.Count) idx = levels.Count - 1;
            sampled.Add(levels[idx]);
        }
        double minS = sampled.Min();
        double maxS = sampled.Max();
        double rangeS = maxS - minS;
        if (rangeS == 0) rangeS = 1;
        char[][] grid = new char[height][];
        for (int r = 0; r < height; r++) grid[r] = new string(' ', cols).ToCharArray();
        for (int i = 0; i < cols; i++)
        {
            int row = (int)((sampled[i] - minS) / rangeS * (height - 1));
            if (row < 0) row = 0;
            if (row >= height) row = height - 1;
            grid[row][i] = '*';
        }
        var lines = new List<string>();
        lines.Add("Level (m)");
        for (int r = height - 1; r >= 0; r--)
        {
            double lvl = minS + (maxS - minS) * r / (height - 1);
            string line = $"{lvl,5:F2} |";
            line += new string(grid[r]);
            lines.Add(line);
        }
        lines.Add("      +" + new string('-', cols - 1));
        string labels = "       ";
        for (int h = 0; h <= duration; h += 3)
        {
            int pos = (int)(h / duration * cols);
            string label = $"{h:D2}:00";
            while (labels.Length < pos) labels += " ";
            labels += label;
        }
        lines.Add(labels);
        return string.Join("\n", lines);
    }

    static void Main(string[] args)
    {
        var parsed = ParseArgs(args);
        var constituents = DefaultConstituents;
        double meanLevel = MeanLevel;

        if (parsed.ContainsKey("config"))
        {
            string json = File.ReadAllText(parsed["config"]);
            var cfg = JsonSerializer.Deserialize<Config>(json);
            if (cfg.Constituents != null) constituents = cfg.Constituents;
            meanLevel = cfg.MeanLevel;
        }

        if (parsed.ContainsKey("save-config"))
        {
            var cfg = new Config { Constituents = constituents, MeanLevel = meanLevel };
            string json = JsonSerializer.Serialize(cfg, new JsonSerializerOptions { WriteIndented = true });
            File.WriteAllText(parsed["save-config"], json);
            Console.WriteLine($"Config saved to {parsed["save-config"]}");
            return;
        }

        DateTime startTime;
        if (parsed.ContainsKey("date"))
        {
            startTime = DateTime.ParseExact(parsed["date"], "yyyy-MM-dd", null).ToUniversalTime();
        }
        else
        {
            var now = DateTime.UtcNow;
            startTime = new DateTime(now.Year, now.Month, now.Day, 0, 0, 0, DateTimeKind.Utc);
        }

        double hours = parsed.ContainsKey("hours") ? double.Parse(parsed["hours"]) : 24.0;
        bool graph = parsed.ContainsKey("graph");
        bool list = parsed.ContainsKey("list") || !graph;

        double startHours = HoursSinceEpoch(startTime);
        var model = new TideModel(constituents, meanLevel);
        var extrema = model.FindExtrema(startHours, hours);

        if (graph)
        {
            var levels = new List<double>();
            double step = 0.1;
            for (double t = startHours; t <= startHours + hours; t += step)
                levels.Add(model.Level(t));
            Console.WriteLine(DrawGraph(levels, startHours, hours));
        }
        else
        {
            Console.WriteLine($"\n🌊 Offline Tide Calendar");
            Console.WriteLine($"Date: {startTime:yyyy-MM-dd HH:mm} – {hours:F0}h forecast\n");
            Console.WriteLine("High/Low Tides:");
            extrema.Sort((a, b) => a.t.CompareTo(b.t));
            foreach (var (t, lvl) in extrema)
            {
                if (t >= startHours && t <= startHours + hours)
                {
                    var dt = startTime.AddSeconds((t - startHours) * 3600);
                    string typ = lvl > meanLevel ? "High" : "Low";
                    Console.WriteLine($"{dt:yyyy-MM-dd HH:mm}  {typ,-4}  {lvl,5:F2} m");
                }
            }
        }
    }

    static Dictionary<string, string> ParseArgs(string[] args)
    {
        var dict = new Dictionary<string, string>();
        for (int i = 0; i < args.Length; i++)
        {
            if (args[i].StartsWith("--"))
            {
                string key = args[i].Substring(2);
                if (i + 1 < args.Length && !args[i + 1].StartsWith("--"))
                    dict[key] = args[++i];
                else
                    dict[key] = "";
            }
        }
        return dict;
    }
}
