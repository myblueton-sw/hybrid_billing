import http from 'node:http';
import path from 'node:path';
import {readFile,lstat,realpath} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';

const repository=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'../..');
export const defaultDirectory=path.join(repository,'output','designs','HB-63-console');
const source='docs/design-modules/00-scalable-console/r1';
const hosts=new Set(['127.0.0.1','::1','localhost']);
const types=new Map([
 ['.html','text/html; charset=utf-8'],['.js','text/javascript; charset=utf-8'],
 ['.mjs','text/javascript; charset=utf-8'],['.css','text/css; charset=utf-8'],
 ['.json','application/json; charset=utf-8'],['.svg','image/svg+xml'],
 ['.woff','font/woff'],['.woff2','font/woff2'],['.ttf','font/ttf'],
 ['.png','image/png'],['.ico','image/x-icon'],['.txt','text/plain; charset=utf-8']
]);

function validRelative(value){
 return typeof value==='string'&&value.length>0&&!/[\\:\x00-\x1f\x7f]/.test(value)&&
  !value.startsWith('/')&&value.split('/').every(segment=>segment&&segment!=='.'&&segment!=='..'&&!segment.startsWith('.'));
}

async function regularFile(root,relative){
 let candidate=root;
 const parts=relative.split('/');
 for(let index=0;index<parts.length;index++){
  candidate=path.join(candidate,parts[index]);
  const stat=await lstat(candidate);
  if(stat.isSymbolicLink()||(index<parts.length-1?!stat.isDirectory():!stat.isFile()))throw new Error('Unsafe preview path');
 }
 const resolved=await realpath(candidate);
 const relativeResolved=path.relative(root,resolved);
 if(relativeResolved.startsWith(`..${path.sep}`)||relativeResolved==='..'||path.isAbsolute(relativeResolved))throw new Error('Unsafe preview path');
 return resolved;
}

async function manifestFiles(root){
 const manifest=JSON.parse(await readFile(await regularFile(root,'build-manifest.json'),'utf8'));
 if(manifest.source!==source||!manifest.files||Array.isArray(manifest.files)||typeof manifest.files!=='object')throw new Error('Invalid preview manifest');
 const files=Object.entries(manifest.files);
 if(!Object.hasOwn(manifest.files,'index.html')||files.some(([name,digest])=>!validRelative(name)||typeof digest!=='string'||!/^[a-f0-9]{64}$/.test(digest)))throw new Error('Invalid preview manifest');
 return new Set([...files.map(([name])=>name),'build-manifest.json']);
}

function respond(response,status,message){
 response.writeHead(status,{'Content-Type':'text/plain; charset=utf-8','Content-Length':Buffer.byteLength(message)});
 response.end(message);
}

export async function createPreviewServer({directory=defaultDirectory}={}){
 let root;
 try{
  const supplied=path.resolve(directory);
  const stat=await lstat(supplied);
  if(!stat.isDirectory()||stat.isSymbolicLink())throw new Error('Unsafe preview directory');
  root=await realpath(supplied);
  await manifestFiles(root);
 }catch{
  throw new Error('Preview build missing or invalid. Run npm run build --prefix tools/design before starting the preview.');
 }
 return http.createServer(async(request,response)=>{
  response.setHeader('X-Content-Type-Options','nosniff');
  response.setHeader('Cache-Control','no-store');
  response.setHeader('Referrer-Policy','no-referrer');
  if(!['GET','HEAD'].includes(request.method)){
   response.setHeader('Allow','GET, HEAD');respond(response,405,'Method not allowed');return;
  }
  let relative;
  try{
   const raw=request.url.split('?')[0];
   if(!raw.startsWith('/')||raw.startsWith('//'))throw new Error('Invalid request target');
   const decoded=decodeURIComponent(raw.slice(1));
   relative=decoded||'index.html';
   if(!validRelative(relative))throw new Error('Invalid path');
  }catch{respond(response,400,'Invalid preview path');return;}
  let files;
  try{files=await manifestFiles(root);}
  catch{respond(response,503,'Preview build unavailable. Rebuild the preview and retry.');return;}
  if(!files.has(relative)){respond(response,404,'Preview asset not found');return;}
  try{
   const location=await regularFile(root,relative);
   const bytes=await readFile(location);
   response.writeHead(200,{'Content-Type':types.get(path.extname(relative).toLowerCase())||'application/octet-stream','Content-Length':bytes.length});
   response.end(request.method==='HEAD'?undefined:bytes);
  }catch{respond(response,404,'Preview asset not found');}
 });
}

export async function startPreviewServer({directory=defaultDirectory,host='127.0.0.1',port=8763}={}){
 if(!hosts.has(host))throw new Error('Preview host must be loopback: 127.0.0.1, ::1 or localhost.');
 if(!Number.isInteger(port)||port<0||port>65535)throw new Error('Preview port must be an integer from 0 to 65535.');
 const server=await createPreviewServer({directory});
 await new Promise((resolve,reject)=>{
  const onError=error=>reject(error);
  server.once('error',onError);
  server.listen(port,host,()=>{server.removeListener('error',onError);resolve();});
 });
 return server;
}

export function parseOptions(args,env=process.env){
 const options={directory:env.HB_PREVIEW_DIRECTORY||defaultDirectory,host:env.HB_PREVIEW_HOST||'127.0.0.1',port:env.HB_PREVIEW_PORT||'8763'};
 for(let index=0;index<args.length;index++){
  const argument=args[index];
  if(argument==='--help')return {help:true};
  const keys={'--directory':'directory','--host':'host','--port':'port'};
  if(!Object.hasOwn(keys,argument)||index+1>=args.length||args[index+1].startsWith('--'))throw new Error(`Unknown or incomplete option: ${argument}`);
  options[keys[argument]]=args[++index];
 }
 if(!/^\d+$/.test(String(options.port)))throw new Error('Preview port must be an integer from 0 to 65535.');
 options.port=Number(options.port);
 return options;
}

async function main(){
 try{
  const options=parseOptions(process.argv.slice(2));
  if(options.help){console.log('Usage: npm start --prefix tools/design -- [--port 8763] [--host 127.0.0.1] [--directory PATH]\nLocal static design preview only. Build first; stop with Ctrl+C.');return;}
  const server=await startPreviewServer(options);
  const address=server.address();
  const host=address.address.includes(':')?`[${address.address}]`:address.address;
  console.log(`Preview ready: http://${host}:${address.port}/?lang=ko#contents`);
  for(const signal of ['SIGINT','SIGTERM'])process.once(signal,()=>server.close(()=>{process.exitCode=0;}));
 }catch(error){console.error(`Preview could not start: ${error.message}`);process.exitCode=1;}
}

if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url))await main();
