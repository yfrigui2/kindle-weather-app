/* @ds-bundle: {"format":4,"namespace":"PocketInk","components":[{"name":"Screen"},{"name":"StatusBar"},{"name":"Window"},{"name":"Readout"},{"name":"Forecast"},{"name":"Stat"},{"name":"Meter"},{"name":"Chip"},{"name":"Button"},{"name":"PixelIcon"}]} */
(function () {
  var React = window.React;
  var h = React.createElement;

  /* 16 x 16 pixel maps. k = ink, w = paper, g = fog, m = mist, . = clear */
  var ICONS = {"sun":["................",".......kk.......","..k....kk....k..","...k........k...","......kkkk......",".....kwwwwk.....","....kwwwwwwk....",".kk.kwkwwkwk.kk.",".kk.kgwwwwgk.kk.","....kwwkkwwk....",".....kwwwwk.....","......kkkk......","...k........k...","..k....kk....k..",".......kk.......","................"],"partly":["................","................","....kk..........",".k......k.......","...kkkk.........","..kwwwwk........",".kwwwwwwk.......",".kwwwwwwkkkkk...","..kwwwwkkwwwwk..","...kkkkkkwwwwwk.",".k...kwwwwwwwwwk","....kwwwkwwkwwwk","....kwwgwkkwgwwk","....kwwwwwwwwwwk",".....kkkkkkkkkk.","................"],"cloud":["................","................","................","........kkkk....",".......kwwwwk...","...kkk.kwwwwwk..","..kwwwkwwwwwwwk.",".kwwwwwwwwwwwwk.",".kwwwwkwwwwkwwwk","kwwwwwgwkkwgwwwk","kwwwwwwwwwwwwwwk","kwwwwwwwwwwwwwwk",".kkkkkkkkkkkkkk.","................","................","................"],"rain":["................","........kkkk....",".......kwwwwk...","...kkk.kwwwwwk..","..kwwwkwwwwwwwk.",".kwwwwwwwwwwwwk.",".kwwwwkwwwwkwwwk","kwwwwwgwkkwgwwwk","kwwwwwwwwwwwwwwk",".kkkkkkkkkkkkkk.","................","...k....k....k..","..k....k....k...","................",".....k....k.....","....k....k......"],"storm":["................","........kkkk....",".......kggggk...","...kkk.kgggggk..","..kgggkgggggggk.",".kggggggggggggk.",".kggggkggggkgggk","kgggggkgkkgkgggk","kggggggggggggggk",".kkkkkkkkkkkkkk.","........kk......",".......kk.......","......kkkkk.....",".........kk.....","........kk......","........k......."],"snow":["................","........kkkk....",".......kwwwwk...","...kkk.kwwwwwk..","..kwwwkwwwwwwwk.",".kwwwwwwwwwwwwk.",".kwwwwkwwwwkwwwk","kwwwwwgwkkwgwwwk","kwwwwwwwwwwwwwwk",".kkkkkkkkkkkkkk.","................","..k.k.....k.k...","...k.......k....","..k.k..k..k.k...","......k.k.......",".......k........"],"fog":["................","................","................","..kkkkkkkk......","................",".....kkkkkkkkk..","................",".kkkkkkk..kkkk..","................","....kkkkkkkkkkk.","................","..kkkk..kkkkkk..","................","......kkkkkk....","................","................"],"wind":["................","................","..........kk....",".........k..k...",".............k..",".kkkkkkkkkkkk...","................","..kkkkkkkkkkkkk.","...............k","..kkkkkkkk....k.","..........kkkk..","................",".....kkkkk......","..........k.....",".........k......","................"],"moon":["................","................","....kkkk....k...","...kwwk....kkk..","..kwwwk.....k...",".kwwwwk.........",".kwwwwk.........",".kwwwwk.......k.",".kwkwkwk........",".kwwkwwk........",".kwgwwwwkk......",".kwwwwwwwwkkkk..","..kwwwwwwwwwk...","...kwwwwwwwk....","....kkkkkkk.....","................"],"drop":["................",".......kk.......",".......kk.......","......kwwk......","......kwwk......",".....kwwwwk.....",".....kwwwwk.....","....kwwwwwwk....","...kwwwwwwwwk...","...kwkwwwwkwk...","...kwkwwwwkwk...","...kgwwkkwwgk...","...kwwwwwwwwk...","....kwwwwwwk....",".....kkkkkk.....","................"],"umbrella":["................",".......kk.......",".....kkkkkk.....","...kkwwwwwwkk...","..kwwwwwwwwwwk..",".kwwwwwwwwwwwwk.",".kwwwwwwwwwwwwk.","kwwwwwwwwwwwwwwk","kwkkwkkwwkkwkkwk",".k..k..kk..k..k.",".......kk.......",".......kk.......",".......kk.......",".......kk..kk...","........kkkk....","................"],"thermometer":["................","......kkkk......",".....kwwwwk.....",".....kwwwwk.kk..",".....kwwwwk.....",".....kwwwwk.kk..",".....kwkkwk.....",".....kwkkwk.kk..",".....kwkkwk.....","....kwwkkwwk....","...kwwkkkkwwk...","...kwkkkkkkwk...","...kwkkkkkkwk...","...kwwkkkkwwk...","....kwwwwwwk....",".....kkkkkk....."],"sunrise":[".......kk.......","......kkkk......",".....kkkkkk.....",".......kk.......",".......kk.......","..k.........k...","...k..kkkk.k....",".....kwwwwk.....","....kwwwwwwk....",".kk.kwwwwwwk.kk.","...kwwwwwwwwk...","kkkkkkkkkkkkkkkk","................","..kkkkk..kkkkk..","................","................"],"sunset":[".......kk.......",".......kk.......",".....kkkkkk.....","......kkkk......",".......kk.......","..k.........k...","...k..kkkk.k....",".....kwwwwk.....","....kwwwwwwk....",".kk.kwwwwwwk.kk.","...kwwwwwwwwk...","kkkkkkkkkkkkkkkk","................","..kkkkk..kkkkk..","................","................"],"heart":["................","................","..kkkk....kkkk..",".kwwwwk..kwwwwk.","kwwggwwkkwwwwwwk","kwggwwwwwwwwwwwk","kwgwwwwwwwwwwwwk","kwwwwwwwwwwwwwwk",".kwwwwwwwwwwwwk.","..kwwwwwwwwwwk..","...kwwwwwwwwk...","....kwwwwwwk....",".....kwwwwk.....","......kwwk......",".......kk.......","................"],"sparkle":["................",".......kk.......",".......kk.......","......kwwk......","......kwwk......",".....kwwwwk.....","..kkkwwwwwwkkk..","kkwwwwwwwwwwwwkk","kkwwwwwwwwwwwwkk","..kkkwwwwwwkkk..",".....kwwwwk.....","......kwwk......","......kwwk......",".......kk.......",".......kk.......","................"],"next":["................","................",".....k..........",".....kk.........",".....kkk........",".....kkkk.......",".....kkkkk......",".....kkkkkk.....",".....kkkkkk.....",".....kkkkk......",".....kkkk.......",".....kkk........",".....kk.........",".....k..........","................","................"],"back":["................","................","..........k.....",".........kk.....","........kkk.....",".......kkkk.....","......kkkkk.....",".....kkkkkk.....",".....kkkkkk.....","......kkkkk.....",".......kkkk.....","........kkk.....",".........kk.....","..........k.....","................","................"]};
  var SIZES = { xs: 16, sm: 32, md: 64, lg: 128 };
  var CLOSE = ["k....k", ".k..k.", "..kk..", "..kk..", ".k..k.", "k....k"];

  function cx() {
    var out = [];
    for (var i = 0; i < arguments.length; i++) if (arguments[i]) out.push(arguments[i]);
    return out.join(" ");
  }

  function rects(rows, map) {
    var out = [];
    for (var y = 0; y < rows.length; y++) {
      var r = rows[y], x = 0;
      while (x < r.length) {
        var c = r.charAt(x);
        if (c === ".") { x++; continue; }
        var s = x;
        while (x < r.length && r.charAt(x) === c) x++;
        out.push(h("rect", { key: y + "-" + s, x: s, y: y, width: x - s, height: 1, className: "pk-i-" + (map[c] || c) }));
      }
    }
    return out;
  }

  function PixelIcon(p) {
    var rows = ICONS[p.name];
    if (!rows) return null;
    var px = typeof p.size === "number" ? p.size : (SIZES[p.size || "md"] || 64);
    var map = p.invert ? { k: "w", w: "k" } : (p.filled ? { w: "k" } : {});
    var label = p.title || null;
    return h("svg", {
      className: cx("pk-icon", p.className),
      viewBox: "0 0 16 16", width: px, height: px, shapeRendering: "crispEdges",
      role: label ? "img" : undefined, "aria-label": label || undefined, "aria-hidden": label ? undefined : true,
      style: p.style
    }, rects(rows, map));
  }
  PixelIcon.names = Object.keys(ICONS);

  function Screen(p) {
    return h("div", { className: cx("pk-screen", p.grid === false && "pk-screen--plain", p.className), style: p.style }, p.children);
  }

  function StatusBar(p) {
    var hearts = null;
    if (typeof p.battery === "number") {
      var n = p.hearts || 4;
      var level = Math.max(0, Math.min(100, p.battery));
      var full = Math.min(n, Math.ceil(level / (100 / n)));
      var list = [];
      for (var i = 0; i < n; i++) list.push(h(PixelIcon, { key: i, name: "heart", size: "xs", filled: i < full }));
      hearts = h("span", { className: "pk-status__hearts", role: "img", "aria-label": "battery " + level + "%" }, list);
    }
    return h("div", { className: cx("pk-status", p.className), style: p.style },
      h("span", { className: "pk-status__left" }, p.left),
      h("span", { className: "pk-status__right" }, p.right != null ? h("span", null, p.right) : null, hearts));
  }

  function controls(kind) {
    if (kind === "none") return null;
    if (kind === "close") {
      return h("span", { className: "pk-window__close", "aria-hidden": true },
        h("svg", { viewBox: "0 0 6 6", width: 12, height: 12, shapeRendering: "crispEdges" }, rects(CLOSE, {})));
    }
    return h("span", { className: "pk-window__dots", "aria-hidden": true }, h("i"), h("i"), h("i"));
  }

  function Window(p) {
    return h("section", { className: cx("pk-window", p.tone === "alert" && "pk-window--alert", p.grow && "pk-window--grow", p.className), style: p.style },
      h("header", { className: "pk-window__bar" },
        h("span", { className: "pk-window__title" }, p.title),
        controls(p.controls || "dots")),
      h("div", { className: cx("pk-window__body", p.flush && "pk-window__body--flush") }, p.children));
  }

  function Readout(p) {
    var unit = p.unit == null ? "°" : p.unit;
    return h("div", { className: cx("pk-readout", p.size === "md" && "pk-readout--md", p.className), style: p.style },
      p.label ? h("div", { className: "pk-readout__label" }, p.label) : null,
      h("div", { className: "pk-readout__value" }, p.value, unit ? h("span", { className: "pk-readout__unit" }, unit) : null),
      p.sub ? h("div", { className: "pk-readout__sub" }, p.sub) : null);
  }

  function Forecast(p) {
    var days = p.days || [];
    var rows = p.layout === "rows";
    var today = p.today == null ? 0 : p.today;
    var unit = p.unit == null ? "°" : p.unit;
    var style = rows ? {} : { gridTemplateColumns: "repeat(" + Math.max(1, days.length) + ", minmax(0, 1fr))" };
    if (p.style) for (var k in p.style) style[k] = p.style[k];
    return h("div", { className: cx("pk-forecast", rows && "pk-forecast--rows", p.className), style: style },
      days.map(function (d, i) {
        var on = i === today;
        var hi = String(d.hi) + unit;
        var rain = d.rain != null && d.rain >= (p.rainFrom == null ? 10 : p.rainFrom);
        return h("div", { key: i, className: cx("pk-forecast__day", on && "is-today") },
          h("span", { className: "pk-forecast__label" }, d.label),
          h(PixelIcon, { name: d.icon, size: p.iconSize || "md", invert: on, title: d.word || d.icon, className: "pk-forecast__icon" }),
          rows ? h("span", { className: "pk-forecast__word" }, d.word || "") : null,
          h("span", { className: cx("pk-forecast__hi", hi.length > 3 && "is-long") }, hi),
          d.lo != null ? h("span", { className: "pk-forecast__lo" }, String(d.lo) + unit) : null,
          rain ? h("span", { className: "pk-forecast__rain", "aria-label": d.rain + "% chance of rain" },
            h(PixelIcon, { name: "drop", size: "xs", invert: on }), d.rain + "%")
               : h("span", { className: "pk-forecast__rain is-empty", "aria-hidden": true }));
      }));
  }

  function Stat(p) {
    return h("div", { className: cx("pk-stat", p.className), style: p.style },
      p.icon ? h(PixelIcon, { name: p.icon, size: "sm" }) : null,
      h("div", { className: "pk-stat__text" },
        h("div", { className: "pk-stat__label" }, p.label),
        h("div", { className: "pk-stat__value" }, p.value, p.unit ? h("span", { className: "pk-stat__unit" }, p.unit) : null)));
  }

  function Meter(p) {
    var max = p.max || 100, n = p.segments || 10;
    var v = Math.max(0, Math.min(max, p.value || 0));
    var on = Math.round((v / max) * n);
    var segs = [];
    for (var i = 0; i < n; i++) segs.push(h("i", { key: i, className: i < on ? "is-on" : undefined }));
    var shown = p.display != null ? p.display : Math.round((v / max) * 100) + "%";
    return h("div", {
      className: cx("pk-meter", p.className), style: p.style,
      role: "meter", "aria-valuemin": 0, "aria-valuemax": max, "aria-valuenow": v, "aria-label": p.label
    },
      p.label != null || p.display !== false
        ? h("div", { className: "pk-meter__head" },
            h("span", { className: "pk-meter__label" }, p.label),
            p.display === false ? null : h("span", { className: "pk-meter__value" }, shown))
        : null,
      h("div", { className: "pk-meter__track" }, segs));
  }

  function Chip(p) {
    var ink = p.tone === "ink";
    return h("span", { className: cx("pk-chip", ink && "pk-chip--ink", p.className), style: p.style },
      p.icon ? h(PixelIcon, { name: p.icon, size: "xs", invert: ink }) : null,
      p.children);
  }

  function Button(p) {
    var rest = {};
    for (var k in p) if (k !== "variant" && k !== "icon" && k !== "iconEnd" && k !== "children" && k !== "className") rest[k] = p[k];
    var primary = p.variant === "primary";
    rest.className = cx("pk-key", primary && "pk-key--primary", p.className);
    if (!rest.type) rest.type = "button";
    var icon = p.icon ? h(PixelIcon, { name: p.icon, size: "xs", invert: primary }) : null;
    return h("button", rest, p.iconEnd ? null : icon, p.children, p.iconEnd ? icon : null);
  }

  var api = {
    Screen: Screen, StatusBar: StatusBar, Window: Window, Readout: Readout, Forecast: Forecast,
    Stat: Stat, Meter: Meter, Chip: Chip, Button: Button, PixelIcon: PixelIcon
  };
  window.PocketInk = window.PocketInk || {};
  for (var name in api) window.PocketInk[name] = api[name];
})();
