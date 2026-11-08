import test from 'node:test';
import assert from 'node:assert/strict';
import {readReviews,saveReview,reviewStatus} from '../../docs/design-modules/00-scalable-console/r1/review-store.js';
function memory(){let value=null;return {getItem:()=>value,setItem:(_,next)=>{value=next;}};}
const fixture={screen:'assistant',revision:'a'.repeat(64),locale:'ko',layout:'wide',width:1440,state:'b'.repeat(64),items:['assistant:field:request'],confirmedAt:'2026-11-01T00:00:00.000Z',source:'local-unauthenticated-confirmation'};
test('local confirmation persists and does not accept another locale or layout',()=>{
 const store=memory();assert.equal(saveReview(store,fixture).ok,true);
 const {records}=readReviews(store);assert.equal(records.length,1);
 assert.equal(reviewStatus(records,'assistant',fixture.revision,'ko','wide',{width:1440,state:fixture.state,items:fixture.items}),'localFix');
 assert.equal(reviewStatus(records,'assistant',fixture.revision,'en','wide'),'otherVariant');
 assert.equal(reviewStatus(records,'assistant',fixture.revision,'ko','compact'),'otherVariant');
});
test('new revisions preserve earlier scope evidence',()=>{
 const store=memory();saveReview(store,fixture);
 assert.equal(reviewStatus(readReviews(store).records,'assistant','c'.repeat(64),'ko','wide'),'changed');
 saveReview(store,{...fixture,revision:'c'.repeat(64)});
 assert.equal(readReviews(store).records.length,2);
 saveReview(store,fixture);assert.equal(readReviews(store).records.length,2);
});
test('corrupt records block writes rather than clearing evidence',()=>{
 const store=memory();store.setItem('', '{invalid');assert.equal(readReviews(store).error,'storage');
 assert.equal(saveReview(store,fixture).ok,false);assert.equal(store.getItem(),'{invalid');
});
test('failed write preserves previously accepted data and reports failure',()=>{
 const store=memory();saveReview(store,fixture);const saved=store.getItem();
 store.setItem=()=>{throw new Error('quota');};
 assert.equal(saveReview(store,{...fixture,locale:'en'}).ok,false);assert.equal(store.getItem(),saved);
});
test('unavailable storage and invalid foreign item IDs are rejected',()=>{
 assert.equal(readReviews({getItem:()=>{throw new Error('disabled');}}).error,'storage');
 assert.equal(saveReview(memory(),{...fixture,items:['another-screen:action:approve']}).ok,false);
 assert.equal(saveReview(memory(),{...fixture,source:'owner-authenticated'}).ok,false);
});

test('exact width and current item/state scopes never inherit acceptance',()=>{
 const store=memory();saveReview(store,fixture);saveReview(store,{...fixture,width:1200});
 const {records}=readReviews(store);assert.equal(records.length,2);
 assert.equal(reviewStatus(records,'assistant',fixture.revision,'ko','wide',{width:1300,state:fixture.state,items:fixture.items}),'otherState');
 assert.equal(reviewStatus(records,'assistant',fixture.revision,'ko','wide',{width:1440,state:'c'.repeat(64),items:fixture.items}),'otherState');
 assert.equal(reviewStatus(records,'assistant',fixture.revision,'ko','wide',{width:1440,state:fixture.state,items:['assistant:field:another']}),'otherState');
});
