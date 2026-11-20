import argparse,json,pathlib,re,shutil,subprocess,tempfile
p=argparse.ArgumentParser()
p.add_argument('--output',required=True)
p.add_argument('--only',default='')
a=p.parse_args()
root=pathlib.Path(__file__).resolve().parents[1]
out=pathlib.Path(a.output).resolve()
out.mkdir(parents=True,exist_ok=True)
cases=[
 ('skip_gate','approvalread.go','if !decision.Allowed {','if !decision.Allowed && false {'),
 ('skip_snapshot','approvalread.go','if (q.Has("snapshot") && q.Get("snapshot") != snapshot) || (q.Has("permission_revision") && q.Get("permission_revision") != permission) {','if false {'),
 ('skip_mac','approvalread.go','!hmac.Equal([]byte(q.Get("cursor")), []byte(s.approvalCursor(offset, snapshot)))','false'),
 ('broaden_workspace','approvalrows.go','if index.Workspace != ws {','if false {'),
 ('ignore_policy_revision','approvalrows.go','p.Revision != v.Policy','false'),
 ('ignore_target_mismatch','approvalrows.go','v.Subject.Target != index.Target','false'),
 ('accept_partial_policy','approvalrows.go','p.Approvals != 1','false'),
 ('unknown_as_zero','approvalrows.go','Candidate: approvalString(v.Subject.Amount)','Candidate: approvalString("0")'),
 ('wrong_decider','approvalrows.go','row.Approver = approvalPerson(d.Actor)','row.Approver = approvalPerson(v.Requester)'),
 ('wrong_row_bound','approvalread.go','end := min(offset+25, len(filtered))','end := min(offset+24, len(filtered))'),
]
if a.only:cases=[c for c in cases if c[0] in a.only.split(',')]
results=[]
with tempfile.TemporaryDirectory(prefix='hb-approval-mutations-') as temp:
 work=pathlib.Path(temp)
 for name in ['modules','apps/api','packages/primitives']:
  shutil.copytree(root/name,work/name)
 for name in ['go.mod','go.sum']:
  if (root/name).exists():shutil.copy2(root/name,work/name)
 def run(name):
  result=subprocess.run(['go','test','./apps/api/server','-run','^TestApprovalRead','-count=1'],cwd=work,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
  (out/(name+'.log')).write_text(result.stdout)
  return result
 baseline=run('baseline')
 if baseline.returncode:raise RuntimeError('Mutation baseline failed')
 for name,file,before,after in cases:
  path=work/'apps/api/server'/file
  original=path.read_text()
  if original.count(before)!=1:raise RuntimeError('Mutation target not unique: '+name)
  try:
   path.write_text(original.replace(before,after))
   result=run(name)
   outcome='survived' if result.returncode==0 else ('compile_failed' if '[build failed]' in result.stdout else 'killed' if re.search(r'^--- FAIL: TestApprovalRead',result.stdout,re.M) and 'panic:' not in result.stdout and 'signal:' not in result.stdout else 'execution_failed')
   results.append({'name':name,'outcome':outcome})
  finally:path.write_text(original)
(out/'results.json').write_text(json.dumps({'baseline':'passed','mutants':results},indent=2))
if any(x['outcome']!='killed' for x in results):raise SystemExit(1)
