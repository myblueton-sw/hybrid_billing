import {useEffect,useState,useSyncExternalStore} from "react";
import type {Session} from "../../../../../service/api";
import {AuthorityReads,type AuthorityReadsState} from "./read-store";
import {readTransport} from "./transport";
const empty:AuthorityReadsState=Object.freeze({history:{phase:"idle"},detail:{phase:"idle"},context:{phase:"idle"},selected:""} as const);
const idle=()=>empty,noSubscribe=()=>()=>{};
export function useAuthorityReads(session:Session){
 const key=JSON.stringify([session.user,session.workspace,session.csrf]);
 const [binding,setBinding]=useState<{key:string;store:AuthorityReads}>();
 useEffect(()=>{
  const store=new AuthorityReads(readTransport(session),session.workspace,session.user);setBinding({key,store});void store.history();
  return()=>store.dispose();
 },[key]);
 const store=binding?.key===key?binding.store:undefined;
 const state=useSyncExternalStore(store?.subscribe??noSubscribe,store?.snapshot??idle,idle);
 return {store,state};
}
