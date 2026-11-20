export async function measurePresentation(page) {
  return page.evaluate(() => {
    const rgb = value => {
      if (!/^rgba?\(/.test(value)) return null;
      const parts = value.match(/[\d.]+/g)?.map(Number);
      return parts?.length >= 3 ? [...parts.slice(0, 3), parts[3] ?? 1] : null;
    };
    const over = (a, b) => [0, 1, 2].map(i => a[i] * a[3] + b[i] * (1 - a[3]));
    const luminance = color => color.map(v => { v /= 255; return v <= .04045 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4; }).reduce((sum, v, i) => sum + v * [.2126, .7152, .0722][i], 0);
    const contrast = (a, b) => (Math.max(luminance(a), luminance(b)) + .05) / (Math.min(luminance(a), luminance(b)) + .05);
    const identity = e => ({ tag: e.tagName, id: e.id, className: typeof e.className === "string" ? e.className : "", type: e.getAttribute("type") });
    const shown = e => {
      if (!e.getClientRects().length || e.closest(".visually-hidden, [hidden], [inert]")) return false;
      const r = e.getBoundingClientRect(), c = getComputedStyle(e);
      return r.width > 1 && r.height > 1 && c.visibility === "visible";
    };
    const background = e => {
      const ancestors = [], uncertainty = [];
      for (let n = e; n; n = n.parentElement) ancestors.unshift(n);
      let color = [255, 255, 255];
      for (const n of ancestors) {
        const c = getComputedStyle(n), parsed = rgb(c.backgroundColor);
        if (!parsed) uncertainty.push("unsupported-background-color");
        else color = over(parsed, color);
        if (c.backgroundImage !== "none") uncertainty.push("background-image-or-gradient");
        if (Number(c.opacity) !== 1) uncertainty.push("group-opacity");
        if (c.filter !== "none" || c.backdropFilter !== "none") uncertainty.push("filter");
        if (c.mixBlendMode !== "normal") uncertainty.push("blend-mode");
      }
      return { color, uncertainty: [...new Set(uncertainty)] };
    };
    const results = [], excluded = [], modal = document.querySelector("dialog:modal");
    const measure = (element, kind, foreground, bg, threshold, additional = []) => {
      const parsed = rgb(foreground), uncertainty = [...bg.uncertainty, ...additional];
      if (!parsed) uncertainty.push("unsupported-foreground-color");
      const effective = parsed ? over(parsed, bg.color) : null;
      const ratio = effective ? contrast(effective, bg.color) : null;
      results.push({ element: identity(element), kind, foreground, effective, background: bg.color, ratio, threshold, uncertainty: [...new Set(uncertainty)], candidate: uncertainty.length === 0 && ratio !== null && ratio < threshold });
    };
    for (const e of document.querySelectorAll("main input, main textarea, main select, main button, dialog[open] input, dialog[open] textarea, dialog[open] select, dialog[open] button")) {
      if (!shown(e)) continue;
      if (modal && !modal.contains(e)) { excluded.push({ element: identity(e), reason: "modal-underlay" }); continue; }
      if (e.closest(".review-heading, .review-panel") || e.matches(":disabled, [aria-disabled=true]")) {
        excluded.push({ element: identity(e), reason: e.matches(":disabled, [aria-disabled=true]") ? "inactive" : "owner-review" }); continue;
      }
      const c = getComputedStyle(e), bg = background(e), native = e.matches('input[type="checkbox"], input[type="radio"], input[type="range"], input[type="color"], select');
      const large = parseFloat(c.fontSize) >= 24 || parseFloat(c.fontSize) >= 18.6667 && parseInt(c.fontWeight) >= 700;
      const nativeAppearance = c.appearance !== "none" ? ["native-appearance-unverified"] : [];
      const extra = [...nativeAppearance, ...(c.textShadow === "none" ? [] : ["text-shadow"])];
      if (native) excluded.push({ element: identity(e), reason: "native-painted-internals-unmeasured" });
      if (e.tagName === "BUTTON") {
        for (const textElement of [e, ...e.querySelectorAll("*")]) {
          if (!shown(textElement) || ![...textElement.childNodes].some(n => n.nodeType === Node.TEXT_NODE && n.textContent.trim())) continue;
          const t = getComputedStyle(textElement), largeText = parseFloat(t.fontSize) >= 24 || parseFloat(t.fontSize) >= 18.6667 && parseInt(t.fontWeight) >= 700;
          measure(textElement, "button-text", t.color, background(textElement), largeText ? 3 : 4.5, [...nativeAppearance, ...(t.textShadow === "none" ? [] : ["text-shadow"])]);
        }
      } else if (!native && e.value.length > 0) {
        measure(e, e.type === "password" ? "masking-glyph" : "control-value", c.color, bg, large ? 3 : 4.5, extra);
      }
      if (e.matches("input, textarea") && e.matches(":placeholder-shown") && e.getAttribute("placeholder")) {
        const p = getComputedStyle(e, "::placeholder"), parsed = rgb(p.color);
        const alpha = parsed ? `rgba(${parsed.slice(0, 3).join(",")},${parsed[3] * Number(p.opacity)})` : p.color;
        const placeholderLarge = parseFloat(p.fontSize) >= 24 || parseFloat(p.fontSize) >= 18.6667 && parseInt(p.fontWeight) >= 700;
        measure(e, "visible-placeholder", alpha, bg, placeholderLarge ? 3 : 4.5, [...nativeAppearance, ...(p.textShadow === "none" ? [] : ["text-shadow"])]);
      }
      if (!native && e.matches("input, textarea")) {
        const outside = background(e.parentElement);
        for (const side of ["Top", "Right", "Bottom", "Left"]) {
          if (parseFloat(c[`border${side}Width`]) <= 0 || c[`border${side}Style`] === "none") continue;
          const border = rgb(c[`border${side}Color`]);
          const additional = [...bg.uncertainty, ...nativeAppearance, ...(c.boxShadow === "none" ? [] : ["box-shadow"]), ...(!border || border[3] !== 1 || c.backgroundClip !== "border-box" ? ["border-underlay-unverified"] : [])];
          measure(e, `field-boundary-${side}-outside`, c[`border${side}Color`], outside, 3, additional);
          measure(e, `field-boundary-${side}-inside`, c[`border${side}Color`], bg, 3, additional);
        }
      }
    }
    const active = document.activeElement;
    let focus = null;
    if (active instanceof HTMLElement && active !== document.body && shown(active)) {
      const c = getComputedStyle(active), r = active.getBoundingClientRect(), padding = Math.max(0, parseFloat(c.outlineOffset)) + parseFloat(c.outlineWidth);
      const clippedBy = [];
      for (let n = active.parentElement; n; n = n.parentElement) {
        const s = getComputedStyle(n), rect = n.getBoundingClientRect();
        const axes = [];
        if (/(hidden|clip|auto|scroll)/.test(s.overflowX) && (r.left - padding < rect.left || r.right + padding > rect.right)) axes.push("x");
        if (/(hidden|clip|auto|scroll)/.test(s.overflowY) && (r.top - padding < rect.top || r.bottom + padding > rect.bottom)) axes.push("y");
        if (axes.length) clippedBy.push({ ...identity(n), axes });
      }
      focus = { element: identity(active), documentFocused: document.hasFocus(), focusVisible: active.matches(":focus-visible"), outline: c.outline, outlineOffset: c.outlineOffset, boxShadow: c.boxShadow, rect: { left: r.left, top: r.top, right: r.right, bottom: r.bottom }, viewport: { width: innerWidth, height: innerHeight }, clippedBy, geometryOnly: true, outsideViewport: r.left - padding < 0 || r.right + padding > innerWidth || r.top - padding < 0 || r.bottom + padding > innerHeight };
      if (focus.focusVisible && c.outlineStyle === "solid" && parseFloat(c.outlineWidth) > 0) {
        measure(active, "focus-outline-outside", c.outlineColor, background(parseFloat(c.outlineOffset) < 0 ? active : active.parentElement), 3, [...background(active).uncertainty, ...(c.appearance !== "none" ? ["native-appearance-unverified"] : []), ...(c.boxShadow === "none" ? [] : ["box-shadow"])]);
      } else focus.requiresVisualReview = true;
    }
    return { results, excluded, focus, forcedColors: matchMedia("(forced-colors: active)").matches, classification: "candidate-measurements-not-compliance" };
  });
}
