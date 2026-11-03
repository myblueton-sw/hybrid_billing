import os
import re

from .contract import Rejected


DEFAULTS = {'PG': 15432, 'CH': 18123, 'S3': 18333}


def port(env, service):
    if service not in DEFAULTS:
        raise Rejected('local_port')
    value = os.environ.get(f'HB_LOCAL_{service}_PORT', env.get(f'{service}_PORT', DEFAULTS[service]))
    if type(value) is int:
        number = value
    elif isinstance(value, str) and re.fullmatch('[0-9]{1,5}', value):
        number = int(value)
    else:
        raise Rejected('local_port')
    if not 1 <= number <= 65535:
        raise Rejected('local_port')
    return number
