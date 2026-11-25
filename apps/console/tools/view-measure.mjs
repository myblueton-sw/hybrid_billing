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
