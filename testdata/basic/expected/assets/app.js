// Drag to pan ER diagrams with the mouse, like scrolling a map.
// A drag that actually moves suppresses the following click, so a simple click
// on a table still follows its link. Touch devices keep native scrolling.
(function () {
  "use strict";
  var threshold = 4; // px; smaller movements are treated as a click

  document.querySelectorAll(".diagram").forEach(function (el) {
    var start = null;
    var moved = false;

    el.addEventListener("pointerdown", function (e) {
      // Any new press starts fresh, so a click left suppressed by an earlier
      // drag never swallows the next one.
      moved = false;
      if (e.pointerType !== "mouse" || e.button !== 0) return;
      // Leave the native scrollbars alone (they lie outside the client area).
      if (e.target === el && (e.offsetX >= el.clientWidth || e.offsetY >= el.clientHeight)) return;
      start = { x: e.clientX, y: e.clientY, left: el.scrollLeft, top: el.scrollTop };
    });

    el.addEventListener("pointermove", function (e) {
      if (!start) return;
      // The button was released where this element could not see it (pointer
      // capture starts only after the threshold): stop instead of panning on hover.
      if ((e.buttons & 1) === 0) {
        end();
        return;
      }
      var dx = e.clientX - start.x;
      var dy = e.clientY - start.y;
      if (!moved) {
        if (Math.abs(dx) + Math.abs(dy) < threshold) return;
        moved = true;
        el.setPointerCapture(e.pointerId);
        el.classList.add("dragging");
      }
      el.scrollLeft = start.left - dx;
      el.scrollTop = start.top - dy;
    });

    function end() {
      start = null;
      el.classList.remove("dragging");
    }
    el.addEventListener("pointerup", end);
    el.addEventListener("pointercancel", end);

    // Capture phase so that the link inside never sees the click that ends a drag.
    // Keyboard activation (detail 0) is never a drag.
    el.addEventListener("click", function (e) {
      if (moved && e.detail !== 0) {
        e.preventDefault();
        e.stopPropagation();
      }
      moved = false;
    }, true);

    // Links and text would otherwise start the browser's own drag and drop.
    el.addEventListener("dragstart", function (e) {
      e.preventDefault();
    });
  });
})();

// Zoom ER diagrams: buttons over the frame and Ctrl/Cmd + wheel around the
// pointer. The SVG is resized through its width and height, so it stays sharp.
// Without JavaScript the diagrams stay at their natural size, with no buttons.
(function () {
  "use strict";
  // The buttons step through these scales, as browsers do for page zoom.
  var levels = [0.1, 0.25, 0.33, 0.5, 0.67, 0.75, 0.8, 0.9, 1, 1.1, 1.25, 1.5, 1.75, 2];
  var minScale = levels[0];
  var maxScale = levels[levels.length - 1];

  document.querySelectorAll(".diagram-frame").forEach(function (frame) {
    var el = frame.querySelector(".diagram");
    var svg = el && el.querySelector("svg");
    if (!svg || !svg.viewBox || !svg.viewBox.baseVal) return;
    // Graphviz sets width and height in pt to the viewBox size.
    var width = svg.viewBox.baseVal.width;
    var height = svg.viewBox.baseVal.height;
    if (!width || !height) return;
    var scale = 1;

    var bar = document.createElement("div");
    bar.className = "diagram-zoom";
    bar.setAttribute("role", "toolbar");
    bar.setAttribute("aria-label", "Zoom");
    // Two groups: stepping the scale, and fitting the frame.
    var group = addGroup();
    var zoomOut = button("−", "Zoom out");
    var level = button("100%", "Reset zoom to 100%");
    level.classList.add("zoom-level");
    var zoomIn = button("+", "Zoom in");
    group = addGroup();
    var fit = button("Fit", "Fit the diagram in the frame");
    var fitWidth = button("Fit width", "Fit the width of the diagram in the frame");
    frame.appendChild(bar);

    function button(text, label) {
      var b = document.createElement("button");
      b.type = "button";
      b.textContent = text;
      b.title = label;
      b.setAttribute("aria-label", label);
      group.appendChild(b);
      return b;
    }

    function addGroup() {
      var g = document.createElement("div");
      g.className = "zoom-group";
      bar.appendChild(g);
      return g;
    }

    // fitScale is the scale at which the whole diagram fits in the frame, at
    // most 100% (a small diagram is not enlarged). 1pt is 4/3 px.
    // The frame of a table diagram shrinks with it, so its height limit is
    // used instead of its current height.
    function fitScale() {
      var w = el.clientWidth, h = el.clientHeight;
      var max = parseFloat(getComputedStyle(el).maxHeight);
      if (max > 0) h = max - el.clientTop * 2;
      if (!w || !h) return 1;
      return floor(Math.min(1, w / (width * 4 / 3), h / (height * 4 / 3)));
    }

    // floor rounds a fitting scale down to the precision zoomTo keeps, so that
    // rounding never makes the diagram overflow by a pixel.
    function floor(s) {
      return Math.floor(s * 1000) / 1000;
    }

    // zoomTo sets the scale, keeping the diagram point under (x, y), relative
    // to the frame's client area, in place; the center by default.
    function zoomTo(s, x, y) {
      s = Math.max(Math.min(minScale, fitScale()), Math.min(maxScale, s));
      s = Math.round(s * 1000) / 1000;
      if (x === undefined) {
        x = el.clientWidth / 2;
        y = el.clientHeight / 2;
      }
      var ratio = s / scale;
      var left = (el.scrollLeft + x) * ratio - x;
      var top = (el.scrollTop + y) * ratio - y;
      scale = s;
      svg.setAttribute("width", width * s + "pt");
      svg.setAttribute("height", height * s + "pt");
      el.scrollLeft = left;
      el.scrollTop = top;
      level.textContent = Math.round(s * 100) + "%";
      zoomOut.disabled = s <= Math.min(minScale, fitScale());
      zoomIn.disabled = s >= maxScale;
      placeBar();
    }

    // Keep the buttons clear of the vertical scrollbar, which comes and goes
    // with the zoom (overlay scrollbars take no space).
    function placeBar() {
      var scrollbar = el.offsetWidth - el.clientWidth - el.clientLeft * 2;
      bar.style.right = 8 + Math.max(0, scrollbar) + "px";
    }

    // The wheel leaves the scale between levels; the buttons go to the next one.
    // Below the lowest level, a very large diagram can still zoom out to the fit.
    zoomOut.addEventListener("click", function () {
      var next = Math.min(minScale, fitScale());
      levels.forEach(function (l) { if (l < scale - 0.001) next = Math.max(next, l); });
      zoomTo(next);
    });
    zoomIn.addEventListener("click", function () {
      var next = maxScale;
      levels.forEach(function (l) { if (l > scale + 0.001) next = Math.min(next, l); });
      zoomTo(next);
    });
    level.addEventListener("click", function () { zoomTo(1); });
    fit.addEventListener("click", function () {
      zoomTo(fitScale());
      el.scrollLeft = 0;
      el.scrollTop = 0;
    });
    // Fit the width only, for a tall diagram to read by scrolling down. A
    // vertical scrollbar that appears takes some width, so fit again once.
    fitWidth.addEventListener("click", function () {
      var widthScale = function () { return floor(Math.min(1, el.clientWidth / (width * 4 / 3))); };
      zoomTo(widthScale());
      if (el.scrollWidth > el.clientWidth) zoomTo(widthScale());
      el.scrollLeft = 0;
      el.scrollTop = 0;
    });

    // Ctrl/Cmd + wheel (and pinch on trackpads, which browsers report as
    // Ctrl + wheel) zooms; the plain wheel keeps scrolling.
    el.addEventListener("wheel", function (e) {
      if (!e.ctrlKey && !e.metaKey) return;
      e.preventDefault();
      var dy = e.deltaY * (e.deltaMode === 1 ? 33 : e.deltaMode === 2 ? el.clientHeight : 1);
      var rect = el.getBoundingClientRect();
      zoomTo(scale * Math.exp(-dy * 0.002), e.clientX - rect.left - el.clientLeft, e.clientY - rect.top - el.clientTop);
    }, { passive: false });

    // A hidden view (another tab on the relationships page) has no size yet.
    if (window.ResizeObserver) {
      new ResizeObserver(placeBar).observe(el);
    }
    placeBar();
  });
})();
