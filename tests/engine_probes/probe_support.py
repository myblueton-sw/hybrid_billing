import re


def first_line(exc):
    return str(exc).splitlines()[0][:240]


def ch_code(exc):
    match = re.search(r"Code: (\d+)", str(exc))
    return int(match.group(1)) if match else None
