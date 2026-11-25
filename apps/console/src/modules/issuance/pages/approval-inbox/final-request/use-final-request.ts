import { useEffect,useState,useSyncExternalStore } from "react";
import { useService } from "../../../../../service/context";
import { ResultError } from "../results/contract";
import { sessionFinalRequestStore,type FinalRequestStore,type State } from "./store";
const idle:State=Object.freeze({target:"",rationale:"",checking:false,entries:Object.freeze([])});
const subscribeIdle=()=>()=>{},snapshotIdle=()=>idle;
export function useFinalRequest(){
 const service=useService(),session=service?.session;
 const key=session?JSON.stringify([session.user,session.workspace,session.csrf]):"";
 const [binding,setBinding]=useState<{key:string;store?:FinalRequestStore;error?:string}>();
 useEffect(()=>{
  if(!session){setBinding(undefined);return;}
  let store:FinalRequestStore|undefined;
  try{store=sessionFinalRequestStore(session);store.invalidate();setBinding({key,store});}
  catch(error){setBinding({key,error:error instanceof ResultError?error.code:"session_unavailable"});}
  return()=>store?.invalidate();
 },[key]);
 const current=binding?.key===key?binding:undefined,store=current?.store;
 const state=useSyncExternalStore(store?.subscribe??subscribeIdle,store?.snapshot??snapshotIdle,snapshotIdle);
 const unresolved=state.entries.some(e=>e.phase==="pending"||e.phase==="unknown");
 const reauthenticate=service?.stepUp?async()=>{store?.invalidate();await service.stepUp!();}:undefined;
 return {store,state,unresolved,reauthenticate,error:current?.error??(!session?"session_unavailable":undefined)};
}
export type Binding=ReturnType<typeof useFinalRequest>;
