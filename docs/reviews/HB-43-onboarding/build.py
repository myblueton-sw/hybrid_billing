from pathlib import Path
from html import escape
from urllib.parse import urlparse
import hashlib
import json
import shutil

from bs4 import BeautifulSoup
from plan import CHAPTERS, PAGES
from screens import additions
from customer_profile import extend_profile
from pricing_search import extend_pricing
from ai_pricing import additions as ai_additions
from cost_explorer import additions as explorer_additions

SOURCE = Path(__file__).resolve().parent
ROOT = SOURCE.parents[2]
OUT = ROOT / 'output/designs/HB-43'
CAPTURES = json.loads((ROOT / 'output/designs/HB-40/captures.json').read_text())
CURRENT = ROOT / 'docs/reviews/HB-38-billing-operations/prototype'
V5 = ROOT / 'docs/reviews/HB-35-screen-planning/prototype-v5'
IDS = {row[0] for row in PAGES}
ROUTES = {row[3]: row[0] for row in PAGES if row[2] in ('v5', 'current')}
ROUTES.update({'register': 'operations', 'contracts': 'supplier-register', 'policies': 'sales-policy', 'preview': 'sales-impact', 'oci': 'supplier-oci', 'history': 'v4-monthly'})
assert len(IDS) == len(PAGES) == 89


def context(page_id, locale):
    ko = locale == 'ko'
    if page_id in ('cost-pivot', 'raw-explorer'):
        return 'XPL-DEMO · 2026-10 · USD · ' + ('독립 비용 탐색 예시' if ko else 'Separate cost exploration example')
    if page_id.startswith('marketplace-'):
        return 'MP-DEMO · ' + ('독립 ISV 비용·계획 예시' if ko else 'Separate ISV cost/planning example')
    if page_id == 'cost-differences':
        return ('서로 독립된 대사 예시 · 통화 합산 금지' if ko else 'Independent reconciliation examples; no mixed-currency total')
    if page_id.startswith('ai-'):
        return 'AI-DEMO-01 · 2026-10 · USD · ' + ('독립 AI 요율 합성 예시' if ko else 'Separate synthetic AI pricing case')
    if page_id.startswith('entry-'):
        return ('가입·접근 설계 예시 · 실제 인증 미연결' if ko else 'Access design example; no live authentication')
    if page_id.startswith('setup-'):
        return ('ONB-DEMO-01 · 신규 설치 · 합성 온보딩 예시' if ko else 'ONB-DEMO-01 · new installation · synthetic onboarding')
    if page_id in ('auto-portfolio', 'auto-bulk', 'auto-exceptions'):
        return 'PORT-OCT-S1 · Meridian MSP · 2026-10 · ' + ('합성 포트폴리오 · 선택 고객 Northstar' if ko else 'Synthetic portfolio; selected customer Northstar')
    if page_id.startswith('auto-'):
        return 'SIM-NS-OCT · Northstar · 2026-10 · USD · ' + ('독립 요율 시뮬레이션' if ko else 'Isolated pricing simulation')
    if page_id.startswith('v4-') and page_id in ('v4-inbox', 'v4-extract', 'v4-differences', 'v4-audit'):
        return 'Cedar · SAMPLE-AWS-CEDAR-09 · 2026-09 · USD'
    if page_id in ('sales-policy', 'sales-impact'):
        return 'Birch 0226 · C00226 · 2026-09 · USD'
    if page_id.startswith('supplier-'):
        return 'Meridian MSP · ' + ('공급자 매입 계약 합성 예시' if ko else 'Synthetic supplier purchase case')
    if page_id == 'operations':
        return 'Meridian MSP · 2026-09 · ' + ('5,000개 회사 운영 예시' if ko else '5,000-company portfolio example')
    if page_id in ('review-issue', 'index-seats', 'index-pricing', 'index-fx'):
        return 'Juniper · ' + ('독립 처리 예시 · 본문 기간 기준' if ko else 'Separate scenario; period shown in content')
    if page_id in ('index-runs', 'index-approval', 'index-intake', 'index-mapping'):
        return 'Aster · ' + ('독립 처리 예시 · 본문 버전 기준' if ko else 'Separate scenario; revision shown in content')
    if page_id in ('overview', 'contents', 'assignment-policy', 'erp', 'license-modes', 'license-status', 'core-reconciliation-setup'):
        return ('전체 검토 / 설정 예시' if ko else 'Whole review / configuration example')
    return ('본문에 표시된 독립 합성 사례 · 대상·기간·버전 확인' if ko else 'Independent synthetic case; subject, period and revision shown in content')


def resolve_link(href, current):
    parsed = urlparse(href)
    if href == '../customer/index.html':
        return href
    if parsed.fragment == 'register/northstar':
        return 'v4-monthly.html'
    if parsed.fragment == 'core-customer-usage-question':
        return 'customer-inquiry.html'
    if parsed.scheme in ('mailto', 'http', 'https'):
        return href
    fragment = parsed.fragment
    if fragment in IDS:
        return fragment + '.html'
    if fragment in ROUTES:
        return ROUTES[fragment] + '.html'
    candidates = []
    if parsed.path:
        stem = Path(parsed.path).stem
        candidates += [stem + '-' + fragment]
    candidates += [current.rsplit('-', 1)[0] + '-' + fragment, 'v4-' + fragment, 'index-' + fragment, 'core-index-' + fragment, 'core-reconciliation-' + fragment, 'core-customer-' + fragment, 'customer-' + fragment]
    for candidate in candidates:
        if fragment and candidate in IDS:
            return candidate + '.html'
    if 'sample-cedar-aws-09.pdf' in href:
        return '../assets/sample-cedar-aws-09.pdf'
    return None


def clean_main(html, page_id, locale, unresolved):
    soup = BeautifulSoup(html, 'html.parser')
    main = soup.find('main')
    if main is None:
        main = soup.new_tag('main')
        for child in list(soup.contents):
            main.append(child.extract())
        soup.append(main)
    for node in main.select('header, footer, script, .eyebrow, .eyebrow4, .back, .contract-tabs, .pager'):
        node.decompose()
    for node in main.select('[id]'):
        if node.get('id') in ('content', 'workspace'):
            node['data-original-id'] = node['id']
            del node['id']
    for tab in main.find_all('table'):
        if not any(set(a.get('class', [])) & {'table-scroll', 'table-wrap'} for a in tab.parents if hasattr(a, 'get')):
            wrapper = soup.new_tag('div', attrs={'class': 'table-scroll', 'tabindex': '0'})
            tab.wrap(wrapper)
    for node in main.select('button, input, select, textarea'):
        node['disabled'] = ''
        node['title'] = '검토용 화면 상태' if locale == 'ko' else 'Read-only review state'
    for node in main.find_all(True):
        for attr in list(node.attrs):
            if attr.startswith('on'):
                del node[attr]
    for a in list(main.select('a[href]')):
        href = a['href']
        resolved = resolve_link(href, page_id)
        if resolved:
            a['href'] = resolved
            a.attrs.pop('download', None)
        else:
            unresolved.append({'page': page_id, 'href': href, 'label': a.get_text(' ', strip=True)})
            a.name = 'span'
            a.attrs = {'class': 'static-reference'}
    for img in main.select('img'):
        if img.get('src') == 'sample-source.png':
            img['src'] = '../assets/sample-source.png'
    if page_id == 'operations':
        for row in main.select('.operation-row')[10:]:
            row.decompose()
        for note in main.select('.print-only'):
            note.string = '조회 결과 첫 10행 발췌 · 원 시연은 25/50/100행 지원' if locale == 'ko' else 'First 10 results shown; the original demonstration supports 25/50/100 rows.'
    if page_id == 'core-index-allocation':
        extra = '<h2>' + ('그룹사 비용 배분 검증' if locale == 'ko' else 'Group-company allocation controls') + '</h2><p>' + ('규칙 버전·적용 기간·배부 기준·가중치·잔여액 배정을 고정합니다. 통화별 배부 합계 + 미배부 = 대상 원가를 확인하고 미귀속·폐쇄 조직 비용을 임의 재배분하지 않습니다. 다른 법인의 그룹 합산 조회는 법적 통합 인보이스가 아닙니다. 발행 당시 배부는 고정 보존합니다.' if locale == 'ko' else 'Pin rule version, effective interval, allocation basis, weights and residual assignment. Allocated plus unallocated equals the target per currency; never silently move unmapped or closed-entity costs. A group report is not a consolidated legal invoice. Preserve issued allocation snapshots.') + '</p>'
        main.append(BeautifulSoup(extra, 'html.parser'))
    if page_id == 'core-index-close':
        main.find('h1').string = '선택 포트폴리오의 월 마감 작업' if locale == 'ko' else 'Selected portfolio close workbench'
    main['class'] = list(main.get('class', [])) + ['book-main']
    return main


def guide(locale, chapter_starts):
    ko = locale == 'ko'
    t = lambda en, kr: kr if ko else en
    rows = ''.join(f'<tr><td>{i + 1:02}</td><td><a href="#{chapter_starts[c]}">{kr if ko else en}</a></td><td>{next(n + 1 for n, p in enumerate(PAGES) if p[0] == chapter_starts[c])}</td></tr>' for i, (c, en, kr) in enumerate(CHAPTERS[1:]))
    intro = f'<h1>{t("Hybrid Billing — complete review", "Hybrid Billing 전체 통합 시안")}</h1><p class="lead">{t("First design sign-off · v10 · 89 pages / 87 screen states", "1차 시안 확정 검토용 · v10 · 89페이지 / 87개 화면 상태")}</p>'
    intro += '<div class="flow"><div>01<br>' + t('Collect and verify', '원천 수집·검증') + '</div><div>02<br>' + t('Contract and calculate', '계약·요율 계산') + '</div><div>03<br>' + t('Correct and review', '교정·검수') + '</div><div>04<br>' + t('Issue and explain', '발행·고객 설명') + '</div></div>'
    intro += '<h2>' + t('Review the entire workflow in order', '전체 업무를 순서대로 검토하세요') + '</h2><table><thead><tr><th>#</th><th>' + t('Chapter', '검토 구간') + '</th><th>' + t('Page', '시작 페이지') + '</th></tr></thead><tbody>' + rows + '</tbody></table>'
    intro += '<h2>' + t('What is integrated', '이번 통합본에 반영한 내용') + '</h2><p>' + t('All 66 v9 pages are retained. Five cost exploration, discrepancy and Marketplace states are added. Two AI pricing/invoice states show stacked discounts and approved cost disclosure. Sixteen new states connect initial registration, login, scoped roles, onboarding, data connection, collection and billing readiness. The large-MSP worklist replaces the overloaded global register. Common contract management, five provider details, recurring and one-off rules, price impact and license states are integrated at their workflow positions.', 'v9의 전체 66페이지를 유지하고 비용 피벗·원천 조회·차이 대사·Marketplace 화면 5개를 추가했습니다. AI 단계별 할인·원가 공개·인보이스 상세 2개 화면과 최초 등록·로그인·역할별 권한·온보딩·데이터 연결·수집·정산 준비 화면 16개를 추가했습니다. 복잡했던 전체 발행 목록은 대규모 MSP 업무 목록으로 교체했고, 공통 계약 관리·5개 공급자 상세·상시 및 일회성 정책·금액 영향·라이선스 상태를 업무 흐름 안에 통합했습니다.') + '</p>'
    intro += '<p class="note">' + t('This is a complete design review book, not a connected production application. Controls are read-only snapshots; page/chapter links navigate the book. Different synthetic cases are labelled at the top and are not one shared invoice. The separate customer entry includes only customer pages.', '실제 운영 앱이 아닌 전체 설계 검토본입니다. 작업 컨트롤은 읽기 전용 화면 상태이며, 페이지·목차 링크로 이동합니다. 상단에 표시한 서로 다른 합성 사례는 하나의 인보이스가 아닙니다. 별도 고객 진입점에는 고객 화면만 포함합니다.') + '</p>'
    intro += '<p><a href="../customer/index.html">' + t('Open separate customer review →', '분리된 고객 화면 검토 열기 →') + '</a></p>'
    return intro


for locale in ('en', 'ko'):
    ko = locale == 'ko'
    t = lambda en, kr: kr if ko else en
    folder = OUT / locale
    for name in ('pages', 'assets', 'customer'):
        (folder / name).mkdir(parents=True, exist_ok=True)
    baseline = ROOT / ('output/designs/HB-35-v5' + ('-ko' if ko else ''))
    old_path = baseline / 'previous/print.html'
    old = BeautifulSoup(old_path.read_text(), 'html.parser')
    legacy = {s['id']: str(s.find('main')) for s in old.select('section.screen')}
    legacy_classes = {s['id']: ' '.join(c for c in s.get('class', []) if c != 'screen') for s in old.select('section.screen')}
    assert len(legacy) == 35
    planned_legacy = {p[3] for p in PAGES if p[2] == 'legacy'}
    assert planned_legacy == set(legacy)
    captured = {(x['family'], x['route']): x['html'] for x in CAPTURES[locale]}
    css = {'legacy': old.find('style').string, 'v5': (V5 / 'style.css').read_text(), 'current': (CURRENT / 'style.css').read_text(), 'new': (CURRENT / 'style.css').read_text()}
    fresh = extend_pricing(additions(locale), locale)
    fresh.update(ai_additions(locale))
    fresh.update(explorer_additions(locale))
    starts = {c: next(p[0] for p in PAGES if p[1] == c) for c, _, _ in CHAPTERS}
    fresh['overview'] = guide(locale, starts)
    unresolved = []
    docs = {}
    manifest = []
    for page_id, chapter, family, source in PAGES:
        raw = legacy[source] if family == 'legacy' else captured[(family, source)] if family in ('v5', 'current') else fresh.get(page_id, '<h1>' + t('Complete screen index', '전체 화면 목차') + '</h1>')
        if page_id == 'customer-detail':
            raw = extend_profile(raw, locale)
        main = clean_main(raw, page_id, locale, unresolved)
        title = main.find('h1').get_text(' ', strip=True)
        docs[page_id] = main
        manifest.append({'id': page_id, 'page': len(manifest) + 1, 'chapter': chapter, 'family': family, 'source': source, 'title': title, 'context': context(page_id, locale)})
    contents = docs['contents']
    for c, en, kr in CHAPTERS[1:]:
        block = BeautifulSoup('<section class="index-group"><h2>' + (kr if ko else en) + '</h2>' + ''.join(f'<a href="{x["id"]}.html"><span>{x["page"]:02}</span>{escape(x["title"])}</a>' for x in manifest if x['chapter'] == c) + '</section>', 'html.parser')
        contents.append(block)
    contents['class'].append('contents-grid')
    for asset in ('sample-source.png', 'sample-cedar-aws-09.pdf'):
        shutil.copyfile(baseline / 'previous' / asset, folder / 'assets' / asset)
    for i, item in enumerate(manifest):
        page_id, chapter, family = item['id'], item['chapter'], item['family']
        main = docs[page_id]
        header = BeautifulSoup('<div class="book-top"><span>' + t('DESIGN REVIEW / ', '전체 시안 검토 / ') + escape(item['context']) + '</span><span>' + t('Read-only screen state', '읽기 전용 화면 상태') + '</span></div>', 'html.parser')
        main.insert(0, header)
        rail = '<aside class="book-rail"><b>HYBRID BILLING</b><small>' + t('Full design review · v10', '전체 통합 검토안 · v10') + '</small><nav>' + ''.join(f'<a href="{starts[c]}.html" '+('aria-current="page"' if c == chapter else '')+f'>{kr if ko else en}</a>' for c, en, kr in CHAPTERS) + '</nav><a class="book-index" href="contents.html">' + t('All 87 screen states', '전체 87개 화면 목차') + '</a><p>' + t('Synthetic cases · no real billing action', '합성 사례 · 실제 청구 작업 없음') + '</p></aside>'
        previous = manifest[max(i - 1, 0)]['id']
        following = manifest[min(i + 1, len(manifest) - 1)]['id']
        footer = f'<div class="book-footer"><a href="{previous}.html">← {t("Previous", "이전")}</a><span>Hybrid Billing · v10 · {i+1:02} / {len(manifest)} · {t("Synthetic review", "합성 시안")}</span><a href="{following}.html">{t("Next", "다음")} →</a></div>'
        body = rail + str(main) + footer
        html = f'<!doctype html><html lang="{locale}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{escape(item["title"])} · Hybrid Billing v10</title><style>{css[family]}</style><link rel="stylesheet" href="../book.css"></head><body class="book family-{family} {legacy_classes.get(page_id, "")}">{body}</body></html>'
        (folder / 'pages' / (page_id + '.html')).write_text(html)
    shutil.copyfile(SOURCE / 'book.css', folder / 'book.css')
    (folder / 'index.html').write_text(f'<!doctype html><html lang="{locale}"><meta charset="utf-8"><meta http-equiv="refresh" content="0;url=pages/overview.html"><title>Hybrid Billing v10</title><a href="pages/overview.html">{t("Open full design review", "전체 시안 열기")}</a></html>')
    customer_items = [x for x in manifest if x['chapter'] == 'customer']
    customer_ids = {x['id'] for x in customer_items}
    for i, item in enumerate(customer_items):
        doc = BeautifulSoup((folder / 'pages' / (item['id'] + '.html')).read_text(), 'html.parser')
        doc.select_one('.book-rail').clear()
        doc.select_one('.book-rail').append(BeautifulSoup('<b>HYBRID BILLING</b><small>' + t('Customer review', '고객 화면 검토') + '</small><nav>' + ''.join(f'<a href="{x["id"]}.html">{escape(x["title"])}</a>' for x in customer_items) + '</nav>', 'html.parser'))
        doc.select_one('.book-top').string = t('CUSTOMER VIEW · Northstar · synthetic read-only design', '고객 화면 · Northstar · 합성 읽기 전용 시안')
        doc.select_one('.book-footer').string = f'{t("Customer review", "고객 화면 검토")} · {i+1} / {len(customer_items)}'
        for a in list(doc.select('a[href]')):
            if a['href'].endswith('.html') and Path(a['href']).stem not in customer_ids:
                a.name = 'span'
                a.attrs = {}
        (folder / 'customer' / (item['id'] + '.html')).write_text(str(doc))
    first = customer_items[0]['id'] + '.html'
    (folder / 'customer/index.html').write_text(f'<html><meta http-equiv="refresh" content="0;url={first}"><a href="{first}">Customer</a></html>')
    provenance = {'v5_print_sha256': hashlib.sha256(old_path.read_bytes()).hexdigest(), 'coverage': {'v9_pages_retained': 66, 'new_entry_states': 16, 'new_ai_states': 2, 'new_cost_visibility_states': 5, 'v8_pages_retained': 54, 'new_pricing_states': 12, 'v5_total': 42, 'legacy_retained': 35, 'v5_routes_retained': 6, 'v5_register_replaced_by': 'operations'}, 'pages': manifest, 'disabled_unmapped_references': unresolved}
    (folder / 'manifest.json').write_text(json.dumps(provenance, ensure_ascii=False, indent=2))
    print(json.dumps({'locale': locale, 'pages': len(manifest), 'unmapped_static_references': len(unresolved)}))
