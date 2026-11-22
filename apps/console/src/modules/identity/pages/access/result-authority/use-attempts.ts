import {useEffect,useState,useSyncExternalStore} from "react";
import type {Session} from "../../../../../service/api";
import {AuthorityError} from "./contract";
import {sessionAuthorityAttempts,type AuthorityAttempts,type AuthorityAttempt} from "./attempts";
const idle:readonly AuthorityAttempt[]=Object.freeze([]),snapshotIdle=()=>idle,noSubscribe=()=>()=>{};
export function useAuthorityAttempts(session:Session){
 const key=JSON.stringify([session.user,session.workspace,session.csrf]);
 const [binding,setBinding]=useState<{key:string;store?:AuthorityAttempts;error?:string}>();
 useEffect(()=>{try{setBinding({key,store:sessionAuthorityAttempts(session)});}catch(e){setBinding({key,error:e instanceof AuthorityError?e.code:"session_unavailable"});}},[key]);
 const current=binding?.key===key?binding:undefined,store=current?.store;
 const entries=useSyncExternalStore(store?.subscribe??noSubscribe,store?.snapshot??snapshotIdle,snapshotIdle);
 return {store,entries,error:current?.error};
}
