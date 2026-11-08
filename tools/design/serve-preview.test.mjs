import test from 'node:test';
import assert from 'node:assert/strict';
import http from 'node:http';
import path from 'node:path';
import os from 'node:os';
import {mkdtemp,mkdir,writeFile,rm,symlink} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import {createPreviewServer,startPreviewServer,parseOptions} from './serve-preview.mjs';

const digest=bytes=>createHash('sha256').update(bytes).digest('hex');
async function fixture(t){
 const temporary=await mkdtemp(path.join(os.tmpdir(),'hb-preview-'));
 const directory=path.join(temporary,'build');
 await mkdir(path.join(directory,'fonts'),{recursive:true});
 const assets={'index.html':'<h1>Design preview</h1>','app.js':'export const ready=true;','data.json':'{"synthetic":true}','fonts/example.woff2':'font-bytes','LICENSE.txt':'Example notice'};
 for(const [name,bytes] of Object.entries(assets))await writeFile(path.join(directory,name),bytes);
 const manifest={source:'docs/design-modules/00-scalable-console/r1',files:Object.fromEntries(Object.entries(assets).map(([name,bytes])=>[name,digest(bytes)]))};
 await writeFile(path.join(directory,'build-manifest.json'),JSON.stringify(manifest));
 await writeFile(path.join(directory,'unowned.txt'),'Must not be exposed');
 const server=await startPreviewServer({directory,port:0});
 t.after(async()=>{server.closeAllConnections();await new Promise(resolve=>server.close(resolve));await rm(temporary,{recursive:true,force:true});});
 return {temporary,directory,server,manifest};
}
function request(server,target,method='GET'){
 return new Promise((resolve,reject)=>{
  const req=http.request({host:'127.0.0.1',port:server.address().port,path:target,method},response=>{
   const chunks=[];response.on('data',chunk=>chunks.push(chunk));
   response.on('end',()=>resolve({status:response.statusCode,headers:response.headers,body:Buffer.concat(chunks).toString()}));
  });
  req.on('error',reject);req.end();
 });
}

test('serves manifest-owned assets and HEAD with browser MIME types',async t=>{
 const {server}=await fixture(t);
 assert.equal(server.address().address,'127.0.0.1');
 const home=await request(server,'/?lang=ko');
 assert.equal(home.status,200);assert.equal(home.body,'<h1>Design preview</h1>');
 assert.equal(home.headers['content-type'],'text/html; charset=utf-8');
 assert.equal(home.headers['cache-control'],'no-store');
 for(const [name,type] of [['app.js','text/javascript'],['data.json','application/json'],['fonts/example.woff2','font/woff2'],['LICENSE.txt','text/plain'],['build-manifest.json','application/json']]){
  const response=await request(server,'/'+name);assert.equal(response.status,200);assert.ok(response.headers['content-type'].startsWith(type));
 }
 const head=await request(server,'/app.js','HEAD');assert.equal(head.status,200);assert.equal(head.body,'');assert.equal(Number(head.headers['content-length']),Buffer.byteLength('export const ready=true;'));
});

test('rejects non-read methods, unknown files and directory listing',async t=>{
 const {server}=await fixture(t);
 for(const target of ['/unowned.txt','/missing.js','/fonts'])assert.equal((await request(server,target)).status,404);
 assert.equal((await request(server,'/fonts/')).status,400);
 for(const method of ['POST','PUT','DELETE','OPTIONS']){
  const response=await request(server,'/',method);assert.equal(response.status,405);assert.equal(response.headers.allow,'GET, HEAD');
 }
});

test('rejects raw, encoded and Windows-style traversal without URL normalization',async t=>{
 const {server}=await fixture(t);
 for(const target of ['/../secret','/%2e%2e/secret','/fonts/%2E%2E/app.js','/%2e/app.js','/..%5csecret','/fonts%5c..%5csecret','/C:%5csecret','//server/share','/.env','/%00','/%ZZ']){
  assert.equal((await request(server,target)).status,400,target);
 }
});

test('rejects manifest-owned symlinks and nested symlink directories',async t=>{
 const {temporary,directory,server,manifest}=await fixture(t);
 const outside=path.join(temporary,'outside');await mkdir(outside);await writeFile(path.join(outside,'secret.txt'),'outside');
 try{
  await symlink(path.join(outside,'secret.txt'),path.join(directory,'leak.txt'),'file');
  await symlink(outside,path.join(directory,'linked'),'junction');
 }catch(error){if(error.code==='EPERM'){t.skip('Host does not permit test symlink creation');return;}throw error;}
 manifest.files['leak.txt']=digest('outside');manifest.files['linked/secret.txt']=digest('outside');
 await writeFile(path.join(directory,'build-manifest.json'),JSON.stringify(manifest));
 assert.equal((await request(server,'/leak.txt')).status,404);assert.equal((await request(server,'/linked/secret.txt')).status,404);
});

test('reloads build ownership and fails closed while manifest is invalid',async t=>{
 const {directory,server,manifest}=await fixture(t);
 await writeFile(path.join(directory,'new.js'),'export default 1;');
 assert.equal((await request(server,'/new.js')).status,404);
 manifest.files['new.js']=digest('export default 1;');await writeFile(path.join(directory,'build-manifest.json'),JSON.stringify(manifest));
 assert.equal((await request(server,'/new.js')).status,200);
 await writeFile(path.join(directory,'build-manifest.json'),'invalid');assert.equal((await request(server,'/')).status,503);
 manifest.files['../private.txt']=digest('outside');await writeFile(path.join(directory,'build-manifest.json'),JSON.stringify(manifest));
 assert.equal((await request(server,'/')).status,503);
});

test('missing build, invalid port and nonloopback binding fail explicitly',async()=>{
 await assert.rejects(()=>createPreviewServer({directory:path.join(os.tmpdir(),'hb-no-build-'+Date.now())}),/Build|build/);
 await assert.rejects(()=>startPreviewServer({host:'0.0.0.0'}),/loopback/);
 await assert.rejects(()=>startPreviewServer({port:65536}),/port/);
 assert.throws(()=>parseOptions(['--port','not-a-port'],{}),/port/);
 assert.throws(()=>parseOptions(['--directory'],{}),/incomplete/);
 assert.deepEqual(parseOptions(['--port','0','--host','::1','--directory','relative build'],{}),{port:0,host:'::1',directory:'relative build'});
 assert.equal(parseOptions([],{HB_PREVIEW_PORT:'9000'}).port,9000);
});
