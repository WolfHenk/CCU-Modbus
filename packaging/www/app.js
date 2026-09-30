(function(){
'use strict';
var sid=window.CCU_MODBUS_SID||'';
var cfg=null, states=[], dirty=false, editDevice=-1, editRegister=-1;
var $=function(id){return document.getElementById(id);};

function api(action,method,body){
  var o={method:method||'GET',cache:'no-store'};
  if(body!==undefined){o.headers={'Content-Type':'application/json'};o.body=JSON.stringify(body);}
  return fetch('api.cgi?sid='+encodeURIComponent(sid)+'&action='+action,o).then(function(r){
    return r.text().then(function(t){
      var j; try{j=JSON.parse(t);}catch(e){throw new Error('Ungueltige Antwort: '+t.slice(0,100));}
      if(!r.ok || j.ok===false) throw new Error(j.error||('HTTP '+r.status));
      return j;
    });
  });
}
function esc(s){return String(s==null?'':s).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c];});}
function setDirty(v){dirty=v;$('dirtyText').textContent=v?'Ungespeicherte Änderungen.':'Keine ungespeicherten Änderungen.';}
function notice(msg,err){var n=$('notice');n.textContent=msg;n.className='notice'+(err?' error':'');setTimeout(function(){n.className='notice hidden';},6000);}
function stateFor(id){for(var i=0;i<states.length;i++)if(states[i].id===id)return states[i];return null;}
function lamp(s){if(!s)return 'gray';if(s.state==='ONLINE')return 'green';if(s.state==='DEGRADED')return 'yellow';if(s.state==='OFFLINE'||s.state==='CONFIG_ERROR')return 'red';return 'gray';}
function stateText(s,d){if(!d.enabled)return 'Deaktiviert';if(!s)return 'Noch kein Status';if(s.state==='ONLINE')return 'Verbunden';if(s.state==='DEGRADED')return 'Verbunden, einzelne Fehler';if(s.state==='OFFLINE')return 'Gerät nicht erreichbar';if(s.state==='CONFIG_ERROR')return 'Konfigurationsfehler';return s.state;}
function age(ts){if(!ts)return '—';var n=Date.now()-new Date(ts).getTime();if(n<0)n=0;var sec=Math.round(n/1000);if(sec<60)return 'vor '+sec+' s';var min=Math.round(sec/60);if(min<60)return 'vor '+min+' min';return 'vor '+Math.round(min/60)+' h';}

function render(){
  var root=$('devices');root.innerHTML='';
  if(!cfg||!cfg.devices||!cfg.devices.length){root.innerHTML='<div class="empty">Noch keine Modbus-Geräte angelegt.</div>';return;}
  cfg.devices.forEach(function(d,idx){
    var s=stateFor(d.id), good=0, err=0;
    if(s&&s.registers){s.registers.forEach(function(r){if(r.value.quality==='GOOD')good++;if(r.value.quality==='READ_ERROR'||r.value.quality==='CONFIG_ERROR')err++;});}
    var c=document.createElement('div');c.className='card';
    c.innerHTML='<div class="cardhead"><span class="lamp '+lamp(s)+'"></span><div><h3>'+esc(d.name)+'</h3><div class="meta">'+esc(d.host)+' : '+d.port+' · Unit '+d.unit_id+'</div><div class="state">'+esc(stateText(s,d))+' · '+d.registers.length+' Werte'+(s&&s.last_seen?' · Letzte Antwort '+age(s.last_seen):'')+'</div></div></div>'+
      '<div class="actions"><button data-values="'+idx+'">Werte anzeigen</button><button data-edit="'+idx+'">Bearbeiten</button></div>'+
      '<div class="values hidden" id="values-'+idx+'"></div>';
    root.appendChild(c);
  });
  root.querySelectorAll('[data-edit]').forEach(function(b){b.onclick=function(){openDevice(Number(this.getAttribute('data-edit')));};});
  root.querySelectorAll('[data-values]').forEach(function(b){b.onclick=function(){toggleValues(Number(this.getAttribute('data-values')));};});
}
function toggleValues(idx){
  var box=$('values-'+idx), d=cfg.devices[idx], s=stateFor(d.id);box.classList.toggle('hidden');
  if(box.classList.contains('hidden'))return;
  var html='';
  d.registers.forEach(function(r){
    var rs=null;if(s&&s.registers){for(var i=0;i<s.registers.length;i++)if(s.registers[i].id===r.id){rs=s.registers[i];break;}}
    var v='—',q='Noch kein Wert';
    if(rs&&rs.value){q=rs.value.quality;if(rs.value.value!==undefined)v=rs.value.value+(r.unit?' '+esc(r.unit):'');if(rs.value.error)q=rs.value.error;}
    html+='<div class="valueRow"><span>'+esc(r.name)+'</span><strong>'+v+'</strong><span class="quality">'+esc(q)+'</span></div>';
  });
  box.innerHTML=html||'<span class="muted">Keine Register vorhanden.</span>';
}

function blankDevice(){return {id:'dev-'+Date.now(),name:'',enabled:true,host:'',port:502,unit_id:1,timeout_ms:1500,retries:1,registers:[]};}
function openDevice(idx){
  editDevice=idx;var d=idx<0?blankDevice():JSON.parse(JSON.stringify(cfg.devices[idx]));
  $('deviceDialog')._working=d;$('deviceTitle').textContent=idx<0?'Gerät hinzufügen':'Gerät bearbeiten';
  $('dName').value=d.name;$('dHost').value=d.host;$('dPort').value=d.port;$('dUnit').value=d.unit_id;$('dTimeout').value=d.timeout_ms;$('dRetries').value=d.retries;$('dEnabled').checked=d.enabled;
  $('connectionResult').textContent='';$('deleteDevice').style.visibility=idx<0?'hidden':'visible';renderRegisters();$('deviceDialog').showModal();
}
function workingDevice(){
  var d=$('deviceDialog')._working;
  d.name=$('dName').value.trim();d.host=$('dHost').value.trim();d.port=Number($('dPort').value);d.unit_id=Number($('dUnit').value);d.timeout_ms=Number($('dTimeout').value);d.retries=Number($('dRetries').value);d.enabled=$('dEnabled').checked;return d;
}
function renderRegisters(){
  var d=$('deviceDialog')._working, tb=$('registerRows');tb.innerHTML='';
  d.registers.forEach(function(r,i){var tr=document.createElement('tr');tr.innerHTML='<td>'+esc(r.name)+'</td><td>'+esc(typeName(r.type))+'</td><td>'+r.address+'</td><td>'+esc(r.datatype)+'</td><td>'+r.factor+'</td><td>'+r.poll_seconds+' s</td><td><button type="button" data-r="'+i+'">Bearbeiten</button></td>';tb.appendChild(tr);});
  tb.querySelectorAll('[data-r]').forEach(function(b){b.onclick=function(){openRegister(Number(this.getAttribute('data-r')));};});
}
function typeName(t){return {holding:'Holding',input:'Input',coil:'Coil',discrete:'Discrete'}[t]||t;}
function blankRegister(){return {id:'reg-'+Date.now(),name:'',enabled:true,type:'holding',address:0,datatype:'INT16',factor:1,offset:0,byte_swap:false,word_swap:false,poll_seconds:10,unit:''};}
function openRegister(idx){
  editRegister=idx;var d=$('deviceDialog')._working,r=idx<0?blankRegister():JSON.parse(JSON.stringify(d.registers[idx]));$('registerDialog')._working=r;$('registerTitle').textContent=idx<0?'Register hinzufügen':'Register bearbeiten';
  $('rName').value=r.name;$('rType').value=r.type;$('rAddress').value=r.address;$('rDatatype').value=String(r.datatype).toUpperCase();$('rUnit').value=r.unit||'';$('rFactor').value=r.factor;$('rOffset').value=r.offset;$('rPoll').value=r.poll_seconds;$('rByteSwap').checked=r.byte_swap;$('rWordSwap').checked=r.word_swap;$('rEnabled').checked=r.enabled;$('registerResult').textContent='';$('deleteRegister').style.visibility=idx<0?'hidden':'visible';syncDatatype();$('registerDialog').showModal();
}
function workingRegister(){
  var r=$('registerDialog')._working;r.name=$('rName').value.trim();r.type=$('rType').value;r.address=Number($('rAddress').value);r.datatype=$('rDatatype').value.toLowerCase();r.unit=$('rUnit').value.trim();r.factor=parseNumber($('rFactor').value,1);r.offset=parseNumber($('rOffset').value,0);r.poll_seconds=Number($('rPoll').value);r.byte_swap=$('rByteSwap').checked;r.word_swap=$('rWordSwap').checked;r.enabled=$('rEnabled').checked;return r;
}
function parseNumber(v,def){var n=Number(String(v).replace(',','.'));return isFinite(n)?n:def;}
function syncDatatype(){var t=$('rType').value;if(t==='coil'||t==='discrete'){$('rDatatype').value='BOOL';$('rDatatype').disabled=true;}else{$('rDatatype').disabled=false;if($('rDatatype').value==='BOOL')$('rDatatype').value='INT16';}}

$('addDevice').onclick=function(){openDevice(-1);};
$('addRegister').onclick=function(){openRegister(-1);};
$('rType').onchange=syncDatatype;
$('applyRegister').onclick=function(){var r=workingRegister();if(!r.name){notice('Bitte eine Bezeichnung eingeben.',true);return;}var d=$('deviceDialog')._working;if(editRegister<0)d.registers.push(r);else d.registers[editRegister]=r;$('registerDialog').close();renderRegisters();};
$('deleteRegister').onclick=function(){if(editRegister>=0&&confirm('Dieses Register wirklich löschen?')){$('deviceDialog')._working.registers.splice(editRegister,1);$('registerDialog').close();renderRegisters();}};
$('applyDevice').onclick=function(){var d=workingDevice();if(!d.name||!d.host){notice('Name und IP-Adresse/Host sind Pflichtfelder.',true);return;}if(editDevice<0)cfg.devices.push(d);else cfg.devices[editDevice]=d;$('deviceDialog').close();setDirty(true);render();};
$('deleteDevice').onclick=function(){if(editDevice>=0&&confirm('Gerät und alle zugehörigen Register wirklich löschen?')){cfg.devices.splice(editDevice,1);$('deviceDialog').close();setDirty(true);render();}};
$('testConnection').onclick=function(){var out=$('connectionResult'),d=workingDevice();out.className='';out.textContent='Prüfe…';api('test_connection','POST',d).then(function(){out.className='ok';out.textContent='✓ TCP-Verbindung erfolgreich';}).catch(function(e){out.className='bad';out.textContent='✗ '+e.message;});};
$('testRegister').onclick=function(){var out=$('registerResult'),d=workingDevice(),r=workingRegister();out.className='';out.textContent='Lese…';api('test_register','POST',{device:d,register:r}).then(function(x){out.className='ok';out.textContent='✓ Rohwert: '+x.raw+' · Wert: '+x.value+(x.unit?' '+x.unit:'');}).catch(function(e){out.className='bad';out.textContent='✗ '+e.message;});};
$('saveAll').onclick=function(){if(!dirty)return;this.disabled=true;var btn=this;api('save','POST',cfg).then(function(){setDirty(false);notice('Konfiguration gespeichert. Modbus-Dienst wird neu gestartet.');setTimeout(loadStatus,1800);}).catch(function(e){notice('Speichern fehlgeschlagen: '+e.message,true);}).finally(function(){btn.disabled=false;});};

function loadStatus(){api('status').then(function(x){states=Array.isArray(x)?x:[];render();}).catch(function(){states=[];render();});}
function init(){Promise.all([api('config'),api('status').catch(function(){return [];})]).then(function(x){cfg=x[0];states=Array.isArray(x[1])?x[1]:[];setDirty(false);render();setInterval(loadStatus,5000);}).catch(function(e){notice('CCU-Modbus konnte nicht geladen werden: '+e.message,true);});}
init();
})();