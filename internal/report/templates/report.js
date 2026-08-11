// Report interactions. No dependencies: the report is a single self-contained
// file that must work from disk, offline, and inside CI artifact viewers.

(function () {
  var THEME_KEY = "skival-report-theme";

  function applyTheme(t) {
    document.documentElement.setAttribute("data-theme", t);
    var btn = document.getElementById("theme-toggle");
    if (btn) btn.textContent = t === "dark" ? "Light" : "Dark";
  }

  function initTheme() {
    var stored = null;
    try { stored = localStorage.getItem(THEME_KEY); } catch (e) { /* file:// or private mode */ }
    var prefersDark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
    applyTheme(stored || (prefersDark ? "dark" : "light"));
  }

  window.toggleTheme = function () {
    var next = document.documentElement.getAttribute("data-theme") === "dark" ? "light" : "dark";
    applyTheme(next);
    try { localStorage.setItem(THEME_KEY, next); } catch (e) { /* ignore */ }
  };

  window.printReport = function () { window.print(); };

  // toggleEval expands or collapses one eval card.
  window.toggleEval = function (btn) {
    var body = btn.parentNode.querySelector(".eval-body");
    var open = body.hasAttribute("hidden");
    if (open) { body.removeAttribute("hidden"); } else { body.setAttribute("hidden", ""); }
    btn.querySelector(".caret").textContent = open ? "\u25be" : "\u25b8";
  };

  // toggleVerdict expands or collapses one judge rationale.
  window.toggleVerdict = function (btn) {
    var body = btn.parentNode.querySelector(".reason");
    var open = body.hasAttribute("hidden");
    if (open) { body.removeAttribute("hidden"); } else { body.setAttribute("hidden", ""); }
    btn.querySelector(".caret").textContent = open ? "\u25be" : "\u25b8";
  };

  // filterVariant shows only rows for the clicked variant across every eval
  // table; an empty value clears the filter.
  window.filterVariant = function (btn) {
    var want = btn.dataset.variant || "";
    var chips = document.querySelectorAll(".chip");
    for (var i = 0; i < chips.length; i++) chips[i].classList.toggle("active", chips[i] === btn);
    var rows = document.querySelectorAll("tr[data-variant]");
    for (var j = 0; j < rows.length; j++) {
      var hit = !want || rows[j].dataset.variant === want;
      if (hit) { rows[j].removeAttribute("hidden"); } else { rows[j].setAttribute("hidden", ""); }
    }
  };

  // sortTable sorts the rows of the table owning `th` by the column `th` heads.
  // Aggregate rows sort with their samples' values but are keyed to stay
  // adjacent to their variant when sorting by variant name.
  window.sortTable = function (th) {
    var colIdx = th.cellIndex;
    var table = th.closest("table");
    var tbody = table.querySelector("tbody");
    var rows = Array.prototype.slice.call(tbody.querySelectorAll("tr"));
    var asc = th.dataset.sortDir !== "asc";
    th.dataset.sortDir = asc ? "asc" : "desc";
    var heads = table.querySelectorAll("th");
    for (var i = 0; i < heads.length; i++) { if (heads[i] !== th) delete heads[i].dataset.sortDir; }
    rows.sort(function (a, b) {
      var av = a.cells[colIdx].textContent.trim();
      var bv = b.cells[colIdx].textContent.trim();
      var an = parseFloat(av.replace(/[$#%sms,\u2013\u2014]/g, ""));
      var bn = parseFloat(bv.replace(/[$#%sms,\u2013\u2014]/g, ""));
      if (!isNaN(an) && !isNaN(bn)) return asc ? an - bn : bn - an;
      return asc ? av.localeCompare(bv) : bv.localeCompare(av);
    });
    rows.forEach(function (r) { tbody.appendChild(r); });
  };

  initTheme();
})();
