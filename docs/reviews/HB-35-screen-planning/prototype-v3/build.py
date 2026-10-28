from pathlib import Path
from html import escape
import shutil
import json
import re
from content import SCREENS
from forms import forms

SOURCE=Path(__file__).resolve().parent
ROOT=SOURCE.parents[3]
OLD=SOURCE.parent/'prototype'

def resolve(value,locale):
    return value[locale] if isinstance(value,dict) else value

def build(locale):
    out=ROOT/'output/designs'/('HB-35-v3'+('-ko' if locale=='ko' else ''))
    out.mkdir(parents=True,exist_ok=True)
    def text(value):
        return escape(str(resolve(value,locale)))
    def tr(en,ko):
        return escape(ko if locale=='ko' else en)
    form_map=forms(locale)
    for customer,filename in [(False,'index.html'),(True,'customer.html')]:
        screens=[s for s in SCREENS if bool(s.get('customer'))==customer]
        nav=''.join(f'<a href="#{s["id"]}">{i+1:02d} {text(s["title"])}</a>' for i,s in enumerate(screens))
        sections=[]
        for i,s in enumerate(screens):
            rows=''.join('<tr>'+''.join(f'<td>{text(c)}</td>' for c in row)+'</tr>' for row in s['rows'])
            heads=''.join(f'<th scope="col">{text(c)}</th>' for c in resolve(s['columns'],locale))
            side=''.join(f'<p>{text(v)}</p>' for v in s['side'])
            target=('customer.html' if s['next'] in ['history','inquiry','response'] else 'index.html')+'#'+s['next']
            prior=('core-customer.html#overview' if customer else 'core-index.html#close')
            case_switch=s['id'] in ['pricing','seats','commitments','fx','approval']
            next_label=tr('Open next example','다음 예시 열기') if case_switch else tr('Continue this case','이 사례 이어가기')
            operator_queue=''
            if s['id']=='correction':
                operator_queue='<div class="queue"><h3>'+tr('Customer inquiry / Q-NS-09','고객 문의 / Q-NS-09')+'</h3><p>'+tr('DEMO-2026-09104 / GCP-09 → billing owner R-LEE → response drafted with CN-NS-1007. Customer preview contains only the service-credit explanation.','DEMO-2026-09104 / GCP-09 → 청구 담당 R-LEE → CN-NS-1007을 연결한 답변 초안. 고객 미리보기에는 서비스 크레딧 설명만 포함합니다.')+'</p><a href="customer.html#response">'+tr('Preview customer-visible response','고객 공개 답변 미리보기')+'</a></div>'
            sections.append(f'''<section class="screen v3 {'customer-v3' if customer else 'operator-v3'}" id="{s['id']}" tabindex="-1">
<aside class="sidebar"><div class="wordmark">Hybrid Billing</div><p>{tr('Customer workspace','고객 청구 공간') if customer else tr('Monthly billing operations','월별 청구 작업')}</p><nav aria-label="{tr('Screen navigation','화면 탐색')}">{nav}</nav><a class="core-link" href="{prior}">{tr('Original billing story','기존 청구 흐름 보기')}</a></aside>
<main class="workspace"><header class="context"><span>{tr('DESIGN REVIEW · v3','화면 검토 · v3')}</span><span>{tr('Synthetic data · local simulation only','합성 데이터 · 로컬 시연 전용')}</span></header>
<div class="page-heading"><div class="step-label">{tr('Customer documents','고객 문서') if customer else tr('Review-driven operating workflow','리뷰 반영 작업 흐름')} / {i+1:02d}</div><h1>{text(s['title'])}</h1><p>{text(s['lead'])}</p></div>
<div class="case-scope">{text(s['scope'])}</div>
<div class="work-grid"><div><div class="table-scroll" role="region" aria-label="{text(s['title'])}" tabindex="0"><table><thead><tr>{heads}</tr></thead><tbody>{rows}</tbody></table></div>
<div class="edit-area"><h2>{tr('Review and act','검토 및 작업')}</h2><form data-flow="{s['form']}" novalidate>{form_map[s['form']]}</form><div class="live-result" role="status" aria-live="polite" data-result>{tr('Ready for your review. Controls simulate only the stated case.','검토 대기. 조작은 표시된 사례만 시연합니다.')}</div></div></div>
<aside class="evidence-panel"><h2>{text(s['side_title'])}</h2>{side}</aside></div>
<div class="outcome"><h2>{tr('Expected outcome and boundary','기대 결과와 처리 경계')}</h2><p>{text(s['result'])}</p></div>{operator_queue}
<footer><span>{tr('No data is sent. No financial action is executed.','데이터 전송·실제 금전 처리는 실행하지 않습니다.')}</span><a href="{target}">{next_label} →</a></footer></main></section>''')
        document='<!doctype html><html lang="'+locale+'"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Hybrid Billing v3</title><link rel="stylesheet" href="v3.css"></head><body>'+''.join(sections)+'<script src="'+('customer.js' if customer else 'operator.js')+'"></script></body></html>'
        (out/filename).write_text(document)
    oldbase=ROOT/'output/designs/HB-35-v2-ko' if locale=='ko' else OLD
    for name in ['index.html','reconciliation.html','customer.html']:
        value=(oldbase/name).read_text()
        for other in ['index.html','reconciliation.html','customer.html']:
            value=value.replace(other+'#','core-'+other+'#')
        (out/('core-'+name)).write_text(value)
    for name in ['style.css','navigation.js']:
        shutil.copyfile(oldbase/name,out/name)
    for name in ['v3.css','operator.js','customer.js']:
        shutil.copyfile(SOURCE/name,out/name)
    (out/'manifest.json').write_text(json.dumps({'locale':locale,'operator':9,'customer':3,'core':15,'screens':27},indent=2))
    return out

if __name__=='__main__':
    for language in ['en','ko']:
        print(build(language))
