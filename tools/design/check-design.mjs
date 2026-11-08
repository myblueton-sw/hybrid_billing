import {readFile,readdir} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
import {createHash} from 'node:crypto';
export const root=fileURLToPath(new URL('../../',import.meta.url));
export const source=path.join(root,'docs/design-modules/00-scalable-console/r1');
export const foundations=path.join(root,'packages/design-system');
export async function checkDesign(){
 const provenance=JSON.parse(await readFile(path.join(foundations,'provenance.json'),'utf8'));
 for(const asset of provenance.assets){
  const bytes=await readFile(path.join(foundations,asset.path));
  if(createHash('sha256').update(bytes).digest('hex')!==asset.sha256)throw Error(`Foundation differs from supplied asset: ${asset.path}`);
 }
 for(const name of await readdir(source)){
  if(!/\.(css|js|html)$/.test(name))continue;
  const text=await readFile(path.join(source,name),'utf8');
  if(name.endsWith('.css')){
   if(/#[\da-f]{3,8}\b|\brgba?\(|\bhsla?\(/i.test(text))throw Error(`Unscoped color in ${name}`);
   for(const rule of text.matchAll(/(?:border-radius|box-shadow):\s*([^;}]+)/g)){
    if(!/^var\(--hb-(?:radius|shadow)\)$/.test(rule[1].trim()))throw Error(`Noncanonical shape in ${name}`);
   }
   if(/^\s*var\([^\n]+:/m.test(text)||/var\(--[\w-]+\(/.test(text))throw Error(`Malformed CSS property/token in ${name}`);
   if(/gradient\(|backdrop-filter:|outline:\s*(?:none|0)\b/.test(text))throw Error(`Forbidden visual effect in ${name}`);
  }
  if(/brand-mark">H|[▾×]/.test(text))throw Error(`Placeholder logo/icon in ${name}`);
 }
 const app=await readFile(path.join(source,'app.js'),'utf8');
 const index=await readFile(path.join(source,'index.html'),'utf8');
 if(!app.includes('hb-lockup-color.svg')||!index.includes('hb-mark-color.svg'))throw Error('Missing canonical logo or favicon');
 if(!index.includes('design-system/fonts.css'))throw Error('Local font stylesheet is missing');
 console.log(`PASS: ${provenance.assets.length} immutable foundations, source token/brand checks`);
}
if(process.argv[1]===fileURLToPath(import.meta.url))await checkDesign();
