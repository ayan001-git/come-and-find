const fs=require('fs'),vm=require('vm'),assert=require('assert');
const src=fs.readFileSync(require('path').join(__dirname,'../web/voice.js'),'utf8');
function scenario({support=true,secure=true,failStart=false}={}){
 const listeners={};
 const elems={};
 for(const id of ['voiceBtn','voiceLanguage','voiceStatus','searchInput','searchBtn'])elems[id]={textContent:'',value:'',disabled:false,dataset:{},attributes:{},setAttribute(k,v){this.attributes[k]=v},addEventListener(k,cb){this[k]=cb},dispatchEvent(){},focus(){this.focused=true}};
 elems.voiceLanguage.value='ru-RU'; elems.searchBtn.click=function(){this.clicked=(this.clicked||0)+1};
 const instances=[];
 class Recognition{constructor(){instances.push(this)}start(){if(failStart)throw Error('test start failure');this.onstart?.()}stop(){this.onend?.()}abort(){this.onend?.()}}
 const context={document:{getElementById(id){return elems[id]}},window:{isSecureContext:secure,addEventListener(k,cb){listeners[k]=cb},...(support?{SpeechRecognition:Recognition}:{})},console:{info(){},error(){}},setTimeout(fn){listeners.timeout=fn;return 1},clearTimeout(){},Event:class{constructor(type){this.type=type}}};
 vm.runInNewContext(src,context);
 return {elems,instances,listeners,click(){elems.voiceBtn.click()}};
}
let s=scenario();assert.match(s.elems.voiceStatus.textContent,/Ready/);s.click();assert.equal(s.elems.voiceBtn.attributes['aria-pressed'],'true');assert.match(s.elems.voiceStatus.textContent,/Listening/);s.instances[0].onresult({results:[[{transcript:'мышь до 10 тысяч'}]]});assert.equal(s.elems.searchInput.value,'мышь до 10 тысяч');assert.equal(s.elems.searchBtn.clicked,1);assert.match(s.elems.voiceStatus.textContent,/Recognized/);s.instances[0].onend();assert.equal(s.elems.voiceBtn.attributes['aria-pressed'],'false');
s=scenario();s.click();s.instances[0].onerror({error:'not-allowed'});assert.match(s.elems.voiceStatus.textContent,/denied/);assert.equal(s.elems.voiceBtn.attributes['aria-pressed'],'false');
s=scenario({support:false});s.click();assert.match(s.elems.voiceStatus.textContent,/unavailable/);assert.equal(s.instances.length,0);
s=scenario({secure:false});s.click();assert.match(s.elems.voiceStatus.textContent,/unavailable/);
s=scenario({failStart:true});s.click();assert.match(s.elems.voiceStatus.textContent,/Cannot start/);assert.equal(s.elems.voiceBtn.attributes['aria-pressed'],'false');
s=scenario();s.click();s.listeners.timeout();assert.match(s.elems.voiceStatus.textContent,/timed out/);
s=scenario();s.click();s.instances[0].onerror({error:'network'});assert.match(s.elems.voiceStatus.textContent,/network error/);
console.log('PASS 7 voice scenarios: recognition+auto-search, denied, unsupported, insecure, start error, timeout, network');
