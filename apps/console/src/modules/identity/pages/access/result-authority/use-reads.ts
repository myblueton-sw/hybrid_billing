import {useEffect,useState,useSyncExternalStore} from "react";
import type {Session} from "../../../../../service/api";
import {AuthorityReads,type AuthorityReadsState} from "./read-store";
import {resultDomain,type AuthorityDomain} from "./contract";
import {readTransport} from "./transport";
const empty:AuthorityReadsState=Object.freeze({history:{phase:"idle"},detail:{phase:"idle"},context:{phase:"idle"},selected:""} as const);
const idle=()=>empty,noSubscribe=()=>()=>{};
export function useAuthorityReads(session:Session,d:AuthorityDomain=resultDomain){
 const key=JSON.stringify([session.user,session.workspace,session.csrf,d.namespace]);
 const [binding,setBinding]=useState<{key:string;store:AuthorityReads}>();
 useEffect(()=>{
  const store=new AuthorityReads(readTransport(session,undefined,d),session.workspace,session.user,d);setBinding({key,store});void store.history();
  return()=>store.dispose();
 },[key]);
 const store=binding?.key===key?binding.store:undefined;
 const state=useSyncExternalStore(store?.subscribe??noSubscribe,store?.snapshot??idle,idle);
 return {store,state};
}
