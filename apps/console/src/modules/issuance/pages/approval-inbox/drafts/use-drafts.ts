import { useEffect,useState,useSyncExternalStore } from "react";
import { useService } from "../../../../../service/context";
import { DraftError,freeze } from "./contract";
import { sessionDraftStore,type DraftStore,type DraftState } from "./store";
const idle=freeze<DraftState>({selected:"",list:{loading:false},inspection:{loading:false},context:{loading:false},entries:[]});
const subscribeIdle=()=>()=>{},snapshotIdle=()=>idle;
export function useDrafts(){
 const service=useService(),session=service?.session;
 const key=session?JSON.stringify([session.user,session.workspace,session.csrf]):"";
 const [binding,setBinding]=useState<{key:string;store?:DraftStore;error?:string}>();
 useEffect(()=>{
  if(!session){setBinding(undefined);return;}
  let store:DraftStore|undefined;
  try{store=sessionDraftStore(session);store.invalidate();setBinding({key,store});}
  catch(error){setBinding({key,error:error instanceof DraftError?error.code:"session_unavailable"});}
  return()=>store?.invalidate();
 },[key]);
 const current=binding?.key===key?binding:undefined,store=current?.store;
 const state=useSyncExternalStore(store?.subscribe??subscribeIdle,store?.snapshot??snapshotIdle,snapshotIdle);
 const unresolved=state.entries.some(e=>e.phase==="pending"||e.phase==="unknown");
 const reauthenticate=service?.stepUp?async()=>{store?.invalidate();await service.stepUp!();}:undefined;
 return {store,state,session,unresolved,reauthenticate,error:current?.error??(!session?"session_unavailable":undefined)};
}
export type DraftBinding=ReturnType<typeof useDrafts>;
