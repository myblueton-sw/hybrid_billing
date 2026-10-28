from pathlib import Path
import shutil

SOURCE = Path(__file__).resolve().parent
ROOT = SOURCE.parents[3]

for locale in ['en', 'ko']:
    suffix = '-ko' if locale == 'ko' else ''
    out = ROOT / ('output/designs/HB-35-v5' + suffix)
    out.mkdir(parents=True, exist_ok=True)
    for name in ['app.js', 'data.js', 'views.js', 'erp.js', 'style.css']:
        shutil.copyfile(SOURCE / name, out / name)
    html = (SOURCE / 'index.html').read_text().replace('lang="en"', f'lang="{locale}"')
    (out / 'index.html').write_text(html)
    previous = ROOT / ('output/designs/HB-35-v4' + suffix)
    shutil.copytree(previous, out / 'previous', dirs_exist_ok=True, ignore=shutil.ignore_patterns('*.png', '*.csv', '*check.json'))
    shutil.copyfile(previous / 'sample-source.png', out / 'previous/sample-source.png')
    for f in (out / 'previous').rglob('*.html'):
        if 'customer' in f.name:
            continue
        content = f.read_text()
        relative = '../' * len(f.relative_to(out).parts[:-1]) + 'index.html'
        label = '고객·계약 및 월별 발행 이력' if locale == 'ko' else 'Customers, contracts and monthly issuance'
        link = f'<a href="{relative}#register" style="display:block;padding:10px;font-size:12px">{label} ↗</a>'
        content = content.replace('</nav>', link + '</nav>')
        for name, company in [('Northstar Labs', 'northstar'), ('Juniper Retail', 'juniper'), ('Aster Health', 'aster')]:
            content = content.replace('>' + name + '<', f'><a href="{relative}#customer/{company}">{name}</a><')
        f.write_text(content)
    print(out)
