import argparse
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from modules.focus_local.database import environment
from modules.focus_import.overview_http import OverviewServer


def main():
    p = argparse.ArgumentParser(description='Read-only authorized collected FOCUS overview')
    p.add_argument('--local-operator', action='store_true', required=True)
    p.add_argument('--actor', required=True)
    p.add_argument('--port', type=int, default=8806)
    p.add_argument('--ui-port', type=int, default=8787)
    p.add_argument('--pg-port', type=int)
    p.add_argument('--ch-port', type=int)
    p.add_argument('--s3-port', type=int)
    args = p.parse_args()
    try:
        for kind in ['PG', 'CH', 'S3']:
            value = getattr(args, kind.lower() + '_port')
            if value is not None:
                os.environ[f'HB_LOCAL_{kind}_PORT'] = str(value)
        environment()
        server = OverviewServer(args.port, args.actor, args.ui_port)
        print(f'Collected FOCUS local reader: http://127.0.0.1:{server.server_port}', flush=True)
        server.serve_forever()
    except KeyboardInterrupt:
        return 0
    except Exception:
        print('Collected overview unavailable: verify trusted local configuration.', file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
