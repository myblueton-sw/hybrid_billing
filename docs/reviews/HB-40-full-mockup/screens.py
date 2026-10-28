from html import escape


def additions(locale):
    def t(en, ko):
        return ko if locale == 'ko' else en

    def table(headers, rows):
        return '<div class="table-scroll"><table><thead><tr>' + ''.join(f'<th>{x}</th>' for x in headers) + '</tr></thead><tbody>' + ''.join('<tr>' + ''.join(f'<td>{v}</td>' for v in row) + '</tr>' for row in rows) + '</tbody></table></div>'

    def heading(title, lead):
        return f'<h1>{title}</h1><p class="lead">{lead}</p>'

    def facts(pairs):
        return '<dl class="facts">' + ''.join(f'<div><dt>{a}</dt><dd>{b}</dd></div>' for a, b in pairs) + '</dl>'

    register = heading(t('Contracts and rates', '계약·요율 관리'), t('Supplier purchase agreements and customer sales agreements have separate prices, terms and owners.', '공급자 매입 계약과 고객 판매 계약의 요율·기간·담당자를 구분합니다.'))
    register += '<div class="tabs"><b>' + t('Supplier purchase contracts', '공급자 매입 계약') + '</b><a href="#sales-policy">' + t('Customer sales contracts', '고객 판매 계약') + '</a></div>'
    register += '<div class="bar"><label>' + t('Find agreement', '계약 찾기') + '<input placeholder="' + t('Contract ID / supplier / purchasing account', '계약 ID / 공급자 / 매입 계정') + '"></label><label>' + t('Status', '상태') + '<select><option>' + t('All statuses', '전체 상태') + '</option></select></label></div>'
    register += table([t('Supplier / agreement', '공급자 / 계약'), t('Account / legal entity', '계정 / 계약 법인'), t('Validity / version', '유효기간 / 버전'), t('Rate basis', '요율 기준'), t('Evidence / responsible', '근거 / 담당')], [
        ['<a href="#supplier-aws">AWS · DEMO-AWS-01</a>', 'DEMO-PAYER-A / Meridian MSP', '2026-01-01 → 2026-12-31 · R3', t('SKU eligibility / private rate schedule', 'SKU 적용 범위 / 개별 요율표'), t('Rate schedule review · Alex Morgan', '요율표 검토 중 · Alex Morgan')],
        ['<a href="#supplier-azure">Azure · DEMO-AZ-01</a>', 'DEMO-EA-A / Meridian MSP', '2026-04-01 → 2027-03-31 · R1', t('Agreement-specific negotiated prices', '계약 유형별 협의 단가'), t('Dataset mapping review · Jamie Kim', '데이터 매핑 검토 · Jamie Kim')],
        ['<a href="#supplier-gcp">GCP · DEMO-GCP-01</a>', 'DEMO-BA-A / Meridian MSP', '2026-01-01 → 2026-12-31 · R2', t('Account pricing / consumption model', '계정 요율 / 사용 모델'), t('Historical price evidence review · Sam Lee', '과거 가격 근거 검토 · Sam Lee')],
        ['<a href="#supplier-oci">OCI · DEMO-OCI-CONTRACT-01</a>', 'DEMO-OCI-SUB-01 / Meridian MSP', '2026-01-01 → 2026-12-31 · R1', 'Universal Credits / ' + t('contract net rate', '계약 순단가'), t('Synthetic reconciliation ready · Alex Morgan', '합성 예시 대사 준비 · Alex Morgan')],
        ['<a href="#supplier-alibaba">Alibaba · DEMO-ALI-01</a>', 'DEMO-UID-A / Meridian MSP', '2026-07-01 → 2027-06-30 · R1', t('Reseller terms / eligible coupons', '리셀러 조건 / 적격 쿠폰'), t('Settlement role review · Jamie Kim', '정산 역할 검토 · Jamie Kim')]
    ])
    register += '<h2>' + t('Renewals and exceptions', '갱신 및 예외 관리') + '</h2>' + table([t('Work', '작업'), t('Target', '대상'), t('Next action', '다음 처리')], [[t('Upcoming rate revision', '예정 요율 개정'), 'DEMO-AWS-01 · R4', t('Review effective date and impacted draft invoices', '적용일과 영향받는 초안 인보이스 검토')], [t('Customer-specific standing exception', '고객별 상시 예외'), 'C00226 · CUSTOMER-R3', '<a href="#sales-policy">' + t('Inspect inherited and additional rules', '기본 정책과 추가 규칙 확인') + '</a>']])
    register += '<p class="note">' + t('All IDs, dates, review states and names are fictional design fixtures. These rows do not establish provider connectivity or actual negotiated rights.', '모든 ID·기간·검토 상태·이름은 설계용 합성 예시입니다. 실제 공급자 연결이나 협의된 계약 권한을 의미하지 않습니다.') + '</p>'

    oci = heading(t('OCI purchase contract', 'OCI 매입 계약 상세'), 'DEMO-OCI-CONTRACT-01 · Meridian MSP · Universal Credits')
    oci += '<div class="tabs"><b>' + t('Rates and validity', '요율·유효기간') + '</b><span>' + t('Commitment usage', '약정 사용') + '</span><span>' + t('Evidence and revisions', '근거·변경 이력') + '</span></div>'
    oci += facts([(t('Subscription / commitment', '구독 / 약정'), 'DEMO-OCI-SUB-01 / DEMO-COMMIT-01'), (t('Term / rate version', '기간 / 요율 버전'), '2026-01-01 → 2026-12-31 · R1'), (t('Purchasing account / currency', '매입 계정 / 통화'), 'DEMO-TENANCY-A · USD'), (t('Statement period', '명세 기간'), '2026-09 · DEMO-STMT-09')])
    oci += '<h2>' + t('Rate and observed consumption', '요율과 실제 사용 비용 대조') + '</h2>' + table([t('SKU / unit', 'SKU / 단위'), t('Quantity', '수량'), t('Contract net unit price', '계약 순단가'), t('Reported cost', '보고 비용'), t('Difference', '차이')], [['DEMO-SKU-01 / ' + t('synthetic unit', '합성 단위'), '1,000', 'USD 0.08', 'USD 80.00', 'USD 0.00']])
    oci += '<div class="metrics"><div>' + t('Opening eligible balance', '기초 적격 잔액') + '<strong>10,000.00</strong><small>USD</small></div><div>' + t('Commitment consumed', '약정 사용액') + '<strong>80.00</strong><small>USD</small></div><div>' + t('Closing eligible balance', '기말 적격 잔액') + '<strong>9,920.00</strong><small>USD</small></div><div>' + t('Included overage', '비용에 포함된 초과 사용') + '<strong>0.00</strong><small>USD</small></div></div>'
    oci += table([t('Evidence', '근거'), t('Reference / status', '참조 / 상태'), t('Treatment', '처리')], [[t('Source report', '원천 보고서'), 'DEMO-REPORT-09 · S1', t('80 already reflects the fictional negotiated rate', '80에 가상 계약 요율이 이미 반영됨')], [t('Usage statement', '사용 명세서'), 'DEMO-STMT-09 · R1', '10,000 − 80 = 9,920'], [t('Supplier invoice / settlement', '공급자 인보이스 / 정산'), t('Not loaded', '미조회'), t('Prepaid invoice and cash settlement remain separate evidence', '선불 인보이스와 납부 상태는 별도 근거 확인')]])
    oci += '<p class="note">' + t('Fixture assumptions: all 80 is eligible and covered; no other movements, expiry or adjustments. Incremental uncovered usage charge is USD0, not an invoice total or payment-complete claim. This purchase example does not determine any customer selling price.', '예시 가정: 사용액 80 전액이 적격 선불 약정으로 충당되며, 다른 잔액 변동·만료·조정은 없습니다. 추가 미충당 사용 요금 0은 인보이스 총액이나 납부 완료를 뜻하지 않습니다. 이 매입 예시로 고객 판매액을 결정하지 않습니다.') + '</p>'

    modes = heading(t('Usage mode and license', '이용 방식과 라이선스'), t('Record the permitted use and its evidence. This is separate from sign-in and billing permissions.', '허용된 이용 목적과 근거를 기록합니다. 로그인 방식 및 빌링 권한과는 별개입니다.'))
    modes += table([t('Mode', '이용 방식'), t('Permission evidence', '이용 근거'), t('What the user sees', '표시 내용')], [[t('Noncommercial', '비상업적 이용'), t('Permitted purpose under the public license', '공개 라이선스에서 허용된 목적'), t('No 14-day countdown or paid-key requirement', '14일 카운트다운·유료 키 요구 없음')], [t('Commercial evaluation', '상업적 평가'), t('Separate permission and accepted evaluation terms', '별도 평가 허가와 동의한 조건'), t('Authorized start/end and remaining time', '허가된 시작·종료 시각과 남은 기간')], [t('Purchased commercial', '구매한 상업적 이용'), t('Commercial agreement and verified signed entitlement', '상업 계약 및 검증된 서명 라이선스'), t('Licensed company, scope and validity', '계약 회사·허용 범위·유효기간')]])
    modes += '<h2>' + t('Selected mode: noncommercial', '선택한 이용 방식: 비상업적 이용') + '</h2>' + facts([(t('Declared purpose', '신고한 목적'), t('Personal noncommercial evaluation · example', '개인 비상업적 검토 · 예시')), (t('Commercial evaluation grant', '상업적 평가 허가'), t('Not granted', '부여되지 않음'))])
    modes += '<p>' + t('Selecting a mode does not grant new rights or prove actual use. Employer name, corporate email and IP address are not automatic commercial-use tests.', '방식 선택만으로 새로운 이용 권한이 생기거나 실제 목적이 입증되지는 않습니다. 회사명·기업 이메일·IP로 상업적 이용을 자동 판정하지 않습니다.') + '</p><a href="#license-status">' + t('Review key registration and expiry states →', '키 등록 및 만료 상태 보기 →') + '</a>'

    status = heading(t('License status and registration', '라이선스 상태 및 키 등록'), t('Example: an expressly permitted evaluation has ended. History remains available.', '예시: 별도로 허가받은 평가 기간이 종료되었습니다. 기존 이력은 계속 조회할 수 있습니다.'))
    status += '<section class="panel"><h2>' + t('Commercial evaluation ended', '상업적 평가 기간이 종료되었습니다') + '</h2><p>' + t('The commercial grant has ended. Obtain permission before further commercial use. This notice does not block records or exports.', '상업적 평가 허가가 종료되었습니다. 상업적으로 계속 이용하려면 별도 허가를 받으세요. 이 알림은 기록 조회와 내보내기를 차단하지 않습니다.') + '</p><p><a href="mailto:myblueton@gmail.com">myblueton@gmail.com</a></p></section>'
    status += '<div class="bar"><label>' + t('Signed license file / key', '서명된 라이선스 파일 / 키') + '<input placeholder="' + t('No real key included in this design', '이 시안에는 실제 키가 없습니다') + '"></label><button>' + t('Verify registration', '등록 검증') + '</button></div>'
    status += table([t('Verification outcome', '검증 결과'), t('Required display', '표시 방식')], [[t('Verified', '검증됨'), t('Company, agreement, scope, term and verification time', '회사·계약·범위·기간·검증 시각')], [t('Invalid signature / altered key', '서명 오류 / 변조된 키'), t('Reject new registration; preserve current valid entitlement', '신규 등록 거부 · 기존 유효한 라이선스 보존')], [t('Unable to verify', '검증 불가'), t('Show uncertainty; do not call network failure expiry', '미확인 표시 · 네트워크 오류를 만료로 판정하지 않음')], [t('Expired', '만료'), t('Persistent nonblocking notice and user-initiated contact', '지속적인 비차단 알림 및 사용자가 선택하는 연락')]])
    status += '<p class="note">' + t('No key verification service, telemetry, automatic email, remote lock or data deletion is implemented or authorized by this screen. Noncommercial permitted use has no trial expiry.', '이 화면은 키 검증 서비스·추적·자동 메일·원격 잠금·데이터 삭제를 구현하거나 허가하지 않습니다. 허용된 비상업적 이용에는 체험 만료를 적용하지 않습니다.') + '</p>'
    return {'supplier-register': register, 'supplier-oci': oci, 'license-modes': modes, 'license-status': status}
