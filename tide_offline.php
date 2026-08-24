# tide_offline.php
#!/usr/bin/env php
<?php

define('DEFAULT_CONSTITUENTS', [
    [1.2, 12.42, 0.0],
    [0.4, 12.00, 0.0],
    [0.2, 12.66, 0.0],
    [0.1, 23.93, 0.0],
    [0.08, 25.82, 0.0],
]);
define('MEAN_LEVEL', 0.5);

class TideModel {
    private $constituents;
    private $mean_level;

    public function __construct($constituents = DEFAULT_CONSTITUENTS, $mean_level = MEAN_LEVEL) {
        $this->constituents = $constituents;
        $this->mean_level = $mean_level;
    }

    public function level($hours) {
        $level = $this->mean_level;
        foreach ($this->constituents as $c) {
            list($amp, $period, $phase) = $c;
            $phase_rad = $phase * M_PI / 180.0;
            $omega = 2 * M_PI / $period;
            $level += $amp * sin($omega * $hours + $phase_rad);
        }
        return $level;
    }

    public function findExtrema($start, $duration, $step = 0.05) {
        $times = [];
        $levels = [];
        $t = $start;
        while ($t <= $start + $duration) {
            $times[] = $t;
            $levels[] = $this->level($t);
            $t += $step;
        }
        $highs = [];
        $lows = [];
        for ($i=1; $i<count($times)-1; $i++) {
            if ($levels[$i] > $levels[$i-1] && $levels[$i] > $levels[$i+1]) {
                $highs[] = ['t' => $times[$i], 'lvl' => $levels[$i]];
            } elseif ($levels[$i] < $levels[$i-1] && $levels[$i] < $levels[$i+1]) {
                $lows[] = ['t' => $times[$i], 'lvl' => $levels[$i]];
            }
        }
        return [$highs, $lows];
    }
}

function hoursSinceEpoch($dt) {
    $epoch = new DateTime('2000-01-01 00:00:00', new DateTimeZone('UTC'));
    return ($dt->getTimestamp() - $epoch->getTimestamp()) / 3600.0;
}

function formatTime($hours) {
    $h = (int)$hours % 24;
    $m = (int)(($hours - $h) * 60);
    return sprintf("%02d:%02d", $h, $m);
}

function drawGraph($levels, $start, $duration, $width = 50, $height = 20) {
    if (empty($levels)) return "No data";
    $min_lvl = min($levels);
    $max_lvl = max($levels);
    $range_lvl = $max_lvl - $min_lvl ?: 1;
    $cols = $width;
    $step = $duration / $cols;
    $sampled = [];
    for ($i=0; $i<$cols; $i++) {
        $t = $start + $i * $step;
        $idx = (int)(($t - $start) / $step);
        if ($idx >= count($levels)) $idx = count($levels)-1;
        $sampled[] = $levels[$idx];
    }
    $min_s = min($sampled);
    $max_s = max($sampled);
    $range_s = $max_s - $min_s ?: 1;
    $grid = array_fill(0, $height, array_fill(0, $cols, ' '));
    for ($i=0; $i<$cols; $i++) {
        $row = (int)(($sampled[$i] - $min_s) / $range_s * ($height - 1));
        if ($row < 0) $row = 0;
        if ($row >= $height) $row = $height - 1;
        $grid[$row][$i] = '*';
    }
    $lines = [];
    $lines[] = "Level (m)";
    for ($r = $height-1; $r >= 0; $r--) {
        $lvl = $min_s + ($max_s - $min_s) * $r / ($height - 1);
        $line = sprintf("%5.2f |", $lvl);
        $line .= implode('', $grid[$r]);
        $lines[] = $line;
    }
    $lines[] = "      +" . str_repeat("-", $cols - 1);
    $labels = "";
    for ($h=0; $h<=$duration; $h+=3) {
        $pos = (int)($h / $duration * $cols);
        $label = sprintf("%02d:00", $h);
        while (strlen($labels) < $pos) $labels .= " ";
        $labels .= $label;
    }
    $lines[] = "       " . $labels;
    return implode("\n", $lines);
}

$opts = getopt("", ["date:", "hours:", "graph", "list", "config:", "save-config:"]);
$date = $opts['date'] ?? null;
$hours = isset($opts['hours']) ? (float)$opts['hours'] : 24.0;
$graph = isset($opts['graph']);
$list = isset($opts['list']);
$configFile = $opts['config'] ?? null;
$saveConfig = $opts['save-config'] ?? null;

$constituents = DEFAULT_CONSTITUENTS;
$mean_level = MEAN_LEVEL;
if ($configFile) {
    $data = json_decode(file_get_contents($configFile), true);
    $constituents = $data['constituents'] ?? DEFAULT_CONSTITUENTS;
    $mean_level = $data['mean_level'] ?? MEAN_LEVEL;
}

if ($saveConfig) {
    file_put_contents($saveConfig, json_encode(['constituents' => $constituents, 'mean_level' => $mean_level], JSON_PRETTY_PRINT));
    echo "Config saved to $saveConfig\n";
    exit(0);
}

if ($date) {
    $startTime = new DateTime($date . ' 00:00:00', new DateTimeZone('UTC'));
} else {
    $now = new DateTime('now', new DateTimeZone('UTC'));
    $startTime = new DateTime($now->format('Y-m-d') . ' 00:00:00', new DateTimeZone('UTC'));
}

$startHours = hoursSinceEpoch($startTime);
$model = new TideModel($constituents, $mean_level);
list($highs, $lows) = $model->findExtrema($startHours, $hours);

if ($graph) {
    $levels = [];
    $step = 0.1;
    for ($t = $startHours; $t <= $startHours + $hours; $t += $step) {
        $levels[] = $model->level($t);
    }
    echo drawGraph($levels, $startHours, $hours);
} else {
    echo "\n🌊 Offline Tide Calendar\n";
    echo "Date: " . $startTime->format('Y-m-d H:i') . " – {$hours}h forecast\n\n";
    echo "High/Low Tides:\n";
    $events = array_merge($highs, $lows);
    usort($events, function($a, $b) { return $a['t'] <=> $b['t']; });
    foreach ($events as $e) {
        if ($e['t'] >= $startHours && $e['t'] <= $startHours + $hours) {
            $dt = clone $startTime;
            $dt->modify('+' . (($e['t'] - $startHours) * 3600) . ' seconds');
            $typ = 'Low';
            foreach ($highs as $h) if (abs($h['lvl'] - $e['lvl']) < 0.001) { $typ = 'High'; break; }
            echo $dt->format('Y-m-d H:i') . "  " . str_pad($typ, 4) . "  " . number_format($e['lvl'], 2) . " m\n";
        }
    }
}
?>
