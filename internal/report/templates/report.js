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

  // detailOf returns the error-detail row that follows a sample row, or null.
  function detailOf(row) {
    var n = row.nextElementSibling;
    return n && n.classList.contains("err-detail") ? n : null;
  }

  // toggleRow expands or collapses the error detail beneath an errored run.
  // The `.expanded` class carries the user's intent so a later filter can
  // restore it, independent of the `hidden` attribute the filter also uses.
  window.toggleRow = function (row) {
    var d = detailOf(row);
    if (!d) return;
    var open = d.classList.toggle("expanded");
    if (open) { d.removeAttribute("hidden"); } else { d.setAttribute("hidden", ""); }
    var caret = row.querySelector(".rowcaret");
    if (caret) caret.textContent = open ? "\u25be" : "\u25b8";
  };

  // filterVariant shows only rows for the clicked variant across every eval
  // table; an empty value clears the filter. A detail row is shown only when
  // its variant matches AND its parent is currently expanded.
  window.filterVariant = function (btn) {
    var want = btn.dataset.variant || "";
    var chips = document.querySelectorAll(".chip");
    for (var i = 0; i < chips.length; i++) chips[i].classList.toggle("active", chips[i] === btn);
    var rows = document.querySelectorAll("tr[data-variant]:not(.err-detail)");
    for (var j = 0; j < rows.length; j++) {
      var hit = !want || rows[j].dataset.variant === want;
      if (hit) { rows[j].removeAttribute("hidden"); } else { rows[j].setAttribute("hidden", ""); }
      var d = detailOf(rows[j]);
      if (d) {
        if (hit && d.classList.contains("expanded")) { d.removeAttribute("hidden"); }
        else { d.setAttribute("hidden", ""); }
      }
    }
  };

  // sortTable sorts the rows of the table owning `th` by the column it heads.
  // Each sample row carries its error-detail row along so the two never split.
  window.sortTable = function (th) {
    var colIdx = th.cellIndex;
    var table = th.closest("table");
    var tbody = table.querySelector("tbody");
    var all = Array.prototype.slice.call(tbody.querySelectorAll("tr"));
    var units = [];
    for (var i = 0; i < all.length; i++) {
      if (all[i].classList.contains("err-detail")) continue;
      units.push([all[i], detailOf(all[i])]);
    }
    var asc = th.dataset.sortDir !== "asc";
    th.dataset.sortDir = asc ? "asc" : "desc";
    var heads = table.querySelectorAll("th");
    for (var k = 0; k < heads.length; k++) { if (heads[k] !== th) delete heads[k].dataset.sortDir; }
    units.sort(function (a, b) {
      var ac = a[0].cells[colIdx], bc = b[0].cells[colIdx];
      var av = ac ? ac.textContent.trim() : "";
      var bv = bc ? bc.textContent.trim() : "";
      var an = parseFloat(av.replace(/[$#%sms,\u2013\u2014]/g, ""));
      var bn = parseFloat(bv.replace(/[$#%sms,\u2013\u2014]/g, ""));
      if (!isNaN(an) && !isNaN(bn)) return asc ? an - bn : bn - an;
      return asc ? av.localeCompare(bv) : bv.localeCompare(av);
    });
    units.forEach(function (u) { tbody.appendChild(u[0]); if (u[1]) tbody.appendChild(u[1]); });
  };

  initTheme();
})();
