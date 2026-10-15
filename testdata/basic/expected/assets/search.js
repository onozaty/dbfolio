// Search over tables and columns, from the index in search-index.js.
// The index is a script assigning a global, not JSON loaded with fetch(),
// because pages opened via file:// cannot fetch local files.
(function () {
  "use strict";

  var maxResults = 20;

  // prepare lowercases the index once, so that each keystroke only compares.
  function prepare(index) {
    return index.map(function (e, i) {
      var name = e.n.toLowerCase();
      return {
        entry: e,
        order: i,
        name: name,
        // The last dotted part, so that "users" is a prefix of "public.users".
        last: name.slice(name.lastIndexOf(".") + 1),
        // A column also matches with its table ("users.id").
        full: e.k === "c" ? (e.t + "." + e.n).toLowerCase() : name,
        comment: (e.c || "").toLowerCase(),
      };
    });
  }

  // search returns the entries matching every space-separated term of query
  // (case-insensitive substring), best first:
  //   0. table name starting with the first term
  //   1. table name containing the terms
  //   2. column name starting with the first term
  //   3. column name containing the terms
  //   4. comment containing the terms
  // Ties keep the shorter name first, then the index order.
  function search(prepared, query, limit) {
    var terms = query.toLowerCase().split(/\s+/).filter(Boolean);
    if (terms.length === 0) return [];
    var hits = [];
    for (var i = 0; i < prepared.length; i++) {
      var p = prepared[i];
      var inName = true;
      var inAny = true;
      for (var j = 0; j < terms.length; j++) {
        if (p.full.indexOf(terms[j]) < 0) {
          inName = false;
          if (p.comment.indexOf(terms[j]) < 0) {
            inAny = false;
            break;
          }
        }
      }
      if (!inAny) continue;
      var rank = 4;
      if (inName) {
        var prefix = startsWith(p.name, terms[0]) || startsWith(p.last, terms[0]);
        rank = (p.entry.k === "t" ? 0 : 2) + (prefix ? 0 : 1);
      }
      hits.push({ p: p, rank: rank });
    }
    hits.sort(function (a, b) {
      return a.rank - b.rank || a.p.full.length - b.p.full.length || a.p.order - b.p.order;
    });
    return hits.slice(0, limit || maxResults).map(function (h) { return h.p.entry; });
  }

  function startsWith(s, prefix) {
    return s.lastIndexOf(prefix, 0) === 0;
  }

  if (typeof module !== "undefined") {
    module.exports = { prepare: prepare, search: search };
  }
  if (typeof document === "undefined") return;

  // Loaded with defer: the document is parsed and search-index.js has run.
  var box = document.querySelector(".search");
  var input = document.getElementById("search-input");
  var list = document.getElementById("search-results");
  var status = document.getElementById("search-status");
  if (!box || !input || !list || !status || !window.DBFOLIO_SEARCH_INDEX) return;
  var index = prepare(window.DBFOLIO_SEARCH_INDEX);
  // Index links are relative to the output root; pages in tables/ are one level down.
  var root = document.documentElement.getAttribute("data-root") || "";
  var results = [];
  var active = -1;

  // Without JavaScript the box would do nothing, so it is shown only now.
  box.hidden = false;

  // close hides the list without removing its items: it also runs while a
  // clicked result is following its link, which a removed link would not.
  function close() {
    list.hidden = true;
    status.textContent = "";
    input.setAttribute("aria-expanded", "false");
    input.removeAttribute("aria-activedescendant");
    results = [];
    active = -1;
  }

  function setActive(i) {
    var items = list.children;
    if (active >= 0 && items[active]) {
      items[active].classList.remove("active");
      items[active].setAttribute("aria-selected", "false");
    }
    active = i;
    if (active >= 0 && items[active]) {
      items[active].classList.add("active");
      items[active].setAttribute("aria-selected", "true");
      items[active].scrollIntoView({ block: "nearest" });
      input.setAttribute("aria-activedescendant", items[active].id);
    } else {
      input.removeAttribute("aria-activedescendant");
    }
  }

  function render() {
    if (input.value.trim() === "") {
      close();
      return;
    }
    results = search(index, input.value);
    list.textContent = "";
    active = -1;
    results.forEach(function (e, i) {
      var li = document.createElement("li");
      li.id = "search-result-" + i;
      li.setAttribute("role", "option");
      li.setAttribute("aria-selected", "false");
      var a = document.createElement("a");
      a.href = root + e.h;
      a.tabIndex = -1;
      // textContent, never innerHTML: names and comments come from the database.
      appendSpan(a, "search-name", e.k === "c" ? e.t + "." + e.n : e.n);
      if (e.y) appendSpan(a, "search-type", e.k === "t" ? "[" + e.y + "]" : e.y);
      if (e.c) appendSpan(a, "search-comment", e.c);
      li.appendChild(a);
      list.appendChild(li);
    });
    // The count is announced by screen readers (aria-live) and shown for no matches.
    status.textContent = results.length === 0 ? "No matches" :
      results.length === 1 ? "1 result" : results.length + " results";
    status.classList.toggle("search-empty", results.length === 0);
    list.hidden = false;
    input.setAttribute("aria-expanded", "true");
    if (results.length > 0) setActive(0);
  }

  function appendSpan(parent, className, text) {
    var span = document.createElement("span");
    span.className = className;
    span.textContent = text;
    parent.appendChild(span);
  }

  input.addEventListener("input", render);
  input.addEventListener("focus", function () {
    if (input.value.trim() !== "") render();
  });
  input.addEventListener("click", function () {
    // A click on the already focused box fires no focus event.
    if (list.hidden && input.value.trim() !== "") render();
  });
  input.addEventListener("keydown", function (e) {
    // Keys during IME composition (e.g. Enter to confirm a conversion) are not commands.
    if (e.isComposing || e.keyCode === 229) return;
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      if (list.hidden) {
        render();
        return;
      }
      if (results.length === 0) return;
      var step = e.key === "ArrowDown" ? 1 : -1;
      setActive((active + step + results.length) % results.length);
    } else if (e.key === "Enter") {
      if (!list.hidden && active >= 0 && results[active]) {
        e.preventDefault();
        var href = root + results[active].h;
        close();
        window.location.href = href;
      }
    } else if (e.key === "Escape") {
      close();
      input.blur();
    }
  });
  // Keep the focus in the box while clicking a result (some browsers do not
  // focus links on click), so the list stays until the click follows the link.
  list.addEventListener("mousedown", function (e) {
    e.preventDefault();
  });
  // A result on the current page only changes the hash; close the list anyway.
  list.addEventListener("click", close);
  input.addEventListener("blur", close);
  document.addEventListener("click", function (e) {
    if (!box.contains(e.target)) close();
  });
  // "/" jumps to the search box, as on many documentation sites.
  document.addEventListener("keydown", function (e) {
    if (e.key !== "/" || e.ctrlKey || e.metaKey || e.altKey || e.isComposing) return;
    var t = e.target;
    if (t && (t.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName))) return;
    e.preventDefault();
    input.focus();
    input.select();
  });
})();
