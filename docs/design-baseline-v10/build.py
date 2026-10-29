from pathlib import Path
import hashlib
import html
import json
import subprocess
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parents[2]
SOURCE = Path(__file__).resolve().parent
OUT = ROOT / 'output/designs/v10-modules'
registry = json.loads((SOURCE / 'registry.json').read_text())
OUT.mkdir(parents=True, exist_ok=True)
results = []
for locale in ('en', 'ko'):
    baseline = ROOT / registry['artifacts'][locale]['path']
    assert hashlib.sha256(baseline.read_bytes()).hexdigest() == registry['artifacts'][locale]['sha256']
    folder = OUT / locale
    folder.mkdir(exist_ok=True)
    links = []
    covered = []
    with tempfile.TemporaryDirectory() as temp:
        subprocess.run(['pdfseparate', str(baseline), temp + '/%d.pdf'], check=True)
        for module in registry['modules']:
            if not module['pages']:
                continue
            pages = [p['baseline_page'] for p in module['pages']]
            covered.extend(pages)
            target = folder / (module['id'] + '-r1.pdf')
            subprocess.run(['pdfunite', *[temp + '/' + str(n) + '.pdf' for n in pages], str(target)], check=True)
            actual = subprocess.check_output(['pdftotext', '-layout', str(target), '-'], text=True)
            expected = ''.join(subprocess.check_output(['pdftotext', '-layout', temp + '/' + str(n) + '.pdf', '-'], text=True) for n in pages)
            assert actual == expected, module['id']
            for path, prefix in [(Path(temp) / (str(pages[0]) + '.pdf'), Path(temp) / 'before'), (target, Path(temp) / 'after')]:
                subprocess.run(['pdftoppm', '-f', '1', '-singlefile', '-scale-to', '600', '-png', str(path), str(prefix)], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            assert (Path(temp) / 'before.png').read_bytes() == (Path(temp) / 'after.png').read_bytes()
            (folder / (module['id'] + '.png')).write_bytes((Path(temp) / 'after.png').read_bytes())
            links.append('<li><a href="' + target.name + '">' + html.escape(module['title'][locale]) + '</a> · v10 r1 · ' + str(pages[0]) + '–' + str(pages[-1]) + '</li>')
            results.append({'locale': locale, 'module': module['id'], 'pages': pages, 'sha256': hashlib.sha256(target.read_bytes()).hexdigest(), 'text_equal': True, 'first_page_pixels_equal': True})
    assert sorted(covered) == list(range(1, 90)) and len(set(covered)) == 89
    title = 'v10 기준 시안 · 기능별 관리' if locale == 'ko' else 'v10 baseline · modular design'
    note = '원본 페이지 번호를 유지한 분할본입니다. PDF 내부 링크는 기존 전체본 기준이며, 이 목차에서 모듈을 여세요. VMware·온프레미스 상세 모듈은 보완 예정입니다.' if locale == 'ko' else 'Split excerpts retain baseline page numbers. Embedded PDF links retain their original targets; use this index to select modules. VMware/on-prem detail is planned, not delivered.'
    (folder / 'index.html').write_text('<!doctype html><html lang="' + locale + '"><meta charset="utf-8"><title>' + title + '</title><style>body{font:17px system-ui;max-width:900px;margin:40px auto;padding:20px;color:#222}li{padding:12px;border-bottom:1px solid #ddd}a{color:#234}p{line-height:1.7}</style><h1>' + title + '</h1><p>' + note + '</p><ol>' + ''.join(links) + '</ol></html>')
    assert hashlib.sha256(baseline.read_bytes()).hexdigest() == registry['artifacts'][locale]['sha256']
(OUT / 'index.html').write_text('<!doctype html><meta charset="utf-8"><title>Hybrid Billing v10 modules</title><h1>Hybrid Billing v10</h1><p><a href="ko/index.html">한국어 기능별 시안</a></p><p><a href="en/index.html">English modules</a></p>')
(OUT / 'verification.json').write_text(json.dumps(results, indent=2) + '\n')
archive = ROOT / 'output/designs/hybrid-billing-v10-modules.zip'
with zipfile.ZipFile(archive, 'w', zipfile.ZIP_DEFLATED) as z:
    for path in sorted(OUT.rglob('*')):
        if path.is_file() and path.suffix in ('.html', '.pdf', '.json'):
            z.write(path, path.relative_to(OUT))
print(json.dumps({'modules': 11, 'pdfs': len(results), 'pages_per_locale': 89, 'baseline_hashes_unchanged': True, 'archive': str(archive.relative_to(ROOT))}))
