import argparse,json,pathlib,re,shutil,subprocess,tempfile
p=argparse.ArgumentParser()
p.add_argument('--output',required=True)
a=p.parse_args()
root=pathlib.Path(__file__).resolve().parents[1]
out=pathlib.Path(a.output).resolve()
out.mkdir(parents=True,exist_ok=True)
needle='return r.check(s, s.requests[expected.Request], expected, now)'
cases=[
 ('skip_current_checks',needle,'_ = now; return nil'),
 ('consume_key_during_check',needle,'s.keys[executedKey(expected)] = expected.ID; '+needle),
 ('historical_success_bypass',needle,'if old := s.executions[expected.ID]; old != nil && old.view.Outcome.Result == ResultSucceeded { return nil }; '+needle),
 ('skip_actor_workspace_validation','if !validID(workspace) || !validActor(actor) {','if false {'),
]
results=[]
with tempfile.TemporaryDirectory(prefix='hb-current-approval-') as temp:
 work=pathlib.Path(temp)
 for name in ['modules/approvals','packages/primitives']:
  shutil.copytree(root/name,work/name)
 for name in ['go.mod','go.sum']:
  if (root/name).exists():shutil.copy2(root/name,work/name)
 path=work/'modules/approvals/check.go';original=path.read_text()
 def run(name):
  result=subprocess.run(['go','test','./modules/approvals','-run','^TestCheckCurrent','-count=1','-timeout=30s'],cwd=work,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
  (out/(name+'.log')).write_text(result.stdout)
  return result
 if run('baseline').returncode:raise RuntimeError('Baseline failed')
 for name,before,after in cases:
  if original.count(before)!=1:raise RuntimeError('Target not unique: '+name)
  try:
   path.write_text(original.replace(before,after));result=run(name)
   outcome='survived' if result.returncode==0 else 'compile_failed' if '[build failed]' in result.stdout else 'killed' if re.search(r'^--- FAIL: TestCheckCurrent',result.stdout,re.M) and 'panic:' not in result.stdout and 'signal:' not in result.stdout else 'execution_failed'
   results.append({'name':name,'outcome':outcome})
  finally:path.write_text(original)
(out/'results.json').write_text(json.dumps({'baseline':'passed','mutants':results},indent=2))
if any(r['outcome']!='killed' for r in results):raise SystemExit(1)
