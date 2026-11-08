import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {existsSync} from 'node:fs';
import {createRequire} from 'node:module';

const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'../..');
const source=path.join(root,'docs','design-modules','00-scalable-console','r1');
const suites={generic:'verify.cjs',journeys:'verify-journeys.cjs',review:'verify-review.cjs'};
const options={suite:'all',module:process.env.HB_PLAYWRIGHT_MODULE||'playwright',chromium:process.env.HB_CHROMIUM,url:process.env.HB_PREVIEW_URL};
const keys={'--suite':'suite','--playwright-module':'module','--chromium':'chromium','--url':'url'};

async function main(){
 const args=process.argv.slice(2);
 if(args.includes('--help')){
  console.log('Usage: npm run qa --prefix tools/design -- [--suite all|generic|journeys|review] [--playwright-module PATH] [--chromium PATH] [--url http://127.0.0.1:8763]\nInstall Playwright separately for browser QA. No browser or OS prerequisite is needed to use the static preview beyond a compatible browser and HTTP server.');
  return;
 }
 for(let index=0;index<args.length;index++){
  if(!Object.hasOwn(keys,args[index])||index+1>=args.length||args[index+1].startsWith('--'))throw new Error(`Unknown or incomplete option: ${args[index]}`);
  options[keys[args[index]]]=args[++index];
 }
 const names=options.suite==='all'?Object.keys(suites):[options.suite];
 if(names.some(name=>!Object.hasOwn(suites,name)))throw new Error('Unknown suite. Choose all, generic, journeys or review.');
 for(const name of names)if(!existsSync(path.join(source,suites[name])))throw new Error(`Suite is not available in this source revision: ${name} (${suites[name]}).`);
 const require=createRequire(import.meta.url);
 let module;
 try{module=require.resolve(options.module);}
 catch{throw new Error('Playwright was not found. Provide --playwright-module with its installed package directory or install it in the tool environment.');}
 if(options.chromium&&!existsSync(options.chromium))throw new Error('The --chromium executable path does not exist.');
 if(options.url){const url=new URL(options.url);if(!['http:','https:'].includes(url.protocol))throw new Error('Preview URL must use HTTP or HTTPS.');}
 const {spawn}=await import('node:child_process');
 for(const name of names){
  console.log(`Running ${name} browser QA`);
  const env={...process.env,HB_PLAYWRIGHT_MODULE:module};
  if(options.chromium)env.HB_CHROMIUM=options.chromium;
  if(options.url)env.HB_PREVIEW_URL=options.url;
  const exitCode=await new Promise((resolve,reject)=>{
   const child=spawn(process.execPath,[path.join(source,suites[name])],{cwd:root,env,stdio:'inherit',shell:false});
   child.on('error',reject);child.on('exit',(code,signal)=>resolve(signal?1:code??1));
  });
  if(exitCode!==0){process.exitCode=exitCode;return;}
 }
}

try{await main();}
catch(error){console.error(`Browser QA could not start: ${error.message}`);process.exitCode=1;}
