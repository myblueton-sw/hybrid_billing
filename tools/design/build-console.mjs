import {readFile,writeFile,mkdir,readdir,rm} from 'node:fs/promises';
import path from 'node:path';
import {createHash} from 'node:crypto';
import {buildRevisions} from './build-revisions.mjs';
import {checkDesign,root,source,foundations} from './check-design.mjs';
await checkDesign();
const output=path.join(root,'output/designs/HB-63-console');
async function inventory(directory,prefix=''){
 const entries=[];
 for(const item of await readdir(directory,{withFileTypes:true})){
  const relative=prefix+item.name;
  if(item.isSymbolicLink())throw Error(`Unexpected symlink in build output: ${relative}`);
  if(item.isDirectory())entries.push(...await inventory(path.join(directory,item.name),relative+'/'));
  else entries.push(relative);
 }
 return entries;
}
let previous=null;
try{previous=JSON.parse(await readFile(path.join(output,'build-manifest.json'),'utf8'));}
catch(error){if(error.code!=='ENOENT')throw error;}
if(previous){
 if(previous.source!=='docs/design-modules/00-scalable-console/r1'||!previous.files)throw Error('Output ownership manifest mismatch');
 const unexpected=(await inventory(output)).filter(file=>file!=='build-manifest.json'&&!Object.hasOwn(previous.files,file));
 if(unexpected.length)throw Error(`Unexpected output files; inspect before rebuilding: ${unexpected.join(', ')}`);
 await rm(output,{recursive:true});
}else{
 try{if((await inventory(output)).length)throw Error('Nonempty output without ownership manifest');}
 catch(error){if(error.code!=='ENOENT')throw error;}
}
const files=new Map();
async function emit(relative,bytes){
 const destination=path.join(output,relative);
 await mkdir(path.dirname(destination),{recursive:true});
 await writeFile(destination,bytes);
 files.set(relative,createHash('sha256').update(bytes).digest('hex'));
}
async function copyTree(directory,prefix){
 for(const item of await readdir(directory,{withFileTypes:true})){
  if(item.isDirectory())await copyTree(path.join(directory,item.name),`${prefix}/${item.name}`);
  else await emit(`${prefix}/${item.name}`,await readFile(path.join(directory,item.name)));
 }
}
for(const name of (await readdir(source)).filter(name=>/\.(js|css|html)$/.test(name)).sort())await emit(name,await readFile(path.join(source,name)));
await copyTree(path.join(foundations,'assets'),'design-system/assets');
await emit('design-system/tokens.css',await readFile(path.join(foundations,'styles/tokens.css')));
await emit('design-system/console.css',await readFile(path.join(foundations,'styles/console.css')));
let fontCss='';
const fonts=[];
for(const family of ['ibm-plex-sans','ibm-plex-sans-arabic','ibm-plex-sans-kr','ibm-plex-mono']){
 const directory=path.join(root,'tools/design/node_modules/@fontsource',family);
 const metadata=JSON.parse(await readFile(path.join(directory,'package.json'),'utf8'));
 const weights=family==='ibm-plex-mono'?[400,500]:[400,500,600,700];
 for(const weight of weights){
  const css=await readFile(path.join(directory,`${weight}.css`),'utf8');
  for(const match of css.matchAll(/url\((['"]?)(\.\/files\/[^)'"\s]+)\1\)/g)){
   const basename=path.basename(match[2]);
   await emit(`design-system/fonts/${basename}`,await readFile(path.join(directory,match[2])));
  }
  fontCss+=css.replace(/\.\/files\//g,'./fonts/')+'\n';
 }
 const license=await readFile(path.join(directory,'LICENSE'));
 await emit(`design-system/licenses/${family}-LICENSE.txt`,license);
 fonts.push({package:metadata.name,version:metadata.version,license:metadata.license});
}
await emit('design-system/fonts.css',fontCss);
await emit('NOTICE.txt',await readFile(path.join(root,'THIRD_PARTY_NOTICES.md')));
await emit('design-revisions.json',JSON.stringify(await buildRevisions(source,files),null,2)+'\n');
const manifest={source:'docs/design-modules/00-scalable-console/r1',foundations:'packages/design-system',fonts,files:Object.fromEntries([...files].sort(([a],[b])=>a.localeCompare(b)))};
await writeFile(path.join(output,'build-manifest.json'),JSON.stringify(manifest,null,2)+'\n');
console.log(`Built ${files.size} files at output/designs/HB-63-console. Serve this directory; no runtime network dependency.`);
