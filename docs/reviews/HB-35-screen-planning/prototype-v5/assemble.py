from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[4]
for suffix in ['', '-ko']:
    directory = ROOT / ('output/designs/HB-35-v5' + suffix)
    files = [directory / f'page-{i}.pdf' for i in range(1, 8)] + [directory / 'previous-flow.pdf']
    for file, expected in zip(files, [1] * 7 + [35]):
        info = subprocess.check_output(['pdfinfo', str(file)], text=True)
        count = int(next(line.split(':')[1] for line in info.splitlines() if line.startswith('Pages:')))
        if count != expected:
            raise ValueError(f'{file.name}: expected {expected} pages, got {count}')
    target = ROOT / ('output/pdf/hybrid-billing-msp-design-v5' + suffix + '.pdf')
    subprocess.run(['pdfunite', *map(str, files), str(target)], check=True)
    print(target)
