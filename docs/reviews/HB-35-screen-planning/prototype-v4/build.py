from pathlib import Path
from html import escape
import json
import shutil
from content import SCREENS, MONTHS

SOURCE=Path(__file__).resolve().parent
ROOT=SOURCE.parents[3]

def build(locale):
    out=ROOT/'output/designs'/('HB-35-v4'+('-ko' if locale=='ko' else ''))
    out.mkdir(parents=True,exist_ok=True)
    def value(v):
        return v[locale] if isinstance(v,dict) else v
    def text(v):
        return escape(str(value(v)))
    def tr(en,ko):
        return escape(ko if locale=='ko' else en)
    def button(action,en,ko):
        return f'<button type="button" data-action="{action}">{tr(en,ko)}</button>'
    def select(name,label,options):
        return '<label>'+tr(*label)+f'<select name="{name}">'+''.join(f'<option value="{v}">{tr(en,ko)}</option>' for v,en,ko in options)+'</select></label>'
    period=select('month',('Period','기간'),[('all','All months','전체 월')]+[(m['month'],m['month'],m['month']) for m in MONTHS])
    basis=select('basis',('Amount basis','금액 기준'),[('original','As originally closed','당시 마감 원본'),('adjusted','With linked corrections / as of Oct 28','연결 정정 반영 / 10월 28일 기준')])
    forms={
        'monthly':period+basis+button('filter','Apply view','조회 적용'),
        'month':button('immutable','Check closed-month protection','마감 원본 보호 확인')+'<a class="button" href="previous/core-customer.html#invoice">'+tr('September original customer bill','9월 원본 고객 청구서')+'</a><a class="button" href="previous/index.html#correction">'+tr('September linked correction','9월 연결 정정')+'</a>',
        'variance':basis+button('variance','Update comparison basis','비교 기준 적용'),
        'inbox':'<label>'+tr('Select a local PDF · metadata only','로컬 PDF 선택 · 메타데이터만 확인')+'<input type="file" name="pdf" accept="application/pdf,.pdf"></label>'+select('sample-state',('Synthetic sample branch','합성 예시 분기'),[('ready','Cedar / ready for review','Cedar / 검토 가능'),('duplicate','Same fingerprint / duplicate','동일 지문 / 중복'),('encrypted','Encrypted / password needed','암호화 / 해제 필요'),('unsupported','Unsupported document / manual review','미지원 문서 / 수동 검토'),('failed','OCR failed / unreadable page','OCR 실패 / 판독 불가')])+button('sample','Load synthetic PDF analysis','합성 PDF 분석 예시 불러오기'),
        'extract':'<label>'+tr('Support amount / p1 R3 / USD','지원 비용 / p1 R3 / USD')+'<input name="support" inputmode="decimal" value="400.00"></label><label>'+tr('Correction reason','교정 사유')+'<input name="reason" value=""></label><label class="check"><input type="checkbox" name="source-checked">'+tr('I checked page 1, row R3','1페이지 R3 원본 확인')+'</label>'+button('verify','Save E2 and verify sums','E2 저장·합계 검증'),
        'differences':select('detail',('Supplemental detail','보완 상세'),[('missing','C2 not supplied / blocked','C2 미확보 / 차단'),('wrong','Wrong period evidence / blocked','기간 불일치 근거 / 차단'),('correct','C2-R2 +200 and C2-R3 4,000 / September USD','C2-R2 +200 및 C2-R3 4,000 / 9월 USD')])+button('reconcile','Reconcile corrected detail','보완 상세 재대사'),
        'audit':button('accept','Accept fixed evidence snapshot · demo','고정 근거 스냅샷 수용 · 시연')+button('revise','Start a new extraction revision','새 추출 버전 시작')+'<a class="button" href="#extract">'+tr('Return to field review','필드 검토로 돌아가기')+'</a>',
        'customer-months':select('customer-month',('Choose a service month','서비스 월 선택'),[(m['month'],m['month'],m['month']) for m in MONTHS])+button('customer-month','View selected month','선택 월 보기')+button('download','Download monthly sample (.csv)','월별 예시 다운로드 (.csv)')+'<a class="button" href="previous/customer.html#history">'+tr('September original and credit','9월 원본·정정 문서')+'</a>'
    }
    for customer,filename in [(False,'index.html'),(True,'customer.html')]:
        screens=[s for s in SCREENS if bool(s.get('customer'))==customer]
        nav=''.join(f'<a href="#{s["id"]}">{i+1:02d} {text(s["title"])}</a>' for i,s in enumerate(screens))
        sections=[]
        for i,s in enumerate(screens):
            headings=''.join('<th>'+text(c)+'</th>' for c in value(s['columns']))
            rows=''.join('<tr>'+''.join('<td>'+text(c)+'</td>' for c in row)+'</tr>' for row in s['rows'])
            if s['kind'] in ['monthly','customer-months']:
                rows=''
                for m in MONTHS:
                    ready=m['sale'] is not None
                    fmt=lambda amount:f'{amount:,.2f}' if amount is not None else tr('Not confirmed','미확정')
                    if customer:
                        cells=[m['month'],fmt(m['sale']),fmt(m['credit']),fmt(m['sale']+m['credit']) if ready else tr('Not issued','미발행'),tr('Issued','발행됨') if ready else tr('Open / incomplete','진행 중 / 자료 미완료')]
                    else:
                        cells=[m['month'],fmt(m['cost']),fmt(m['sale']),fmt(m['sale']-m['cost']) if ready else tr('Unavailable','산정 불가'),f'<button type="button" data-open-month="{m["month"]}">'+tr('Open month','월 상세')+'</button>']
                    rows+=f'<tr data-month="{m["month"]}">'+''.join('<td>'+c+'</td>' for c in cells)+'</tr>'
            table=f'<div class="table-scroll" tabindex="0"><table><thead><tr>{headings}</tr></thead><tbody>{rows}</tbody></table></div>'
            notes='<aside class="notes"><h2>'+tr('Review context','검토 근거')+'</h2>'+''.join('<p>'+text(n)+'</p>' for n in s['notes'])+'</aside>'
            controls='<div class="controls"><h2>'+tr('Review and action','검토 및 작업')+'</h2><form>'+forms[s['kind']]+'</form><div class="result" data-result role="status" aria-live="polite">'+tr('Awaiting review. All figures are synthetic design data.','검토 대기. 모든 금액은 합성 시안 데이터입니다.')+'</div></div>'
            if s['kind']=='extract':
                preview='<div class="source-pane"><h2>'+tr('Original PDF / page 1','원본 PDF / 1페이지')+'</h2><a href="sample-cedar-aws-09.pdf" target="_blank" rel="noopener"><img src="sample-source.png" alt="'+tr('Synthetic invoice original: compute 40,000, storage 20,000, support 4,000, total USD 64,000','합성 청구서 원본: 컴퓨팅 40,000 · 스토리지 20,000 · 지원 비용 4,000 · 총액 USD 64,000')+'"></a><p>'+tr('Synthetic supplier-format example, not a real vendor invoice. Open PDF for full-page inspection.','공급자 형식의 합성 예시이며 실제 공급자 발행 문서가 아닙니다. PDF를 열어 원문 전체를 확인하세요.')+'</p><a class="button" href="sample-cedar-aws-09.pdf" download>'+tr('Download sample PDF','예시 PDF 다운로드')+'</a></div>'
                main='<div class="three-pane">'+preview+'<div><h2>'+tr('Extracted fields / E1','추출 항목 / E1')+'</h2>'+table+controls+'</div>'+notes+'</div>'
            else:
                main='<div class="work-grid"><div>'+table+controls+'</div>'+notes+'</div>'
            prior='previous/customer.html#history' if customer else 'previous/index.html#intake'
            target=('#'+s['next'])
            nextlabel=tr('Switch to separate Cedar PDF case','별도 Cedar PDF 사례로 전환') if s['id']=='month' else tr('Continue','다음 단계')
            sections.append(f'''<section class="screen v4" id="{s['id']}" tabindex="-1"><aside class="sidebar4"><div class="brand4">Hybrid Billing</div><p>{tr('Customer billing','고객 청구') if customer else tr('Monthly close and evidence','월별 마감과 원본 검증')}</p><nav>{nav}</nav><a class="reference" href="{prior}">{tr('Previous workflow examples','기존 작업 흐름 보기')}</a></aside><main class="workspace4"><header><span>{tr('BILLING WORKSPACE / v4','빌링 작업 공간 / v4')}</span><span>{tr('Synthetic local prototype','합성 데이터 로컬 시안')}</span></header><div class="heading4"><p class="eyebrow4">{i+1:02d} / {tr('Monthly evidence trail','월별 근거와 처리 이력')}</p><h1>{text(s['title'])}</h1><p>{text(s['lead'])}</p></div><div class="scope4">{text(s['scope'])}</div>{main}<div class="outcome4"><h2>{tr('Expected result and boundary','기대 결과와 처리 경계')}</h2><p>{text(s['result'])}</p></div><footer><span>{tr('No upload, real AI parsing or financial action is connected.','실제 업로드·AI 파싱·금전 처리는 연결되지 않았습니다.')}</span><a href="{target}">{nextlabel} →</a></footer></main></section>''')
        data=[{k:v for k,v in m.items() if k!='cost'} if customer else m for m in MONTHS]
        payload=json.dumps(data,ensure_ascii=False).replace('<','\\u003c')
        html='<!doctype html><html lang="'+locale+'"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Hybrid Billing v4</title><link rel="stylesheet" href="v4.css"></head><body>'+''.join(sections)+'<script type="application/json" id="month-data">'+payload+'</script><script src="common.js"></script><script src="'+('customer.js' if customer else 'operator.js')+'"></script></body></html>'
        (out/filename).write_text(html)
    for name in ['v4.css','common.js','operator.js','customer.js','source.html']:
        shutil.copyfile(SOURCE/name,out/name)
    previous=ROOT/'output/designs'/('HB-35-v3'+('-ko' if locale=='ko' else ''))
    (out/'previous').mkdir(exist_ok=True)
    for f in previous.iterdir():
        if f.suffix in ['.html','.css','.js']:
            shutil.copyfile(f,out/'previous'/f.name)
    print(out)

if __name__=='__main__':
    for locale in ['en','ko']:
        build(locale)
