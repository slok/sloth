// --------- THEME ---------.
function setColorTheme(darkMode) {
    mode = darkMode ? "dark" : "light";

    // Based on picocoss docs: <html data-theme="light|dark">
    document.documentElement.setAttribute("data-theme", mode);
}

function isColorThemeLight() {
    let theme = document.documentElement.getAttribute("data-theme");
    if (theme) {
        return theme != "dark";
    }

    if (window.matchMedia) {
        return !window.matchMedia('(prefers-color-scheme: dark)').matches;
    }

    return true; // Default to light.
}

// --------- CSS  ---------.
function getCSSVariableValue(cssVar) {
    // Sanitize variable name.
    if (cssVar.startsWith("--")) {
        cssVar = cssVar.trimStart("--");
    }
    return getComputedStyle(document.documentElement).getPropertyValue(cssVar).trim();
}

// Re-create lucide icons after htmx content swap.
window.addEventListener('load', function () {
    document.body.addEventListener('htmx:afterSwap', function(evt) {
        lucide.createIcons();
    });
})


// --------- PLOTS ---------.

function renderUplotSLIChart(domElID, json) {
    const container = document.getElementById(domElID);
    const sloLine = Array(json.timestamps.length).fill(json.slo_objective);
    const light = isColorThemeLight();
    sliColor = getCSSVariableValue("--sloth-neutral");
    objectiveColor = getCSSVariableValue("--sloth-critical");

    // If width is 0, set it to container width.
    if (json.width === 0) {
        json.width = container.clientWidth;
    }
    
    const data = [
        json.timestamps,
        json.sli_values,
        sloLine
    ];

    const opts = {
        title: json.title,
        width: json.width,
        height: json.height,
        axes: [
            xAxisTimeUPlotConfig(light),
            yAxisPercentageUPlotConfig(light),
        ],
        series: [
            {},
            {
                label: "SLI",
                stroke: sliColor,
                width: 2,
                points: { show: false },
                lineInterpolation: 3,
                fill: sliColor + "1A",
                value: (u, v) => v == null ? "-" : v.toFixed(2) + "%",
            },
            {
                label: `Objective`,
                stroke: objectiveColor,
                width: 1,
                dash: [10, 5],
                points: { show: false },
                value: (u, v) => v == null ? "-" : v.toFixed(2) + "%",
            }
        ]
    };

    new uPlot(opts, data, document.getElementById(domElID));
}


function renderUPlotBudgetBurnChart(domElID, json) {
    const container = document.getElementById(domElID);
    const light = isColorThemeLight();
    let realBurnColor = getCSSVariableValue("--sloth-ok");
    const perfectBurnColor = getCSSVariableValue("--sloth-neutral");
    if (!json.color_line_ok) {
        realBurnColor = getCSSVariableValue("--sloth-critical");
    }

    // If width is 0, set it to container width.
    if (json.width === 0) {
        json.width = container.clientWidth;
    }
    
    
    const data = [
        json.timestamps,
        json.real_burned_values,
        json.perfect_burned_values,
    ];

    const opts = {
        title: json.title,
        width: json.width,
        height: json.height,
        axes: [
            xAxisTimeUPlotConfig(light),
            yAxisPercentageUPlotConfig(light),
        ],
        series: [
            {},
            {
                label: "Budget Burn",
                stroke: realBurnColor,
                width: 2,
                points: { show: false },
                lineInterpolation: 3,
                fill: rgbColorWithAlpha(realBurnColor, 0.1),
                value: (u, v) => v == null ? "-" : v.toFixed(2) + "%",
            },
            {
                label: `Perfect Budget Burn`,
                stroke: perfectBurnColor,
                width: 1,
                dash: [10, 5],
                points: { show: false },
                value: (u, v) => v == null ? "-" : v.toFixed(2) + "%",
            }
        ]
    };

    new uPlot(opts, data, document.getElementById(domElID));
}

// X axis (time).
// Got from https://github.com/prometheus/prometheus/blob/987b28e26ccaba6d39590b0dc55a430ae70b3716/web/ui/mantine-ui/src/pages/query/uPlotChartHelpers.ts#L334.
function xAxisTimeUPlotConfig(light) {
    return {
        space: 80,
        labelSize: 20,
        stroke: light ? "#333" : "#eee",
        ticks: {
            stroke: light ? "#00000010" : "#ffffff20",
        },
        grid: {
            show: true,
            stroke: light ? "#00000010" : "#ffffff20",
            width: 2,
            dash: [],
        },
        values: [
            // See https://github.com/leeoniya/uPlot/tree/master/docs#axis--grid-opts and https://github.com/leeoniya/uPlot/issues/83.
            //
            // We want to achieve 24h-based time formatting instead of the default AM/PM-based time formatting.
            // We also want to render dates in an unambiguous format that uses the abbreviated month name instead of a US-centric DD/MM/YYYY format.
            //
            // The "tick incr" column defines the breakpoint in seconds at which the format changes.
            // The "default" column defines the default format for a tick at this breakpoint.
            // The "year"/"month"/"day"/"hour"/"min"/"sec" columns define additional values to display for year/month/day/... rollovers occurring around a tick.
            // The "mode" column value "1" means that rollover values will be concatenated with the default format (instead of replacing it).
            //
            // tick incr        default                  year                  month  day             hour   min    sec    mode
            // prettier-ignore
            [3600 * 24 * 365,   "{YYYY}",                null,                 null,  null,           null,  null,  null,     1],
            // prettier-ignore
            [3600 * 24 * 28,    "{MMM}",                 "\n{YYYY}",           null,  null,           null,  null,  null,     1],
            // prettier-ignore
            [3600 * 24,         "{MMM} {D}",             "\n{YYYY}",           null,  null,           null,  null,  null,     1],
            // prettier-ignore
            [3600,              "{HH}:{mm}",             "\n{MMM} {D} '{YY}",  null,  "\n{MMM} {D}",  null,  null,  null,     1],
            // prettier-ignore
            [60,                "{HH}:{mm}",             "\n{MMM} {D} '{YY}",  null,  "\n{MMM} {D}",  null,  null,  null,     1],
            // prettier-ignore
            [1,                 "{HH}:{mm}:{ss}",        "\n{MMM} {D} '{YY}",  null,  "\n{MMM} {D}",  null,  null,  null,     1],
            // prettier-ignore
            [0.001,             "{HH}:{mm}:{ss}.{fff}",  "\n{MMM} {D} '{YY}",  null,  "\n{MMM} {D}",  null,  null,  null,     1],
        ],
    };
}

function yAxisPercentageUPlotConfig(light) {
    return {
        ticks: {
            stroke: light ? "#00000010" : "#ffffff20",
        },
        grid: {
            show: true,
            stroke: light ? "#00000010" : "#ffffff20",
            width: 2,
            dash: [],
        },
        stroke: light ? "#333" : "#eee",
        values: (u, vals) => vals.map(v => v.toFixed(2) + '%'),
        size: 90,
    };
}

function rgbColorWithAlpha(c, alpha) {
    if(c.indexOf('a') == -1){
        return c.replace(')', `, ${alpha})`).replace('rgb', 'rgba');
    }
    
    return c;
}