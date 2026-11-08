import {createHash} from 'node:crypto';
import {readFile,readdir} from 'node:fs/promises';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
const digest=value=>createHash('sha256').update(value).digest('hex');
export async function buildRevisions(source,files){
 const {screens}=await import(pathToFileURL(path.join(source,'catalog.js')));
 const names=(await readdir(source)).filter(name=>/\.(js|css|html)$/.test(name)).sort();
 const normalized={};
 for(const name of names)normalized[name]=digest((await readFile(path.join(source,name),'utf8')).replace(/\r\n?/g,'\n'));
 const shared=['app.js','widgets.js','copy.js','data.js','style.css','index.html','design-review.js','design-review.css','review-store.js','review-copy.js'];
 const foundation=[...files].filter(([name])=>name.startsWith('design-system/')).sort(([a],[b])=>a.localeCompare(b));
 const revisions={};
 for(const screen of screens){
  let specific;
  if(['overview','contents'].includes(screen.id))specific=['design-directory.js'];
  else if(screen.id.startsWith('setup-'))specific=names.filter(name=>name.startsWith('onboarding'));
  else if(screen.id==='assistant')specific=names.filter(name=>name.startsWith('analysis'));
  else if(screen.id==='community')specific=['community.js'];
  else if(screen.kind==='report'||screen.id==='index-approval'||screen.id==='core-customer-overview')specific=['journeys.js','journeys-copy.js'];
  else specific=names.filter(name=>name.startsWith('screen-design'));
  const dependencies=Object.fromEntries([...new Set([...shared,...specific])].sort().map(name=>[name,normalized[name]]));
  dependencies['catalog-entry']=digest(JSON.stringify(screen));
  dependencies['navigation-catalog']=normalized['catalog.js'];
  dependencies.foundations=digest(JSON.stringify(foundation));
  revisions[screen.id]={revision:digest(JSON.stringify(dependencies)),dependencies};
 }
 return {version:1,screens:revisions};
}
