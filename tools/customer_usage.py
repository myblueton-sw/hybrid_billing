import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from modules.focus_import.customer_usage import main


if __name__ == '__main__':
    raise SystemExit(main())
