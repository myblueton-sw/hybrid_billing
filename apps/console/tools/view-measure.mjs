export function inspectView(page) {
  return page.evaluate(() => {
  const vw = document.documentElement.clientWidth;
  const rgb = (v) => { const p = v.match(/[\d.]+/g)?.map(Number); return /^rgba?\(/.test(v) && p?.length >= 3 ? [p[0], p[1], p[2], p[3] ?? 1] : null; };
  const over = (a, b) => [0, 1, 2].map((i) => a[i] * a[3] + b[i] * (1 - a[3]));
  const lum = (c) => c.map((v) => { v /= 255; return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; }).reduce((s, v, i) => s + v * [0.2126, 0.7152, 0.0722][i], 0);
  const ratio = (a, b) => (Math.max(lum(a), lum(b)) + 0.05) / (Math.min(lum(a), lum(b)) + 0.05);
  const name = (e) => `${e.tagName.toLowerCase()}${e.id ? "#" + e.id : ""}${typeof e.className === "string" && e.className ? "." + e.className.trim().split(/\s+/).slice(0, 2).join(".") : ""}`;
  const hidden = (e) => !!e.closest(".visually-hidden, [hidden], [inert], [aria-hidden=true]");
  const scrolls = (e) => { for (let n = e.parentElement; n && n !== document.body; n = n.parentElement) { const c = getComputedStyle(n); if (/(auto|scroll|hidden|clip)/.test(c.overflowX)) return true; } return false; };
  const cut = [], seenCut = new Set(); let measured = 0, unsureCount = 0;
  const clipped = [], low = [], seenClip = new Set(), seenLow = new Set();
  for (const e of document.querySelectorAll("main *")) {
    if (hidden(e) || !e.getClientRects().length) continue;
    const r = e.getBoundingClientRect(), c = getComputedStyle(e);
    if (r.width < 2 || r.height < 2 || c.visibility !== "visible") continue;
    if ((r.right > vw + 1 || r.left < -1) && !scrolls(e)) { const k = name(e); if (!seenClip.has(k)) { seenClip.add(k); clipped.push({ el: k, left: Math.round(r.left), right: Math.round(r.right), vw }); } }
    if (/(hidden|clip)/.test(c.overflowX) && e.scrollWidth > e.clientWidth + 1 && c.textOverflow !== "ellipsis") { const k = name(e); if (!seenCut.has(k)) { seenCut.add(k); cut.push({ el: k, content: e.scrollWidth, box: e.clientWidth }); } }
    const text = [...e.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim().length > 0);
    if (!text || e.matches(":disabled, [aria-disabled=true]") || e.closest(":disabled")) continue;
    const fg = rgb(c.color); if (!fg) continue;
    let bg = [255, 255, 255], unsure = false; const chain = [];
    for (let n = e; n; n = n.parentElement) chain.unshift(n);
    for (const n of chain) { const s = getComputedStyle(n), p = rgb(s.backgroundColor); if (!p) unsure = true; else bg = over(p, bg); if (s.backgroundImage !== "none" || Number(s.opacity) !== 1) unsure = true; }
    if (unsure) { unsureCount++; continue; }
    measured++;
    const size = parseFloat(c.fontSize), bold = Number(c.fontWeight) >= 700, large = size >= 24 || (bold && size >= 18.66);
    const got = ratio(over(fg, bg), bg), need = large ? 3 : 4.5;
    if (got < need) { const k = name(e) + c.color; if (!seenLow.has(k)) { seenLow.add(k); low.push({ el: name(e), color: c.color, on: bg.map(Math.round).join(","), ratio: Math.round(got * 100) / 100, need, sample: e.textContent.trim().slice(0, 30) }); } }
  }
  return { overflow: document.documentElement.scrollWidth - vw, clipped: clipped.slice(0, 12), cut: cut.slice(0, 12), low: low.slice(0, 12), measured, unsure: unsureCount };
});
}

export function inspectColour(page) {
  return page.evaluate(() => {
    const rgb = (v) => { const p = v.match(/[\d.]+/g)?.map(Number); return /^rgba?\(/.test(v) && p?.length >= 3 ? [p[0], p[1], p[2], p[3] ?? 1] : null; };
    const chroma = (c) => { if (!c || c[3] < 0.2) return 0; const max = Math.max(c[0], c[1], c[2]), min = Math.min(c[0], c[1], c[2]); return max === 0 ? 0 : (max - min) / max; };
    const name = (e) => `${e.tagName.toLowerCase()}${e.id ? "#" + e.id : ""}${typeof e.className === "string" && e.className ? "." + e.className.trim().split(/\s+/).slice(0, 2).join(".") : ""}`;
    const named = (e) => !!(e.getAttribute("aria-label") || e.getAttribute("title") || e.getAttribute("aria-labelledby"));
    const marks = [], strips = [], seenMark = new Set(), seenStrip = new Set(), colouredKinds = {}, tinted = {}; let boxes = 0, coloured = 0;
    for (const e of document.querySelectorAll("main *")) {
      if (e.closest(".visually-hidden, [hidden], [inert]") || !e.getClientRects().length) continue;
      const r = e.getBoundingClientRect(), c = getComputedStyle(e);
      if (r.width < 2 || r.height < 2 || c.visibility !== "visible") continue;
      boxes++;
      const sides = ["Top", "Right", "Bottom", "Left"].map((s) => ({ side: s, width: parseFloat(c[`border${s}Width`]), style: c[`border${s}Style`], chroma: chroma(rgb(c[`border${s}Color`])) })).filter((b) => b.width > 0 && b.style !== "none");
      const vivid = sides.filter((b) => b.chroma > 0.35);
      const fill = chroma(rgb(c.backgroundColor));
      if (fill > 0.35 || vivid.length) { coloured++; colouredKinds[name(e)] = (e.textContent.trim().slice(0, 24) || "(no text)"); }
      const ownText = [...e.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim().length > 0);
      if (ownText && chroma(rgb(c.color)) > 0.35 && !e.closest("a, button")) tinted[name(e) + " " + c.color] = e.textContent.trim().slice(0, 30);
      const thick = vivid.filter((b) => b.width >= 3);
      if (thick.length && thick.length < 4 && sides.every((b) => thick.includes(b) || b.width < 3) && !e.matches(":focus-visible")) { const k = name(e); if (!seenStrip.has(k)) { seenStrip.add(k); strips.push({ el: k, sides: thick.map((b) => b.side + ":" + b.width).join(","), text: e.textContent.trim().slice(0, 30) }); } }
      const empty = e.textContent.trim() === "" && !e.querySelector("img, svg, input, select, textarea, button") && !["INPUT", "SELECT", "TEXTAREA", "IMG", "SVG", "PATH", "HR", "BR"].includes(e.tagName.toUpperCase());
      if (empty && (fill > 0.35 || vivid.length) && r.width <= 48 && r.height <= 48 && !named(e)) { const k = name(e); if (!seenMark.has(k)) { seenMark.add(k); marks.push({ el: k, size: Math.round(r.width) + "x" + Math.round(r.height), hidden: !!e.closest("[aria-hidden=true]"), beside: (e.parentElement?.textContent ?? "").trim().slice(0, 30) }); } }
    }
    return { boxes, coloured, colouredKinds, tinted, marks: marks.slice(0, 12), strips: strips.slice(0, 12) };
  });
}
