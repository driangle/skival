// sortTable sorts the rows of the table owning `th` by the column `th` heads.
// The column index is read from the header cell itself, so callers only need
// to pass the clicked element: onclick="sortTable(this)".
function sortTable(th) {
  var colIdx = th.cellIndex;
  var table = th.closest("table");
  var tbody = table.querySelector("tbody");
  var rows = Array.from(tbody.querySelectorAll("tr"));
  var asc = th.dataset.sortDir !== "asc";
  th.dataset.sortDir = asc ? "asc" : "desc";
  // Reset other headers in same table
  Array.from(table.querySelectorAll("th")).forEach(function(h) { if (h !== th) delete h.dataset.sortDir; });
  rows.sort(function(a, b) {
    var av = a.cells[colIdx].textContent.trim();
    var bv = b.cells[colIdx].textContent.trim();
    // Try numeric comparison (strip $, #, %, s, ms)
    var an = parseFloat(av.replace(/[$#%sms,]/g, ""));
    var bn = parseFloat(bv.replace(/[$#%sms,]/g, ""));
    if (!isNaN(an) && !isNaN(bn)) return asc ? an - bn : bn - an;
    return asc ? av.localeCompare(bv) : bv.localeCompare(av);
  });
  rows.forEach(function(r) { tbody.appendChild(r); });
}
