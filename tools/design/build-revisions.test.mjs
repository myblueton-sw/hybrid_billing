import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import os from 'node:os';
import {mkdtemp,writeFile,rm} from 'node:fs/promises';
import {buildRevisions} from './build-revisions.mjs';
const catalog="export const groups=[{id:'overview',label:'Overview'}];\nexport const screens=[{id:'contents',kind:'overview'},{id:'setup-profile',kind:'form'},{id:'assistant',kind:'form'},{id:'entry-login',kind:'form'}];\n";
const names=['app.js','widgets.js','copy.js','data.js','style.css','index.html','design-review.js','design-review.css','review-store.js','review-copy.js','design-directory.js','onboarding.js','analysis.js','screen-designs.js'];
async function fixture(t,{lineEnding='\n',changed={},catalogText=catalog}={}){
 const directory=await mkdtemp(path.join(os.tmpdir(),'hb-revisions-'));
 t.after(()=>rm(directory,{recursive:true,force:true}));
 await writeFile(path.join(directory,'package.json'),JSON.stringify({type:'module'}));
 await writeFile(path.join(directory,'catalog.js'),catalogText.replaceAll('\n',lineEnding));
 for(const name of names)await writeFile(path.join(directory,name),(changed[name]??'first\nsecond\n').replaceAll('\n',lineEnding));
 return directory;
}
const foundations=new Map([['design-system/fonts.css','a'.repeat(64)]]);
test('revision identity is deterministic across LF and CRLF checkouts',async t=>{
 const a=await fixture(t),b=await fixture(t,{lineEnding:'\r\n'});
 assert.deepEqual(await buildRevisions(a,foundations),await buildRevisions(b,foundations));
});
test('dedicated screen changes affect only dependent screen revisions',async t=>{
 const a=await fixture(t),b=await fixture(t,{changed:{'analysis.js':'changed\n'}});
 const before=(await buildRevisions(a,foundations)).screens,after=(await buildRevisions(b,foundations)).screens;
 assert.notEqual(before.assistant.revision,after.assistant.revision);
 for(const id of ['contents','setup-profile','entry-login'])assert.equal(before[id].revision,after[id].revision);
});
test('navigation and shared foundation changes invalidate dependent fingerprints',async t=>{
 const a=await fixture(t),b=await fixture(t,{catalogText:catalog.replace("label:'Overview'","label:'New title'")});
 const before=(await buildRevisions(a,foundations)).screens,nav=(await buildRevisions(b,foundations)).screens;
 const font=(await buildRevisions(a,new Map([['design-system/fonts.css','b'.repeat(64)]]))).screens;
 for(const id of Object.keys(before)){assert.notEqual(before[id].revision,nav[id].revision);assert.notEqual(before[id].revision,font[id].revision);}
});
test('directory source changes are covered by directory screen fingerprints',async t=>{
 const a=await fixture(t),b=await fixture(t,{changed:{'design-directory.js':'changed\n'}});
 const before=(await buildRevisions(a,foundations)).screens,after=(await buildRevisions(b,foundations)).screens;
 assert.notEqual(before.contents.revision,after.contents.revision);assert.equal(before.assistant.revision,after.assistant.revision);
});
