// tide_offline.js
#!/usr/bin/env node
const fs = require('fs');
const { program } = require('commander');

const DEFAULT_CONSTITUENTS = [
    [1.2, 12.42, 0.0],
    [0.4, 12.00, 0.0],
    [0.2, 12.66, 0.0],
    [0.1, 23.93, 0.0],
    [0.08, 25.82, 0.0],
];
const MEAN_LEVEL = 0.5;

class TideModel {
    constructor(constituents = DEFAULT_CONSTITUENTS, meanLevel = MEAN_LEVEL) {
        this.constituents = constituents;
        this.meanLevel = meanLevel;
    }

    level(hours) {
        let level = this.meanLevel;
        for (const [amp, period, phase] of this.constituents) {
            const phaseRad = phase * Math.PI / 180;
            const omega = 2 * Math.PI / period;
            level += amp * Math.sin(omega * hours + phaseRad);
        }
        return level;
    }

    findExtrema(start, duration, step = 0.05) {
        const times = [];
        const levels = [];
        for (let t = start; t <= start + duration; t += step) {
            times.push(t);
            levels.push(this.level(t));
        }
        const highs = [], lows = [];
        for (let i = 1; i < times.length - 1; i++) {
            if (levels[i] > levels[i-1] && levels[i] > levels[i+1]) {
                highs.push({ t: times[i], lvl: levels[i] });
            } else if (levels[i] < levels[i-1] && levels[i] < levels[i+1]) {
                lows.push({ t: times[i], lvl: levels[i] });
            }
        }
        return { highs, lows };
    }
}

function hoursSinceEpoch(dt) {
    const epoch = new Date(Date.UTC(2000, 0, 1));
    return (dt.getTime() - epoch.getTime()) / (1000 * 60 * 60);
}

function formatTime(hours) {
    const h = Math.floor(hours) % 24;
    const m = Math.floor((hours - h) * 60);
    return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
}

function drawGraph(levels, start, duration, width = 50, height = 20) {
    if (levels.length === 0) return "No data";
    const minLvl = Math.min(...levels);
    const maxLvl = Math.max(...levels);
    const rangeLvl = maxLvl - minLvl || 1;
    const cols = width;
    const step = duration / cols;
    const sampled = [];
    for (let i = 0; i < cols; i++) {
        const t = start + i * step;
        const idx = Math.floor((t - start) / step);
        sampled.push(levels[idx] || levels[levels.length-1]);
    }
    const minS = Math.min(...sampled);
    const maxS = Math.max(...sampled);
    const rangeS = maxS - minS || 1;
    const grid = Array.from({ length: height }, () => Array(cols).fill(' '));
    for (let i = 0; i < cols; i++) {
        const row = Math.floor((sampled[i] - minS) / rangeS * (height - 1));
        if (row >= 0 && row < height) {
            grid[row][i] = '*';
        }
    }
    const lines = [];
    lines.push('Level (m)');
    for (let r = height - 1; r >= 0; r--) {
        const lvl = minS + (maxS - minS) * r / (height - 1);
        let line = lvl.toFixed(2).padStart(6) + ' |';
        line += grid[r].join('');
        lines.push(line);
    }
    let line = '      +' + '-'.repeat(cols - 1);
    lines.push(line);
    let labels = '';
    for (let h = 0; h <= duration; h += 3) {
        const pos = Math.floor(h / duration * cols);
        const label = String(h).padStart(2, '0') + ':00';
        while (labels.length < pos) labels += ' ';
        labels += label;
    }
    lines.push('       ' + labels);
    return lines.join('\n');
}

program
    .option('--date <date>', 'Start date YYYY-MM-DD')
    .option('--hours <hours>', 'Forecast duration (hours)', parseFloat, 24)
    .option('--graph', 'Show ASCII graph')
    .option('--list', 'Show high/low tides', true)
    .option('--config <file>', 'JSON config file')
    .option('--save-config <file>', 'Save config to file')
    .parse(process.argv);

const opts = program.opts();

let constituents = DEFAULT_CONSTITUENTS;
let meanLevel = MEAN_LEVEL;
if (opts.config) {
    const data = JSON.parse(fs.readFileSync(opts.config, 'utf8'));
    constituents = data.constituents || DEFAULT_CONSTITUENTS;
    meanLevel = data.mean_level || MEAN_LEVEL;
}

if (opts.saveConfig) {
    fs.writeFileSync(opts.saveConfig, JSON.stringify({ constituents, mean_level: meanLevel }, null, 2));
    console.log(`Config saved to ${opts.saveConfig}`);
    process.exit(0);
}

let startTime;
if (opts.date) {
    startTime = new Date(opts.date + 'T00:00:00Z');
} else {
    const now = new Date();
    startTime = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
}

const startHours = hoursSinceEpoch(startTime);
const model = new TideModel(constituents, meanLevel);
const { highs, lows } = model.findExtrema(startHours, opts.hours);

if (opts.graph) {
    const levels = [];
    const step = 0.1;
    for (let t = startHours; t <= startHours + opts.hours; t += step) {
        levels.push(model.level(t));
    }
    console.log(drawGraph(levels, startHours, opts.hours));
} else {
    console.log(`\n🌊 Offline Tide Calendar`);
    console.log(`Date: ${startTime.toISOString().slice(0,16).replace('T',' ')} – ${opts.hours}h forecast\n`);
    console.log('High/Low Tides:');
    const events = [...highs, ...lows].sort((a, b) => a.t - b.t);
    for (const e of events) {
        if (e.t >= startHours && e.t <= startHours + opts.hours) {
            const dt = new Date(startTime.getTime() + (e.t - startHours) * 3600 * 1000);
            const typ = highs.some(h => Math.abs(h.lvl - e.lvl) < 0.001) ? 'High' : 'Low';
            console.log(`${dt.toISOString().slice(0,16).replace('T',' ')}  ${typ.padEnd(4)}  ${e.lvl.toFixed(2)} m`);
        }
    }
}
