// ==UserScript==
// @name         M365 Copilot Upload Forensic Trace
// @namespace    local.m365.forensic
// @version      2.0.0
// @description  Local-only forensic capture for M365 Copilot image upload and ChatHub flows. Secrets and image bytes are redacted.
// @match        https://m365.cloud.microsoft/*
// @match        https://*.office.com/*
// @match        https://*.microsoft.com/*
// @run-at       document-start
// @inject-into  page
// @grant        none
// ==/UserScript==

(() => {
  'use strict';
  if (window.__M365_FORENSIC_TRACE__) return;
  window.__M365_FORENSIC_TRACE__ = true;

  const state = {
    enabled: false,
    rows: [],
    startedAt: null,
    maxRows: 2000,
    maxText: 800,
    watchAll: false,
    includeChatHubFrames: true,
    includePerformance: true,
  };
  const SECRET = /^(authorization|proxy-authorization|cookie|set-cookie|x-api-key|api-key|access_token|refresh_token|id_token|client_secret|password|secret|token)$/i;
  const SECRET_TEXT = /(access_token|refresh_token|id_token|client_secret|authorization|bearer\s|cookie\s*=|password\s*=)/i;
  const INTERESTING = /(substrate\.office\.com|UploadFile|upload|attach|file|image|media|content|resource|chathub|copilot)/i;
  const BINARY_CT = /(image\/|audio\/|video\/|application\/octet-stream|multipart\/form-data)/i;
  const RS = '\x1e';

  const now = () => new Date().toISOString();
  const clip = (v, n = state.maxText) => String(v ?? '').slice(0, n).replace(/[\r\n]+/g, ' ');
  const safeKey = k => SECRET.test(String(k || ''));
  const safeText = (v, key = '') => {
    if (safeKey(key) || SECRET_TEXT.test(String(v ?? ''))) return '[REDACTED]';
    const s = String(v ?? '');
    return s.length > state.maxText ? `[STRING length=${s.length} prefix=${JSON.stringify(s.slice(0, 180))}]` : s;
  };
  const safeURL = raw => {
    try {
      const u = new URL(String(raw), location.href);
      const params = [];
      u.searchParams.forEach((v, k) => params.push({name:k, value:safeKey(k) || /token|auth|key|secret|sig/i.test(k) ? '[REDACTED]' : safeText(v, k)}));
      return {origin:u.origin, path:u.pathname, query:params, hash:u.hash ? '[PRESENT]' : ''};
    } catch { return {raw: clip(raw)}; }
  };
  const safeHeaders = input => {
    const out = {};
    try {
      if (!input) return out;
      if (input instanceof Headers) input.forEach((v,k) => out[k] = safeKey(k) ? '[REDACTED]' : safeText(v,k));
      else if (Array.isArray(input)) input.forEach(([k,v]) => out[k] = safeKey(k) ? '[REDACTED]' : safeText(v,k));
      else Object.keys(input).forEach(k => out[k] = safeKey(k) ? '[REDACTED]' : safeText(input[k],k));
    } catch (e) { out.__error = String(e); }
    return out;
  };
  const headerValue = (h, name) => {
    try { return h?.get?.(name) || h?.[name] || h?.[name.toLowerCase()] || ''; } catch { return ''; }
  };
  const bodySummary = async (body, contentType = '') => {
    if (body == null) return {kind:'none'};
    try {
      if (body instanceof FormData) {
        const fields = [];
        for (const [name, value] of body.entries()) {
          if (value instanceof File || value instanceof Blob) fields.push({name, kind:'file', filename:value.name || '', type:value.type || '', size:value.size});
          else fields.push({name, kind:'text', length:String(value).length, value:safeText(value,name)});
        }
        return {kind:'FormData', fields};
      }
      if (body instanceof URLSearchParams) {
        const fields=[]; body.forEach((v,k)=>fields.push({name:k,value:safeText(v,k)}));
        return {kind:'URLSearchParams', fields};
      }
      if (body instanceof Blob) return {kind:'Blob', type:body.type, size:body.size};
      if (body instanceof ArrayBuffer) return {kind:'ArrayBuffer', bytes:body.byteLength};
      if (ArrayBuffer.isView(body)) return {kind:'TypedArray', bytes:body.byteLength};
      if (typeof body === 'string') {
        if (BINARY_CT.test(contentType) || /^data:/i.test(body)) return {kind:'string', length:body.length, binary:true};
        try { return {kind:'json', value:safeJSON(JSON.parse(body))}; } catch { return {kind:'string', length:body.length, preview:safeText(body)}; }
      }
      return {kind:Object.prototype.toString.call(body)};
    } catch (e) { return {kind:'error', error:String(e)}; }
  };
  const safeJSON = (v, key='') => {
    if (safeKey(key)) return '[REDACTED]';
    if (typeof v === 'string') {
      if (/^data:/i.test(v)) return `[DATA_URL length=${v.length}]`;
      return safeText(v,key);
    }
    if (Array.isArray(v)) return v.slice(0,40).map(x=>safeJSON(x,''));
    if (v && typeof v === 'object') { const o={}; Object.keys(v).slice(0,120).forEach(k=>o[k]=safeJSON(v[k],k)); return o; }
    return v;
  };
  const parseFrames = raw => {
    if (typeof raw !== 'string' || !state.includeChatHubFrames) return {kind:typeof raw, length:raw?.length || 0};
    return raw.split(RS).filter(Boolean).slice(0,30).map(frame => { try { return safeJSON(JSON.parse(frame)); } catch { return {unparsed:true,length:frame.length,prefix:clip(frame,240)}; } });
  };
  const add = (kind, data) => {
    if (!state.enabled) return;
    const row = {time:now(), elapsedMs:state.startedAt ? Math.round(performance.now()-state.startedAt) : 0, kind, ...data};
    state.rows.push(row); if (state.rows.length > state.maxRows) state.rows.splice(0,state.rows.length-state.maxRows);
    console.info('[M365 forensic trace]', row);
    updateUI();
  };
  const isInteresting = url => state.watchAll || INTERESTING.test(String(url));

  // Fetch
  const nativeFetch = window.fetch;
  window.fetch = async function(input, init) {
    const req = input instanceof Request ? input : null;
    const url = req?.url || String(input);
    const method = init?.method || req?.method || 'GET';
    const reqHeaders = safeHeaders(init?.headers || req?.headers);
    const ct = headerValue(init?.headers || req?.headers,'content-type');
    const body = await bodySummary(init?.body,ct);
    const t0=performance.now();
    try {
      const response = await nativeFetch.apply(this,arguments);
      if (isInteresting(url) && (method !== 'GET' || state.watchAll)) {
        const rh=safeHeaders(response.headers); let rb=null;
        const rct=headerValue(response.headers,'content-type');
        if (/json|text|javascript/i.test(rct)) { try { rb=safeJSON(await response.clone().json()); } catch { try { rb={text:clip(await response.clone().text())}; } catch {} } }
        add('fetch',{url:safeURL(url),method,status:response.status,statusText:response.statusText,requestHeaders:reqHeaders,requestBody:body,responseHeaders:rh,responseBody:rb,durationMs:Math.round(performance.now()-t0),credentials:init?.credentials || req?.credentials || ''});
      }
      return response;
    } catch(e) { if(isInteresting(url)) add('fetch-error',{url:safeURL(url),method,requestHeaders:reqHeaders,requestBody:body,error:String(e),durationMs:Math.round(performance.now()-t0)}); throw e; }
  };

  // XHR
  const NativeXHR=window.XMLHttpRequest, xp=NativeXHR.prototype;
  const xopen=xp.open, xsend=xp.send, xset=xp.setRequestHeader;
  xp.open=function(method,url,...rest){ this.__m365={method,url:String(url),headers:{},t0:0}; return xopen.call(this,method,url,...rest); };
  xp.setRequestHeader=function(k,v){ if(this.__m365) this.__m365.headers[k]=safeKey(k)?'[REDACTED]':safeText(v,k); return xset.call(this,k,v); };
  xp.send=function(body){ const m=this.__m365||{method:'?',url:'',headers:{}}; m.t0=performance.now(); this.addEventListener('loadend',async()=>{ if(isInteresting(m.url) && (m.method!=='GET'||state.watchAll)){let rb=null;try{const ct=this.getResponseHeader('content-type')||'';if(/json/i.test(ct))rb=safeJSON(JSON.parse(this.responseText));else if(/text/i.test(ct))rb={text:clip(this.responseText)};}catch{} add('xhr',{url:safeURL(m.url),method:m.method,status:this.status,statusText:this.statusText,requestHeaders:m.headers,requestBody:await bodySummary(body,m.headers['content-type']||''),responseHeaders:{contentType:this.getResponseHeader('content-type')||'',requestId:this.getResponseHeader('request-id')||this.getResponseHeader('x-request-id')||''},responseBody:rb,durationMs:Math.round(performance.now()-m.t0)});}}); return xsend.call(this,body); };

  // WebSocket / SignalR frames
  const NativeWS=window.WebSocket;
  function WrappedWS(url,protocols){
    const ws=protocols===undefined?new NativeWS(url):new NativeWS(url,protocols);
    const s=String(url), target=INTERESTING.test(s);
    if(target){ add('ws-open-attempt',{url:safeURL(s),protocols:protocols||[]}); const oldSend=ws.send; ws.send=function(data){ add('ws-send',{url:safeURL(s),data:parseFrames(data)}); return oldSend.call(this,data); }; ws.addEventListener('open',()=>add('ws-open',{url:safeURL(s),protocol:ws.protocol})); ws.addEventListener('message',e=>add('ws-recv',{url:safeURL(s),data:parseFrames(e.data)})); ws.addEventListener('close',e=>add('ws-close',{url:safeURL(s),code:e.code,reason:clip(e.reason),wasClean:e.wasClean})); ws.addEventListener('error',()=>add('ws-error',{url:safeURL(s)})); }
    return ws;
  }
  WrappedWS.prototype=NativeWS.prototype; for(const k of ['CONNECTING','OPEN','CLOSING','CLOSED']) WrappedWS[k]=NativeWS[k]; window.WebSocket=WrappedWS;

  // Performance resource timing captures requests made before/around our hooks.
  const seenPerf=new Set();
  setInterval(()=>{ if(!state.enabled||!state.includePerformance)return; for(const e of performance.getEntriesByType('resource')){if(seenPerf.has(e.name)||!isInteresting(e.name))continue;seenPerf.add(e.name);add('resource-timing',{url:safeURL(e.name),initiatorType:e.initiatorType,startTime:Math.round(e.startTime),durationMs:Math.round(e.duration),transferSize:e.transferSize,encodedBodySize:e.encodedBodySize,decodedBodySize:e.decodedBodySize});}},1000);

  function download(){ const blob=new Blob([JSON.stringify({meta:{page:location.href,origin:location.origin,startedAt:state.startedAt?new Date(Date.now()-(performance.now()-state.startedAt)).toISOString():null,version:'2.0.0'},records:state.rows},null,2)],{type:'application/json'}); const a=document.createElement('a');a.href=URL.createObjectURL(blob);a.download='m365-upload-forensic-trace.json';document.body.appendChild(a);a.click();a.remove();setTimeout(()=>URL.revokeObjectURL(a.href),3000); }
  function clear(){state.rows.length=0;updateUI();}
  function updateUI(){const el=document.getElementById('__m365_forensic_trace');if(el)el.querySelector('.count').textContent=String(state.rows.length);if(el)el.querySelector('.state').textContent=state.enabled?'CAPTURING':'PAUSED';}
  function ui(){if(!document.body||document.getElementById('__m365_forensic_trace'))return;const b=document.createElement('div');b.id='__m365_forensic_trace';b.innerHTML='<b>M365 forensic</b> <span class="state">PAUSED</span> <span class="count">0</span><br><button class="start">Start</button> <button class="stop">Stop</button> <button class="save">Export</button> <button class="clear">Clear</button> <label><input class="all" type="checkbox"> all requests</label>';b.style.cssText='position:fixed!important;right:8px!important;bottom:8px!important;z-index:2147483647!important;background:#171717!important;color:#fff!important;padding:10px!important;border:2px solid #fbbc04!important;border-radius:8px!important;font:12px monospace!important;display:block!important;opacity:.96!important';b.querySelector('.start').onclick=()=>{state.enabled=true;state.startedAt=performance.now();add('capture-start',{page:location.href});};b.querySelector('.stop').onclick=()=>{state.enabled=false;updateUI();};b.querySelector('.save').onclick=download;b.querySelector('.clear').onclick=clear;b.querySelector('.all').onchange=e=>state.watchAll=e.target.checked;document.body.appendChild(b);}
  window.__m365ForensicTrace={state,download,clear,start:()=>{state.enabled=true;state.startedAt=performance.now();}};
  const boot=()=>{ui();setTimeout(ui,500);setTimeout(ui,2000);}; if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',boot,{once:true});else boot();
  console.log('[M365 forensic trace] installed. Reload page, click Start, reproduce upload, then Export.');
})();
