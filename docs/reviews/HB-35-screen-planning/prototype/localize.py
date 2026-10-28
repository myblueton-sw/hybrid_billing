from pathlib import Path
import html
import json
import re
import shutil

source = Path(__file__).resolve().parent
root = source.parents[3]
target = root / 'output/designs/HB-35-v2-ko'
target.mkdir(parents=True, exist_ok=True)
translations = json.loads((source / 'ko.json').read_text())
missing = set()

def translate(value):
    decoded = html.unescape(value)
    key = decoded.strip()
    if not key:
        return value
    if key in translations:
        return decoded.replace(key, html.escape(translations[key], quote=False))
    if re.search('[A-Za-z]', key):
        missing.add(key)
    return value

for name in ['index.html', 'reconciliation.html', 'customer.html']:
    text = (source / name).read_text()
    text = re.sub(r'(?<=>)[^<>]+(?=<)', lambda m: translate(m[0]), text)
    text = re.sub(r'(data-title|placeholder|aria-label)="([^"]*)"', lambda m: m[1] + '="' + html.escape(translations.get(html.unescape(m[2]), html.unescape(m[2])), quote=True) + '"', text)
    text = text.replace('<html lang="en">', '<html lang="ko">')
    (target / name).write_text(text)
for name in ['style.css', 'navigation.js']:
    shutil.copyfile(source / name, target / name)
if missing:
    raise ValueError('Unmapped UI text: ' + repr(sorted(missing)))
print('Korean HTML generated: all English text nodes mapped or preserved identifiers.')
