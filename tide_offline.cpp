// tide_offline.cpp
#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include <cmath>
#include <ctime>
#include <iomanip>
#include <sstream>
#include <nlohmann/json.hpp>
#include <getopt.h>

using namespace std;
using json = nlohmann::json;

const double DEFAULT_CONSTITUENTS[][3] = {
    {1.2, 12.42, 0.0},
    {0.4, 12.00, 0.0},
    {0.2, 12.66, 0.0},
    {0.1, 23.93, 0.0},
    {0.08, 25.82, 0.0}
};
const double MEAN_LEVEL = 0.5;

class TideModel {
private:
    vector<array<double,3>> constituents;
    double meanLevel;
public:
    TideModel(const vector<array<double,3>>& cons, double mean) : constituents(cons), meanLevel(mean) {}

    double level(double hours) const {
        double lvl = meanLevel;
        for (auto& c : constituents) {
            double amp = c[0], period = c[1], phase = c[2];
            double phaseRad = phase * M_PI / 180.0;
            double omega = 2 * M_PI / period;
            lvl += amp * sin(omega * hours + phaseRad);
        }
        return lvl;
    }

    vector<pair<double,double>> findExtrema(double start, double duration) const {
        double step = 0.05;
        vector<double> times, levels;
        for (double t = start; t <= start + duration; t += step) {
            times.push_back(t);
            levels.push_back(level(t));
        }
        vector<pair<double,double>> extrema;
        for (size_t i = 1; i < times.size()-1; i++) {
            if (levels[i] > levels[i-1] && levels[i] > levels[i+1])
                extrema.push_back({times[i], levels[i]});
            else if (levels[i] < levels[i-1] && levels[i] < levels[i+1])
                extrema.push_back({times[i], levels[i]});
        }
        return extrema;
    }
};

double hoursSinceEpoch(const tm& dt) {
    time_t epoch = 946684800; // 2000-01-01 00:00:00 UTC
    tm epoch_tm = *gmtime(&epoch);
    time_t t = mktime(const_cast<tm*>(&dt));
    return difftime(t, epoch) / 3600.0;
}

string formatTime(double hours) {
    int h = (int)hours % 24;
    int m = (int)((hours - h) * 60);
    char buf[6];
    snprintf(buf, sizeof(buf), "%02d:%02d", h, m);
    return string(buf);
}

string drawGraph(const vector<double>& levels, double start, double duration, int width=50, int height=20) {
    if (levels.empty()) return "No data";
    double minLvl = *min_element(levels.begin(), levels.end());
    double maxLvl = *max_element(levels.begin(), levels.end());
    double rangeLvl = maxLvl - minLvl;
    if (rangeLvl == 0) rangeLvl = 1;
    int cols = width;
    double step = duration / cols;
    vector<double> sampled;
    for (int i=0; i<cols; i++) {
        double t = start + i * step;
        int idx = (int)((t - start) / step);
        if (idx >= (int)levels.size()) idx = levels.size()-1;
        sampled.push_back(levels[idx]);
    }
    double minS = *min_element(sampled.begin(), sampled.end());
    double maxS = *max_element(sampled.begin(), sampled.end());
    double rangeS = maxS - minS;
    if (rangeS == 0) rangeS = 1;
    vector<vector<char>> grid(height, vector<char>(cols, ' '));
    for (int i=0; i<cols; i++) {
        int row = (int)((sampled[i] - minS) / rangeS * (height - 1));
        if (row < 0) row = 0;
        if (row >= height) row = height - 1;
        grid[row][i] = '*';
    }
    stringstream ss;
    ss << "Level (m)\n";
    for (int r = height-1; r >= 0; r--) {
        double lvl = minS + (maxS - minS) * r / (height - 1);
        ss << fixed << setprecision(2) << setw(6) << lvl << " |";
        for (int c=0; c<cols; c++) ss << grid[r][c];
        ss << "\n";
    }
    ss << "      +" << string(cols-1, '-') << "\n";
    ss << "       ";
    for (int h=0; h<=duration; h+=3) {
        int pos = (int)(h / duration * cols);
        string label = (h<10?"0":"") + to_string(h) + ":00";
        while ((int)ss.str().length() < pos) ss << " ";
        ss << label;
    }
    ss << "\n";
    return ss.str();
}

int main(int argc, char* argv[]) {
    static struct option long_options[] = {
        {"date", required_argument, 0, 'd'},
        {"hours", required_argument, 0, 'h'},
        {"graph", no_argument, 0, 'g'},
        {"list", no_argument, 0, 'l'},
        {"config", required_argument, 0, 'c'},
        {"save-config", required_argument, 0, 's'},
        {0,0,0,0}
    };
    int opt;
    string dateStr, configFile, saveConfigFile;
    double hours = 24.0;
    bool graph = false, list = false;
    while ((opt = getopt_long(argc, argv, "d:h:glc:s:", long_options, nullptr)) != -1) {
        switch (opt) {
            case 'd': dateStr = optarg; break;
            case 'h': hours = stod(optarg); break;
            case 'g': graph = true; break;
            case 'l': list = true; break;
            case 'c': configFile = optarg; break;
            case 's': saveConfigFile = optarg; break;
            default:
                cerr << "Usage: tide_offline --date YYYY-MM-DD --hours N --graph --list --config FILE --save-config FILE\n";
                return 1;
        }
    }

    vector<array<double,3>> constituents;
    for (auto& c : DEFAULT_CONSTITUENTS) constituents.push_back({c[0], c[1], c[2]});
    double meanLevel = MEAN_LEVEL;

    if (!configFile.empty()) {
        ifstream f(configFile);
        if (!f.is_open()) { cerr << "Cannot open config file\n"; return 1; }
        json j;
        f >> j;
        if (j.contains("constituents")) {
            constituents.clear();
            for (auto& item : j["constituents"]) {
                constituents.push_back({item[0], item[1], item[2]});
            }
        }
        if (j.contains("mean_level")) meanLevel = j["mean_level"];
    }

    if (!saveConfigFile.empty()) {
        json j;
        j["constituents"] = json::array();
        for (auto& c : constituents) {
            j["constituents"].push_back({c[0], c[1], c[2]});
        }
        j["mean_level"] = meanLevel;
        ofstream f(saveConfigFile);
        f << setw(2) << j << endl;
        cout << "Config saved to " << saveConfigFile << "\n";
        return 0;
    }

    time_t now = time(nullptr);
    tm dt = *gmtime(&now);
    if (!dateStr.empty()) {
        strptime(dateStr.c_str(), "%Y-%m-%d", &dt);
    }
    dt.tm_hour = 0; dt.tm_min = 0; dt.tm_sec = 0;

    double startHours = hoursSinceEpoch(dt);
    TideModel model(constituents, meanLevel);
    auto extrema = model.findExtrema(startHours, hours);

    if (graph) {
        vector<double> levels;
        double step = 0.1;
        for (double t = startHours; t <= startHours + hours; t += step)
            levels.push_back(model.level(t));
        cout << drawGraph(levels, startHours, hours);
    } else {
        cout << "\n🌊 Offline Tide Calendar\n";
        char dateBuf[20];
        strftime(dateBuf, sizeof(dateBuf), "%Y-%m-%d %H:%M", &dt);
        cout << "Date: " << dateBuf << " – " << hours << "h forecast\n\n";
        cout << "High/Low Tides:\n";
        sort(extrema.begin(), extrema.end());
        for (auto& e : extrema) {
            if (e.first >= startHours && e.first <= startHours + hours) {
                time_t t = mktime(&dt) + (time_t)((e.first - startHours) * 3600);
                tm evt = *gmtime(&t);
                char buf[20];
                strftime(buf, sizeof(buf), "%Y-%m-%d %H:%M", &evt);
                string typ = (e.second > meanLevel) ? "High" : "Low";
                cout << buf << "  " << typ << "    " << fixed << setprecision(2) << e.second << " m\n";
            }
        }
    }
    return 0;
}
