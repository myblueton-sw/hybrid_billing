import argparse
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from modules.focus_local.http_service import LocalServer


def main():
    parser = argparse.ArgumentParser(description='Loopback local-operator console and actual FOCUS ingestion service')
    parser.add_argument('--local-operator', action='store_true', required=True)
    parser.add_argument('--actor', choices=['loader-a', 'loader-b'], default='loader-a')
    parser.add_argument('--port', type=int, default=8793)
    parser.add_argument('--build', type=Path, default=Path(__file__).resolve().parents[1] / 'apps/console/dist')
    args = parser.parse_args()
    with LocalServer(args.port, args.actor, args.build) as server:
        print(f'{server.origin}/#/imports?lang=ko&service=local', flush=True)
        try:
            server.serve_forever()
        except KeyboardInterrupt:
            pass


if __name__ == '__main__':
    main()
