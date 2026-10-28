import json
from unittest.mock import patch

from cases import fresh_denied, unaffected
from environment import ACTORS, COMPANY, KEYS
from modules.focus_import import authority, control, lifecycle
from modules.focus_import.model import Rejected


class CheckpointFailure:
    def __init__(self, connection, observed):
        self.connection = connection
        self.observed = observed

    def __getattr__(self, name):
        return getattr(self.connection, name)

    def execute(self, query, *args, **kwargs):
        if isinstance(query, str) and query.startswith('UPDATE hb_focus.control_checkpoint SET sequence='):
            self.observed.append('checkpoint-update-after-durable-head')
            raise RuntimeError('synthetic-checkpoint-unavailable')
        return self.connection.execute(query, *args, **kwargs)


def checkpoint_state(env):
    with authority.connect(env) as connection:
        checkpoint = control.checkpoint(connection)
    head = json.loads((control.DIRECTORY / 'head.json').read_text())
    return {'sequence': checkpoint['sequence'], 'digest': checkpoint['digest'],
        'head_sequence': head['sequence'], 'head_digest': head['digest']}


def assert_normal_load_denied(env):
    with authority.connect(env) as connection:
        try:
            control.Journal(env, connection).load()
        except Rejected as error:
            assert str(error) == 'control_checkpoint'
        else:
            raise AssertionError('normal journal load accepted an unmatched checkpoint')


def inject_and_resume(env, completion):
    target = ACTORS[KEYS[0]]
    observed = []
    original = control.Journal.save

    def save(journal, state):
        selected = state.get('operation') is None if completion else (state.get('operation') or {}).get('kind') == 'revoke'
        if not selected:
            return original(journal, state)
        connection = journal.conn
        assert connection.autocommit
        journal.conn = CheckpointFailure(connection, observed)
        try:
            return original(journal, state)
        finally:
            journal.conn = connection

    with patch.object(control.Journal, 'save', save):
        try:
            lifecycle.revoke('operator', target)
        except RuntimeError as error:
            assert str(error) == 'synthetic-checkpoint-unavailable'
        else:
            raise AssertionError('checkpoint failure was not propagated')
    assert observed == ['checkpoint-update-after-durable-head']
    boundary = checkpoint_state(env)
    assert boundary['head_sequence'] == boundary['sequence'] + 1
    assert boundary['head_digest'] != boundary['digest']
    assert_normal_load_denied(env)
    if not completion:
        try:
            lifecycle.revoke('operator', ACTORS[KEYS[1]])
        except Rejected as error:
            assert str(error) == 'operation_unresolved'
        else:
            raise AssertionError('a different target repaired pending revocation')
        assert checkpoint_state(env) == boundary
        assert_normal_load_denied(env)
    lifecycle.revoke('operator', target)
    with authority.connect(env) as connection:
        state = control.Journal(env, connection).load()
        assert state['operation'] is None and target in state['revoked_actors']
    repaired = checkpoint_state(env)
    assert repaired['head_sequence'] == repaired['sequence']
    assert repaired['head_digest'] == repaired['digest']
    return {'failed_stage': 'completion' if completion else 'intent',
        'durable_head_ahead_by': boundary['head_sequence'] - boundary['sequence'],
        'normal_load_denied': True, 'different_target_repair': 'DENIED' if not completion else 'NOT_RUN',
        'same_target_retry_reconciled': True, 'tenant': fresh_denied(env, target),
        'dependent_audience': fresh_denied(env, COMPANY), 'unaffected': unaffected(env)}


def run(env, record):
    record('R12:intent-checkpoint-crash', lambda: inject_and_resume(env, False))
    record('R13:completion-checkpoint-crash', lambda: inject_and_resume(env, True))
