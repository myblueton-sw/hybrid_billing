import test from "node:test";
import assert from "node:assert/strict";
import { emptyDraft, packetOf, PacketError } from "./packet";
import type { Context } from "./contract";
const hash="a".repeat(64);
const context: Context={preparation:{preparation:"prep",preparation_digest:hash,document:"doc",document_digest:hash,connection:"erp",customer:"customer",period:"2026-10",currency:"USD",preparation_actor:"actor"},revision:"r1",permission:{state:"allowed",code:""},readiness:{state:"ready",code:""},references:[{line:"line",expected_product:"expected",expected_unit:"hour"}]};
test("result packet begins blank and preserves independently reported mismatches",()=>{
 const blank=emptyDraft(context);assert.deepEqual(Object.values(blank.rows.line),["","","","","",""]);
 const draft={...blank,evidence:"evidence",external:"external",revision:"2",rationale:"Independent report",rows:{line:{row:"row",product:"different",unit:"day",quantity:"2",net:"3.25",tax:"0"}}};
 const built=packetOf(context,draft);assert.equal(built.packet.Rows[0].Product,"different");assert.equal(built.packet.Rows[0].Net,"3.25");assert.equal(built.packet.PreparationDigest,hash);
 for(const net of ["3.250","1e3","-1","01","0.00001"]){assert.throws(()=>packetOf(context,{...draft,rows:{line:{...draft.rows.line,net}}}),PacketError);}
 assert.throws(()=>packetOf(context,{...draft,revision:"2147483648"}),PacketError);
 assert.throws(()=>packetOf({...context,readiness:{state:"blocked",code:"prerequisite_failed"}},draft));
});
