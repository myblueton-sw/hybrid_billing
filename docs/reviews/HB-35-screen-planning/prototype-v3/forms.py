from html import escape
from content import t

def forms(locale):
    def tr(en, ko):
        return escape(ko if locale == 'ko' else en)
    def select(name, label, choices):
        opts=''.join(f'<option value="{v}">{tr(en,ko)}</option>' for v,en,ko in choices)
        return f'<label>{tr(*label)}<select name="{name}">{opts}</select></label>'
    def field(name,label,value,kind='text'):
        return f'<label>{tr(*label)}<input name="{name}" type="{kind}" value="{value}" required></label>'
    def button(action,en,ko):
        return f'<button type="button" data-action="{action}">{tr(en,ko)}</button>'
    scope=field('start',('Effective from','적용 시작일'),'2026-09-01','date')+field('end',('Effective until (exclusive)','적용 종료일 (미포함)'),'2026-10-01','date')
    return {
        'intake':select('evidence',('Review incoming evidence','추가 원본 검토'),[('wrong','Wrong period / October file','기간 불일치 / 10월 파일'),('valid','F1 / September fixed fee / USD 1,000','F1 / 9월 고정 비용 / USD 1,000')])+button('intake','Accept reviewed evidence · demo','검토 원본 수용 · 시연'),
        'mapping':select('customer',('Customer assignment','고객 귀속'),[('','Unassigned','미귀속'),('Aster','Aster / ACCT-A17','Aster / ACCT-A17')])+scope+button('mapping','Preview and save mapping draft','귀속 미리보기·초안 저장'),
        'pricing':select('discount',('Usage discount','사용료 할인'),[('10','10% / proposed contract','10% / 제안 계약'),('0','0% / original contract','0% / 기존 계약')])+scope+select('overlap',('Broad rule R-ALL','포괄 규칙 R-ALL'),[('conflict','Overlaps R-J10 / blocked','R-J10 중복 / 차단'),('resolved','Exclude R-J10 scope / Marketplace excluded','R-J10 범위 제외 / Marketplace 할인 제외')])+button('pricing','Preview and save price draft','요율 미리보기·초안 저장'),
        'seats':select('policy',('Proration policy','일할 정책'),[('','Not selected / blocked','미지정 / 차단'),('actual','DAY-30 / actual September days','DAY-30 / 9월 실제 일수')])+button('seats','Verify quantity adjustment','좌석 조정 검증'),
        'commitments':select('model',('GCP credit mapping','GCP 크레딧 매핑'),[('generic','Generic discount / unresolved','일반 할인 / 미해결'),('typed','FEE_UTILIZATION_OFFSET / identified model','FEE_UTILIZATION_OFFSET / 지정 모델')])+button('commitments','Verify typed treatment','유형별 처리 검증'),
        'fx':select('rate',('Selling snapshot','판매 환율 스냅샷'),[('0.90','FX-J1 / 2026-09-30 / 0.90 EUR per USD','FX-J1 / 2026-09-30 / USD당 EUR 0.90'),('0.91','FX-J2 / proposed replacement / 0.91','FX-J2 / 대체 제안 / 0.91'),('0','Invalid raw rate / 0','무효 원시 환율 / 0')])+button('fx','Preview currency and approval impact','환산·승인 영향 미리보기'),
        'runs':select('mode',('Processing scope','처리 범위'),[('pinned','Recalculate accepted SRC-A2 / MAP-A2','수용된 SRC-A2 / MAP-A2 재계산'),('refresh','Acquire new source / SRC-A3 candidate','새 원본 수집 / SRC-A3 후보')])+button('readback','Read back unknown R-A1','결과 미확인 R-A1 조회')+button('queue','Queue corrected R-A2 · demo','수정 R-A2 대기 등록 · 시연')+button('advance','Advance staged run','실행 단계 진행'),
        'approval':field('reason',('Return reason','반려 사유'),'')+button('reject','Return A2 for correction','A2 보완 요청')+button('attach','Attach F1 / reverify A3','F1 첨부 / A3 재검증')+button('approve','Approve fixed A3 · demo','고정 A3 승인 · 시연')+button('stale','Simulate changed inputs','입력 변경 시연')+button('issue','Issue approved A3 · demo','승인된 A3 발행 · 시연'),
        'correction':select('treatment',('Contract treatment','계약 처리'),[('pending','Unresolved / no customer amount','미해결 / 고객 금액 미확정'),('pass','CREDIT-NS-1 / approved 100% pass-through','CREDIT-NS-1 / 승인된 100% 전달')])+button('creditcheck','Verify correction c1','정정 c1 검증')+button('creditapprove','Approve fixed c1 · demo','고정 c1 승인 · 시연')+button('creditissue','Issue linked credit · demo','연결 정정 문서 발행 · 시연'),
        'history':select('document',('Select document','문서 선택'),[('original','DEMO-2026-09104 / original','DEMO-2026-09104 / 원본'),('credit','CN-NS-1007 / correction','CN-NS-1007 / 정정')])+button('download','Download sample summary (.txt)','예시 문서 요약 다운로드 (.txt)')+button('detail','Download sample selling detail (.csv)','예시 판매 상세 다운로드 (.csv)'),
        'inquiry':'<label>'+tr('Your question','문의 내용')+'<textarea name="question" rows="3"></textarea></label>'+button('inquiry','Confirm local inquiry example','로컬 문의 예시 확인'),
        'response':button('acknowledge','Acknowledge explanation · demo','설명 확인 · 시연')
    }
