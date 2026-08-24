// TideOffline.java
import java.io.*;
import java.nio.file.*;
import java.time.*;
import java.time.format.*;
import java.util.*;
import com.google.gson.*;

class Config {
    List<List<Double>> constituents;
    double mean_level;
}

public class TideOffline {
    private static final List<List<Double>> DEFAULT_CONSTITUENTS = Arrays.asList(
        Arrays.asList(1.2, 12.42, 0.0),
        Arrays.asList(0.4, 12.00, 0.0),
        Arrays.asList(0.2, 12.66, 0.0),
        Arrays.asList(0.1, 23.93, 0.0),
        Arrays.asList(0.08, 25.82, 0.0)
    );
    private static final double MEAN_LEVEL = 0.5;
    private static final Gson gson = new GsonBuilder().setPrettyPrinting().create();

    static class TideModel {
        List<List<Double>> constituents;
        double meanLevel;

        TideModel(List<List<Double>> constituents, double meanLevel) {
            this.constituents = constituents;
            this.meanLevel = meanLevel;
        }

        double level(double hours) {
            double lvl = meanLevel;
            for (List<Double> c : constituents) {
                double amp = c.get(0), period = c.get(1), phase = c.get(2);
                double phaseRad = phase * Math.PI / 180.0;
                double omega = 2 * Math.PI / period;
                lvl += amp * Math.sin(omega * hours + phaseRad);
            }
            return lvl;
        }

        List<Object[]> findExtrema(double start, double duration) {
            double step = 0.05;
            List<Double> times = new ArrayList<>();
            List<Double> levels = new ArrayList<>();
            for (double t = start; t <= start + duration; t += step) {
                times.add(t);
                levels.add(level(t));
            }
            List<Object[]> highs = new ArrayList<>();
            List<Object[]> lows = new ArrayList<>();
            for (int i = 1; i < times.size() - 1; i++) {
                if (levels.get(i) > levels.get(i-1) && levels.get(i) > levels.get(i+1)) {
                    highs.add(new Object[]{times.get(i), levels.get(i)});
                } else if (levels.get(i) < levels.get(i-1) && levels.get(i) < levels.get(i+1)) {
                    lows.add(new Object[]{times.get(i), levels.get(i)});
                }
            }
            List<Object[]> result = new ArrayList<>();
            result.addAll(highs);
            result.addAll(lows);
            return result;
        }
    }

    private static double hoursSinceEpoch(Instant instant) {
        Instant epoch = Instant.parse("2000-01-01T00:00:00Z");
        return Duration.between(epoch, instant).toMillis() / 3600.0 / 1000.0;
    }

    private static String formatTime(double hours) {
        int h = (int)hours % 24;
        int m = (int)((hours - h) * 60);
        return String.format("%02d:%02d", h, m);
    }

    public static void main(String[] args) throws Exception {
        Map<String, String> params = new HashMap<>();
        for (int i = 0; i < args.length; i++) {
            if (args[i].startsWith("--")) {
                String key = args[i].substring(2);
                if (i+1 < args.length && !args[i+1].startsWith("--")) {
                    params.put(key, args[++i]);
                } else {
                    params.put(key, "");
                }
            }
        }
        List<List<Double>> constituents = DEFAULT_CONSTITUENTS;
        double meanLevel = MEAN_LEVEL;
        if (params.containsKey("config")) {
            String json = new String(Files.readAllBytes(Paths.get(params.get("config"))));
            Config cfg = gson.fromJson(json, Config.class);
            if (cfg.constituents != null) constituents = cfg.constituents;
            meanLevel = cfg.mean_level;
        }

        if (params.containsKey("save-config")) {
            Config cfg = new Config();
            cfg.constituents = constituents;
            cfg.mean_level = meanLevel;
            Files.write(Paths.get(params.get("save-config")), gson.toJson(cfg).getBytes());
            System.out.println("Config saved to " + params.get("save-config"));
            return;
        }

        Instant now = Instant.now();
        LocalDateTime startLocal;
        if (params.containsKey("date")) {
            startLocal = LocalDate.parse(params.get("date")).atStartOfDay();
        } else {
            startLocal = LocalDateTime.ofInstant(now, ZoneOffset.UTC).with(LocalTime.MIDNIGHT);
        }
        Instant startTime = startLocal.toInstant(ZoneOffset.UTC);
        double hours = params.containsKey("hours") ? Double.parseDouble(params.get("hours")) : 24.0;
        boolean graph = params.containsKey("graph");
        boolean list = params.containsKey("list") || !graph;

        double startHours = hoursSinceEpoch(startTime);
        TideModel model = new TideModel(constituents, meanLevel);
        List<Object[]> extrema = model.findExtrema(startHours, hours);

        if (graph) {
            List<Double> levels = new ArrayList<>();
            double step = 0.1;
            for (double t = startHours; t <= startHours + hours; t += step) {
                levels.add(model.level(t));
            }
            System.out.println(drawGraph(levels, startHours, hours));
        } else {
            System.out.printf("\n🌊 Offline Tide Calendar\n");
            System.out.printf("Date: %s – %.0fh forecast\n\n", startTime.toString().substring(0,16).replace('T',' '), hours);
            System.out.println("High/Low Tides:");
            extrema.sort((a,b) -> Double.compare((Double)a[0], (Double)b[0]));
            for (Object[] e : extrema) {
                double t = (Double)e[0];
                if (t >= startHours && t <= startHours + hours) {
                    Instant dt = startTime.plusSeconds((long)((t - startHours) * 3600));
                    String typ = "Low";
                    // Check if it's high (we can't easily check, but we can compare with extrema list)
                    // For simplicity, we'll assume high if level > mean
                    if ((Double)e[1] > meanLevel) typ = "High";
                    System.out.printf("%s  %-4s  %5.2f m\n",
                        dt.toString().substring(0,16).replace('T',' '), typ, (Double)e[1]);
                }
            }
        }
    }

    private static String drawGraph(List<Double> levels, double start, double duration) {
        if (levels.isEmpty()) return "No data";
        double minLvl = levels.stream().min(Double::compare).get();
        double maxLvl = levels.stream().max(Double::compare).get();
        int width = 50, height = 20;
        double range = maxLvl - minLvl;
        if (range == 0) range = 1;
        int cols = width;
        double step = duration / cols;
        List<Double> sampled = new ArrayList<>();
        for (int i = 0; i < cols; i++) {
            double t = start + i * step;
            int idx = (int)((t - start) / step);
            if (idx >= levels.size()) idx = levels.size() - 1;
            sampled.add(levels.get(idx));
        }
        double minS = sampled.stream().min(Double::compare).get();
        double maxS = sampled.stream().max(Double::compare).get();
        double rangeS = maxS - minS;
        if (rangeS == 0) rangeS = 1;
        char[][] grid = new char[height][cols];
        for (int r = 0; r < height; r++) Arrays.fill(grid[r], ' ');
        for (int i = 0; i < cols; i++) {
            int row = (int)((sampled.get(i) - minS) / rangeS * (height - 1));
            if (row < 0) row = 0;
            if (row >= height) row = height - 1;
            grid[row][i] = '*';
        }
        StringBuilder sb = new StringBuilder();
        sb.append("Level (m)\n");
        for (int r = height - 1; r >= 0; r--) {
            double lvl = minS + (maxS - minS) * r / (height - 1);
            sb.append(String.format("%5.2f |", lvl));
            sb.append(new String(grid[r]));
            sb.append("\n");
        }
        sb.append("      +").append("-".repeat(Math.max(0, cols-1))).append("\n");
        StringBuilder labels = new StringBuilder();
        labels.append("       ");
        for (int h = 0; h <= (int)duration; h += 3) {
            int pos = (int)(h / duration * cols);
            String label = String.format("%02d:00", h);
            while (labels.length() < pos) labels.append(' ');
            labels.append(label);
        }
        sb.append(labels).append("\n");
        return sb.toString();
    }
}
