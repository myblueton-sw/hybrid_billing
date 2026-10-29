from bs4 import BeautifulSoup


def extend_profile(raw, locale):
    t = lambda en, ko: ko if locale == 'ko' else en
    soup = BeautifulSoup(raw, 'html.parser')
    host = soup.find('main') or soup
    for node in host.select('.note'):
        node.decompose()
    heading = soup.new_tag('h2')
    heading.string = t('Payment conditions and master agreement', '결제 조건·방법 및 마스터 계약')
    host.append(heading)
    rows = [
        (t('Business registration number', '사업자등록번호'), t('Not entered · jurisdiction and identifier type required; separate from tax ID', '미입력 · 국가·등록번호 종류 함께 입력 · 세금 식별번호와 별도')),
        (t('Pricing basis / billing cadence', '과금 기준 / 청구 주기'), t('Fixed / usage-based / hybrid × monthly / quarterly / semiannual / other (custom period)', '정액 / 종량제 / 혼합 × 매월 / 분기 / 반기 / 기타(기간 지정)')),
        (t('Billing timing / due-date rule', '청구 시점 / 지급기한'), t('Prepaid / postpaid · current Net 30; custom trigger, days and calendar require agreement', '선불 / 후불 · 현재 Net 30 · 개별 기준일·일수·달력은 계약으로 지정')),
        (t('Settlement schedule', '결제 일정'), t('Single / advance–interim–balance · each stage: trigger, amount or %, due date; validate total', '일시 / 선금–중도금–잔금 · 단계별 조건·금액 또는 비율·기한 · 합계 검증')),
        (t('Deposit deduction', '예치금 차감'), t('Optional combination · confirmed balance only; allocation priority and insufficient-balance handling required', '다른 조건과 조합 가능 · 확인된 잔액만 · 차감 순서·부족분 처리 지정')),
        (t('Payment method', '결제 방법'), t('Bank transfer / credit card / electronic promissory note / other · permitted methods and default', '계좌입금 / 신용카드 거래 / 전자 어음 / 기타 · 허용 방법·기본 방법 지정')),
        (t('Master agreement', '마스터 계약 유무'), t('None / present / unverified · current: unverified; select master ID, signed status, parties, version and dates', '없음 / 있음 / 미확인 · 현재 미확인 · 마스터 계약 ID·체결 상태·당사자·버전·기간 연결')),
        (t('Linked child agreements', '연관 하위 계약'), t('CT-NS-v3 · parent unverified · list child IDs, scope, status, dates and inherited/overridden terms', 'CT-NS-v3 · 상위 미확인 · 하위 계약 목록: ID·범위·상태·기간·상속/별도 조건')),
    ]
    table = '<table><thead><tr><th>' + t('Field', '항목') + '</th><th>' + t('Configurable options / current state', '설정 가능한 옵션 / 현재 상태') + '</th></tr></thead><tbody>'
    table += ''.join('<tr><td>' + a + '</td><td>' + b + '</td></tr>' for a, b in rows) + '</tbody></table>'
    host.append(BeautifulSoup(table, 'html.parser'))
    note = soup.new_tag('p', attrs={'class': 'note'})
    note.string = t('Options shown, not active selections. Deposit deduction is not a discount; method selection does not execute payment. Unknown required terms block issuance. Approved child exceptions retain provenance; master changes never rewrite issued snapshots. Open linked contracts to review effective terms.', '선택 가능한 옵션이며 모두 적용된 상태가 아닙니다. 예치금 차감은 할인이 아니며 결제 방법 선택으로 결제가 실행되지 않습니다. 필수 조건 미확정 시 발행을 보류합니다. 하위 계약 예외의 근거를 남기고 마스터 변경으로 발행 이력을 덮어쓰지 않습니다.')
    host.append(note)
    return str(soup)
