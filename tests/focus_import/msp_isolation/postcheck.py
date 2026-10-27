import json

from database_cases import rejected
from fixture_data import COMPANY
from runtime import isolated, save
from modules.focus_import import collection, model


def main():
    with isolated():
        reason = rejected(lambda: collection.query(COMPANY), 'view_member_denied')
    bindings = json.loads(model.BINDINGS.read_text())
    assert len(bindings['views']) == 1
    counts = {actor: collection.query(actor, limit=1)['row_count'] for actor in sorted(bindings['actors'])}
    union = collection.query(bindings['views'][0], limit=1)['row_count']
    assert sorted(counts.values()) == [2, 9, 1112] and union == 1121
    body = {'status': 'PASS', 'revoked_member_denies_whole_company_view': reason,
        'default_legacy_tenant_row_counts': sorted(counts.values()), 'default_company_rows': union,
        'legacy_customer_metadata_isolation_certified': False}
    save('postcheck.json', body)
    print(json.dumps(body, sort_keys=True))


if __name__ == '__main__':
    main()
