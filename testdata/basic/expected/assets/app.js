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
