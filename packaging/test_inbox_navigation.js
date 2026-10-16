const fs=require('fs'),vm=require('vm'),assert=require('assert');
const src=fs.readFileSync(require('path').join(__dirname,'www/app.js'),'utf8');
const start=src.indexOf('function openInboxIn(');
const end=src.indexOf('function stateFor(',start);
assert(start>0&&end>start);
let calls=[],tab=null,timeoutCalls=[];
function frame(ok){return {closed:false,WebUI:{enter:(x)=>calls.push(['enter',x])},NewDeviceListPage:'NativeInbox',ConfigData:{check:(f)=>{calls.push(['check']);f();}}};}
const ctx={window:{opener:null,parent:null,top:null},notice:(m)=>calls.push(['notice',m]),setInterval:(f,ms)=>{timeoutCalls.push(f);return 1;},clearInterval:()=>{},setTimeout:(f)=>f(),api:()=>Promise.resolve([]),Array,console};
vm.createContext(ctx);vm.runInContext(src.slice(start,end),ctx);
ctx.window.opener=frame();assert.strictEqual(ctx.showNativeInbox(null),true);assert.deepStrictEqual(calls,[['check'],['enter','NativeInbox']]);console.log('PASS opener window -> native inbox');
calls=[];ctx.window.opener=null;ctx.window.parent=frame();assert(ctx.showNativeInbox(null));assert.deepStrictEqual(calls,[['check'],['enter','NativeInbox']]);console.log('PASS parent frame -> native inbox');
calls=[];ctx.window.parent=ctx.window;ctx.window.top=frame();assert(ctx.showNativeInbox(null));assert.deepStrictEqual(calls,[['check'],['enter','NativeInbox']]);console.log('PASS top frame -> native inbox');
calls=[];ctx.window.top=ctx.window;tab=frame();assert(ctx.showNativeInbox(tab));assert(timeoutCalls.length>0);timeoutCalls.pop()();assert.deepStrictEqual(calls,[['check'],['enter','NativeInbox']]);console.log('PASS new tab -> native inbox');
calls=[];ctx.window.opener=null;ctx.window.parent=ctx.window;ctx.window.top=ctx.window;assert.strictEqual(ctx.showNativeInbox(null),false);assert(calls.some(x=>x[0]==='notice'));console.log('PASS blocked popup -> explicit failure message');
console.log('NOTE integration with actual CCU browser and save form not covered');

async function testSave(label,devices,known,fail,expectedOpens,expectedWaits){
  let opened=0,waited=[],notices=[],changes=[],tabObject={closed:false,close:function(){this.closed=true;}};
  const elements={saveAll:{disabled:false}};
  const w={opener:null};w.parent=w;w.top=w;
  w.open=()=>{opened++;return tabObject;};
  const ctx2={
    cfg:{devices},savedDeviceIds:Object.assign({},known),dirty:true,
    window:w,$:id=>elements[id],api:()=>fail?Promise.reject(new Error('failed')):Promise.resolve({ok:true}),
    setDirty:x=>changes.push(x),notice:s=>notices.push(s),
    setTimeout:()=>{},loadStatus:()=>{},waitAndShowInbox:ids=>waited.push(...ids)
  };
  vm.createContext(ctx2);
  const start2=src.indexOf("$('saveAll').onclick=");
  const end2=src.indexOf("var updateAvailable",start2);
  assert(start2>0&&end2>start2);
  vm.runInContext(src.slice(start2,end2),ctx2);
  ctx2.$('saveAll').onclick.call(ctx2.$('saveAll'));
  await new Promise(resolve=>setTimeout(resolve,0));
  assert.strictEqual(opened,expectedOpens,label+' popup');
  assert.strictEqual(waited.length,expectedWaits,label+' wait count');
  assert.strictEqual(ctx2.$('saveAll').disabled,false,label+' button reset');
  assert.strictEqual(tabObject.closed,!!fail && expectedOpens>0,label+' popup close on error');
  if(fail){assert(notices.some(x=>x.includes('fehlgeschlagen')));assert(!ctx2.savedDeviceIds.new);}
  else assert(changes.includes(false));
  console.log('PASS '+label);
}
(async()=>{
  const existing={id:'existing',enabled:true},fresh={id:'new',enabled:true},disabled={id:'new',enabled:false};
  await testSave('new active device opens inbox', [existing,fresh],{existing:true},false,1,1);
  await testSave('editing existing device does not open inbox',[existing],{existing:true},false,0,0);
  await testSave('new disabled device does not open inbox',[existing,disabled],{existing:true},false,0,0);
  await testSave('failed save does not open inbox',[existing,fresh],{existing:true},true,1,0);
})().catch(e=>{console.error(e);process.exitCode=1;});
