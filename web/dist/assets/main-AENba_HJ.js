(function(){let e=document.createElement(`link`).relList;if(e&&e.supports&&e.supports(`modulepreload`))return;for(let e of document.querySelectorAll(`link[rel="modulepreload"]`))n(e);new MutationObserver(e=>{for(let t of e)if(t.type===`childList`)for(let e of t.addedNodes)e.tagName===`LINK`&&e.rel===`modulepreload`&&n(e)}).observe(document,{childList:!0,subtree:!0});function t(e){let t={};return e.integrity&&(t.integrity=e.integrity),e.referrerPolicy&&(t.referrerPolicy=e.referrerPolicy),t.credentials=e.crossOrigin===`use-credentials`?`include`:e.crossOrigin===`anonymous`?`omit`:`same-origin`,t}function n(e){if(e.ep)return;e.ep=!0;let n=t(e);fetch(e.href,n)}})();var e={xmlns:`http://www.w3.org/2000/svg`,width:24,height:24,viewBox:`0 0 24 24`,fill:`none`,stroke:`currentColor`,"stroke-width":2,"stroke-linecap":`round`,"stroke-linejoin":`round`},t=([e,n,r])=>{let i=document.createElementNS(`http://www.w3.org/2000/svg`,e);return Object.keys(n).forEach(e=>{i.setAttribute(e,String(n[e]))}),r?.length&&r.forEach(e=>{let n=t(e);i.appendChild(n)}),i},n=(n,r={})=>t([`svg`,{...e,...r},n]),r=(...e)=>e.filter((e,t,n)=>!!e&&e.trim()!==``&&n.indexOf(e)===t).join(` `).trim(),i=e=>{for(let t in e)if(t.startsWith(`aria-`)||t===`role`||t===`title`)return!0;return!1},a=e=>{let t=``,n=!1;for(let r of e)r===`-`||r===`_`||r<=` `?n=t.length>0:(t.length===0?t+=r.toLowerCase():t+=n?r.toUpperCase():r,n=!1);return t},o=e=>{let t=a(e);return t.charAt(0).toUpperCase()+t.slice(1)},s=e=>Array.from(e.attributes).reduce((e,t)=>(e[t.name]=t.value,e),{}),c=e=>typeof e==`string`?e:!e||!e.class?``:e.class&&typeof e.class==`string`?e.class.split(` `):e.class&&Array.isArray(e.class)?e.class:``,l=(t,{nameAttr:a,icons:l,attrs:u})=>{let d=t.getAttribute(a);if(d==null)return;let f=l[o(d)];if(!f)return console.warn(`${t.outerHTML} icon name was not found in the provided icons object.`);let p=s(t),m=i(p)?{}:{"aria-hidden":`true`},h={...e,"data-lucide":d,...m,...u,...p},g=c(p),_=c(u),v=r(`lucide`,`lucide-${d}`,...g,..._);v&&Object.assign(h,{class:v});let y=n(f,h);return t.parentNode?.replaceChild(y,t)},u=[[`path`,{d:`M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2`}]],d=[[`path`,{d:`M20 6 9 17l-5-5`}]],f=[[`path`,{d:`m6 9 6 6 6-6`}]],p=[[`circle`,{cx:`12`,cy:`12`,r:`10`}],[`path`,{d:`m16 9-5.5 5.5L8 12`}]],m=[[`circle`,{cx:`12`,cy:`12`,r:`10`}],[`path`,{d:`M8 12h8`}],[`path`,{d:`M12 8v8`}]],h=[[`circle`,{cx:`12`,cy:`12`,r:`10`}],[`path`,{d:`m15 9-6 6`}],[`path`,{d:`m9 9 6 6`}]],g=[[`rect`,{width:`14`,height:`14`,x:`8`,y:`8`,rx:`2`,ry:`2`}],[`path`,{d:`M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2`}]],_=[[`path`,{d:`M12 15V3`}],[`path`,{d:`M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4`}],[`path`,{d:`m7 10 5 5 5-5`}]],v=[[`path`,{d:`M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0`}],[`circle`,{cx:`12`,cy:`12`,r:`3`}]],y=[[`path`,{d:`M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z`}],[`path`,{d:`M14 2v5a1 1 0 0 0 1 1h5`}],[`path`,{d:`M12 18v-6`}],[`path`,{d:`m9 15 3 3 3-3`}]],b=[[`path`,{d:`M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z`}],[`path`,{d:`M14 2v5a1 1 0 0 0 1 1h5`}],[`path`,{d:`M10 9H8`}],[`path`,{d:`M16 13H8`}],[`path`,{d:`M16 17H8`}]],x=[[`circle`,{cx:`18`,cy:`18`,r:`3`}],[`circle`,{cx:`6`,cy:`6`,r:`3`}],[`path`,{d:`M6 21V9a9 9 0 0 0 9 9`}]],S=[[`path`,{d:`M15 21v-8a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v8`}],[`path`,{d:`M3 10a2 2 0 0 1 .709-1.528l7-6a2 2 0 0 1 2.582 0l7 6A2 2 0 0 1 21 10v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z`}]],C=[[`circle`,{cx:`12`,cy:`12`,r:`10`}],[`path`,{d:`M12 16v-4`}],[`path`,{d:`M12 8h.01`}]],w=[[`path`,{d:`M2.586 17.414A2 2 0 0 0 2 18.828V21a1 1 0 0 0 1 1h3a1 1 0 0 0 1-1v-1a1 1 0 0 1 1-1h1a1 1 0 0 0 1-1v-1a1 1 0 0 1 1-1h.172a2 2 0 0 0 1.414-.586l.814-.814a6.5 6.5 0 1 0-4-4z`}],[`circle`,{cx:`16.5`,cy:`7.5`,r:`.5`,fill:`currentColor`}]],ee=[[`rect`,{width:`7`,height:`7`,x:`3`,y:`3`,rx:`1`}],[`rect`,{width:`7`,height:`7`,x:`14`,y:`3`,rx:`1`}],[`rect`,{width:`7`,height:`7`,x:`14`,y:`14`,rx:`1`}],[`rect`,{width:`7`,height:`7`,x:`3`,y:`14`,rx:`1`}]],T=[[`path`,{d:`M3 5h.01`}],[`path`,{d:`M3 12h.01`}],[`path`,{d:`M3 19h.01`}],[`path`,{d:`M8 5h13`}],[`path`,{d:`M8 12h13`}],[`path`,{d:`M8 19h13`}]],te=[[`path`,{d:`M21 12a9 9 0 1 1-6.219-8.56`}]],E=[[`path`,{d:`m16 17 5-5-5-5`}],[`path`,{d:`M21 12H9`}],[`path`,{d:`M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4`}]],D=[[`rect`,{width:`20`,height:`14`,x:`2`,y:`3`,rx:`2`}],[`line`,{x1:`8`,x2:`16`,y1:`21`,y2:`21`}],[`line`,{x1:`12`,x2:`12`,y1:`17`,y2:`21`}]],O=[[`path`,{d:`M11 21.73a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73z`}],[`path`,{d:`M12 22V12`}],[`polyline`,{points:`3.29 7 12 12 20.71 7`}],[`path`,{d:`m7.5 4.27 9 5.15`}]],ne=[[`path`,{d:`M5 5a2 2 0 0 1 3.008-1.728l11.997 6.998a2 2 0 0 1 .003 3.458l-12 7A2 2 0 0 1 5 19z`}]],k=[[`path`,{d:`M5 12h14`}],[`path`,{d:`M12 5v14`}]],A=[[`path`,{d:`M18.36 6.64A9 9 0 0 1 20.77 15`}],[`path`,{d:`M6.16 6.16a9 9 0 1 0 12.68 12.68`}],[`path`,{d:`M12 2v4`}],[`path`,{d:`m2 2 20 20`}]],re=[[`path`,{d:`M12 2v10`}],[`path`,{d:`M18.4 6.6a9 9 0 1 1-12.77.04`}]],ie=[[`path`,{d:`M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8`}],[`path`,{d:`M21 3v5h-5`}],[`path`,{d:`M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16`}],[`path`,{d:`M8 16H3v5`}]],ae=[[`path`,{d:`M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8`}],[`path`,{d:`M3 3v5h5`}],[`path`,{d:`M12 7v5l4 2`}]],oe=[[`path`,{d:`M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8`}],[`path`,{d:`M3 3v5h5`}]],se=[[`path`,{d:`M15.2 3a2 2 0 0 1 1.4.6l3.8 3.8a2 2 0 0 1 .6 1.4V19a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z`}],[`path`,{d:`M17 21v-7a1 1 0 0 0-1-1H8a1 1 0 0 0-1 1v7`}],[`path`,{d:`M7 3v4a1 1 0 0 0 1 1h7`}]],ce=[[`path`,{d:`m21 21-4.34-4.34`}],[`circle`,{cx:`11`,cy:`11`,r:`8`}]],le=[[`rect`,{width:`20`,height:`8`,x:`2`,y:`2`,rx:`2`,ry:`2`}],[`rect`,{width:`20`,height:`8`,x:`2`,y:`14`,rx:`2`,ry:`2`}],[`line`,{x1:`6`,x2:`6.01`,y1:`6`,y2:`6`}],[`line`,{x1:`6`,x2:`6.01`,y1:`18`,y2:`18`}]],ue=[[`path`,{d:`M9.671 4.136a2.34 2.34 0 0 1 4.659 0 2.34 2.34 0 0 0 3.319 1.915 2.34 2.34 0 0 1 2.33 4.033 2.34 2.34 0 0 0 0 3.831 2.34 2.34 0 0 1-2.33 4.033 2.34 2.34 0 0 0-3.319 1.915 2.34 2.34 0 0 1-4.659 0 2.34 2.34 0 0 0-3.32-1.915 2.34 2.34 0 0 1-2.33-4.033 2.34 2.34 0 0 0 0-3.831A2.34 2.34 0 0 1 6.35 6.051a2.34 2.34 0 0 0 3.319-1.915`}],[`circle`,{cx:`12`,cy:`12`,r:`3`}]],de=[[`path`,{d:`M10 5H3`}],[`path`,{d:`M12 19H3`}],[`path`,{d:`M14 3v4`}],[`path`,{d:`M16 17v4`}],[`path`,{d:`M21 12h-9`}],[`path`,{d:`M21 19h-5`}],[`path`,{d:`M21 5h-7`}],[`path`,{d:`M8 10v4`}],[`path`,{d:`M8 12H3`}]],fe=[[`rect`,{width:`18`,height:`18`,x:`3`,y:`3`,rx:`2`}]],pe=[[`path`,{d:`M10 11v6`}],[`path`,{d:`M14 11v6`}],[`path`,{d:`M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6`}],[`path`,{d:`M3 6h18`}],[`path`,{d:`M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2`}]],me=[[`path`,{d:`m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3`}],[`path`,{d:`M12 9v4`}],[`path`,{d:`M12 17h.01`}]],he=[[`path`,{d:`M12 3v12`}],[`path`,{d:`m17 8-5-5-5 5`}],[`path`,{d:`M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4`}]],ge=[[`path`,{d:`M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2`}],[`circle`,{cx:`9`,cy:`7`,r:`4`}],[`line`,{x1:`22`,x2:`16`,y1:`11`,y2:`11`}]],_e=[[`path`,{d:`M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2`}],[`circle`,{cx:`9`,cy:`7`,r:`4`}],[`line`,{x1:`19`,x2:`19`,y1:`8`,y2:`14`}],[`line`,{x1:`22`,x2:`16`,y1:`11`,y2:`11`}]],ve=[[`path`,{d:`M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2`}],[`path`,{d:`M16 3.128a4 4 0 0 1 0 7.744`}],[`path`,{d:`M22 21v-2a4 4 0 0 0-3-3.87`}],[`circle`,{cx:`9`,cy:`7`,r:`4`}]],ye=[[`path`,{d:`M18 6 6 18`}],[`path`,{d:`m6 6 12 12`}]],be=({icons:e={},nameAttr:t=`data-lucide`,attrs:n={},root:r=document,inTemplates:i}={})=>{if(!Object.values(e).length)throw Error(`Please provide an icons object.
If you want to use all the icons you can import it like:
 \`import { createIcons, icons } from 'lucide';
lucide.createIcons({icons});\``);if(r===void 0)throw Error("`createIcons()` only works in a browser environment.");if(Array.from(r.querySelectorAll(`[${t}]`)).forEach(r=>l(r,{nameAttr:t,icons:e,attrs:n})),i&&Array.from(r.querySelectorAll(`template`)).forEach(r=>be({icons:e,nameAttr:t,attrs:n,root:r.content,inTemplates:i})),t===`data-lucide`){let t=r.querySelectorAll(`[icon-name]`);t.length>0&&(console.warn(`[Lucide] Some icons were found with the now deprecated icon-name attribute. These will still be replaced for backwards compatibility, but will no longer be supported in v1.0 and you should switch to data-lucide`),Array.from(t).forEach(t=>l(t,{nameAttr:`icon-name`,icons:e,attrs:n})))}},xe={Activity:u,AlertTriangle:me,Check:d,CheckCircle2:p,ChevronDown:f,Copy:g,Download:_,Eye:v,FileDown:y,FileText:b,GitMerge:x,History:ae,Home:S,Info:C,KeyRound:w,LayoutGrid:ee,List:T,Loader2:te,LogOut:E,Monitor:D,Package:O,Play:ne,Plus:k,PlusCircle:m,Power:re,PowerOff:A,RefreshCw:ie,RotateCcw:oe,Save:se,Search:ce,Server:le,Settings:ue,SlidersHorizontal:de,Square:fe,Trash2:pe,Upload:he,UserMinus:ge,UserPlus:_e,Users:ve,X:ye,XCircle:h};function j(e=document){e&&be({icons:xe,attrs:{"stroke-width":1.5,width:16,height:16},nameAttr:`data-lucide`,root:e})}var M=`/api`,Se={Accept:`application/json`,"Content-Type":`application/json`},Ce=class extends Error{constructor(e,t){super(t&&t.mensaje||`La consola respondió con el código ${e}.`),this.name=`ApiError`,this.estado=e,this.mensaje=this.message,this.campo=t&&t.campo,this.tecnico=t&&t.tecnico,this.solicitud=``}get noImplementado(){return this.estado===501}};function we(e,t){if(!t)return M+e;let n=Object.entries(t).filter(([,e])=>e!=null&&e!==!1).map(([e,t])=>`${encodeURIComponent(e)}=${encodeURIComponent(String(t))}`);return n.length>0?`${M}${e}?${n.join(`&`)}`:M+e}async function N(e,t,{cuerpo:n,consulta:r,descarga:i=!1}={}){let a={method:e,headers:{Accept:`application/json`}};n!==void 0&&(a.headers=Se,a.body=JSON.stringify(n));let o=await fetch(we(t,r),a),s=o.headers.get(`X-Request-ID`)||``;if(o.status===204)return null;let c=o.headers.get(`content-type`)||``;if(!o.ok){let e=null;if(c.includes(`application/json`))try{e=await o.json()}catch{e=null}let t=new Ce(o.status,e);throw t.solicitud=s,t}return i?o.blob():c.includes(`application/json`)?o.json():o.text()}function Te(e,t){let n=URL.createObjectURL(e),r=document.createElement(`a`);r.href=n,r.download=t,document.body.appendChild(r),r.click(),r.remove(),URL.revokeObjectURL(n)}var P={auth:{estado:()=>N(`GET`,`/auth/estado`),configurar:(e,t)=>N(`POST`,`/auth/setup`,{cuerpo:{usuario:e,password:t}}),entrar:(e,t)=>N(`POST`,`/auth/login`,{cuerpo:{usuario:e,password:t}}),salir:()=>N(`POST`,`/auth/logout`)},sesion:{obtener:()=>N(`GET`,`/session`),vincular:e=>N(`PUT`,`/session`,{cuerpo:{vm:e}}),registrar:(e,t)=>N(`POST`,`/session/register`,{cuerpo:{usuario:e,password:t}}),olvidar:()=>N(`DELETE`,`/session/register`),diagnostico:()=>N(`GET`,`/session/diagnostico`)},historial:{listar:e=>N(`GET`,`/historial`,{consulta:e})},maquinas:{listar:e=>N(`GET`,`/vms`,{consulta:{q:e&&e.trim()?e.trim():void 0}}),encender:e=>N(`POST`,`/vms/${encodeURIComponent(e)}/poweron`)},paquetes:{listar:()=>N(`GET`,`/packages`),instalar:e=>N(`POST`,`/packages`,{cuerpo:{nombre:e}}),eliminar:e=>N(`DELETE`,`/packages/${encodeURIComponent(e)}`)},llaves:{generarRoot:()=>N(`POST`,`/keys/root`),estadoRoot:()=>N(`GET`,`/keys/root/status`),descargarRoot:()=>N(`GET`,`/keys/root`,{descarga:!0})},issue:{obtener:()=>N(`GET`,`/issue`),reemplazar:e=>N(`PUT`,`/issue`,{cuerpo:e})},hostname:{obtener:()=>N(`GET`,`/hostname`),reemplazar:e=>N(`PUT`,`/hostname`,{cuerpo:e})},usuarios:{listar:()=>N(`GET`,`/users`),crear:e=>N(`POST`,`/users`,{cuerpo:e}),obtener:e=>N(`GET`,`/users/${encodeURIComponent(e)}`),eliminar:(e,t={})=>N(`DELETE`,`/users/${encodeURIComponent(e)}`,{consulta:{files:t.archivos?!0:void 0}}),descargarLlave:e=>N(`GET`,`/users/${encodeURIComponent(e)}/key`,{descarga:!0}),generarLlave:(e,t={})=>N(`POST`,`/users/${encodeURIComponent(e)}/key`,{consulta:{regenerar:t.regenerar?!0:void 0}})},unidades:{listar:()=>N(`GET`,`/units`),obtener:e=>N(`GET`,`/units/${encodeURIComponent(e)}`),estadoDetallado:e=>N(`GET`,`/units/${encodeURIComponent(e)}/status`),desplegar:(e,t)=>N(`PUT`,`/units/${encodeURIComponent(e)}`,{cuerpo:t}),cambiarEstado:(e,t)=>N(`PATCH`,`/units/${encodeURIComponent(e)}`,{cuerpo:t}),reiniciar:e=>N(`POST`,`/units/${encodeURIComponent(e)}/restarts`)},haproxy:{listar:()=>N(`GET`,`/haproxy/backend/servers`),agregar:e=>N(`POST`,`/haproxy/backend/servers`,{cuerpo:e}),eliminar:e=>N(`DELETE`,`/haproxy/backend/servers/${encodeURIComponent(e)}`)}};function F(e){return e==null?``:String(e).replaceAll(`&`,`&amp;`).replaceAll(`<`,`&lt;`).replaceAll(`>`,`&gt;`).replaceAll(`"`,`&quot;`).replaceAll(`'`,`&#39;`)}function I(e){switch(e){case`running`:return{texto:`Encendida`,clase:`text-vm-success border-vm-success bg-[#e8f5e9]`,punto:`bg-vm-success`};case`paused`:return{texto:`Pausada`,clase:`text-[#b9770e] border-vm-warning bg-[#fdf3e3]`,punto:`bg-vm-warning`};case`poweroff`:case`saved`:return{texto:`Apagada`,clase:`text-vm-muted border-vm-muted bg-[#f2f4f4]`,punto:`bg-vm-muted`};default:return{texto:`Desconocido`,clase:`text-vm-fgSecondary border-vm-border bg-vm-bg`,punto:`bg-vm-muted`}}}function L(e){switch(e){case`active`:return{texto:`activa`,clase:`text-vm-success border-vm-success bg-[#e8f5e9]`,punto:`bg-vm-success`};case`failed`:return{texto:`falló`,clase:`text-vm-accent border-vm-accent bg-[#fdf1ef]`,punto:`bg-vm-accent`};case`inactive`:return{texto:`inactiva`,clase:`text-vm-fgSecondary border-vm-border bg-vm-bg`,punto:`bg-vm-muted`};default:return{texto:`desconocida`,clase:`text-vm-fgSecondary border-vm-border bg-vm-bg`,punto:`bg-vm-muted`}}}function Ee(e){switch(e){case`up`:return{texto:`UP`,clase:`text-vm-success border-vm-success bg-[#e8f5e9]`,punto:`bg-vm-success`};case`down`:return{texto:`DOWN`,clase:`text-vm-accent border-vm-accent bg-[#fdf1ef]`,punto:`bg-vm-accent`};default:return{texto:`DESCONOCIDO`,clase:`text-vm-fgSecondary border-vm-border bg-vm-bg`,punto:`bg-vm-muted`}}}function R(e){return e&&e.noImplementado?`Este módulo aún no está conectado con la máquina.`:e&&e.mensaje?e.mensaje:e instanceof TypeError?`No fue posible contactar con la consola. Verifica que el servidor esté en ejecución.`:`Ocurrió un error inesperado al ejecutar la operación.`}function De(e,t=``){return{ok:!1,mensaje:R(e),operacion:t,codigo:e&&e.codigo,stderr:e&&e.stderr||``,tecnico:e&&(e.tecnico||e.campo)||``,solicitud:e&&e.solicitud||``}}function z(e,t,n=`text-vm-fgSecondary`){return`<tr><td colspan="${e}" class="px-3 py-3 text-center text-[12px] ${n}">${F(t)}</td></tr>`}function B(e,t){e.querySelectorAll(`button, input, select, textarea`).forEach(e=>{e.disabled=t})}async function V({accion:e,operacion:t,recargar:n,informar:r}){try{let i=await e();return r({ok:!0,mensaje:i&&i.mensaje||``,operacion:i&&i.operacion||t,codigo:i&&i.codigo||0,salida:i&&i.salida||``,stderr:i&&i.stderr||``,tecnico:i&&i.tecnico||``}),n&&n(),!0}catch(e){return r(De(e,t)),!1}}var Oe=200;function ke({alVincular:e}={}){let t=document.createElement(`div`);t.className=`flex items-center gap-1.5`,t.innerHTML=`
    <div class="relative">
      <button id="vm-boton" type="button" class="btn h-vm-btn pr-1.5 max-w-[260px]"
        aria-haspopup="listbox" aria-expanded="false" aria-controls="vm-panel" title="Elegir la máquina virtual (registrada o con Guest Additions, red puente/host-only)">
        <span id="vm-punto" class="w-1.5 h-1.5 rounded-full bg-vm-muted shrink-0"></span>
        <span id="vm-nombre" class="truncate">Buscando máquinas…</span>
        <i data-lucide="chevron-down" class="w-3.5 h-3.5 shrink-0"></i>
      </button>
      <div id="vm-panel" class="hidden absolute left-0 top-full mt-1 w-[330px] z-30 bg-vm-surface border border-vm-border shadow-none">
        <div class="flex items-center gap-1.5 px-2 h-vm-input border-b border-vm-border">
          <i data-lucide="search" class="w-4 h-4 text-vm-fgSecondary"></i>
          <input id="vm-busqueda" class="flex-1 bg-transparent outline-none text-[12px]" placeholder="Buscar por nombre o UUID…"
            autocomplete="off" spellcheck="false" aria-label="Buscar máquina virtual">
        </div>
        <ul id="vm-lista" role="listbox" aria-label="Máquinas virtuales" class="max-h-[280px] overflow-auto"></ul>
        <footer id="vm-pie" class="px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary"></footer>
      </div>
    </div>
    <button id="vm-encender" type="button" class="btn h-vm-btn px-1.5 hidden" title="Encender la máquina virtual">
      <i data-lucide="power" class="w-4 h-4"></i>
    </button>
    <button id="vm-olvidar" type="button" class="btn h-vm-btn px-1.5 hidden" title="Olvidar el registro de esta máquina">
      <i data-lucide="trash-2" class="w-4 h-4"></i>
    </button>
  `,j(t);let n=t.querySelector(`#vm-boton`),r=t.querySelector(`#vm-nombre`),i=t.querySelector(`#vm-punto`),a=t.querySelector(`#vm-panel`),o=t.querySelector(`#vm-busqueda`),s=t.querySelector(`#vm-lista`),c=t.querySelector(`#vm-pie`),l=t.querySelector(`#vm-encender`),u=t.querySelector(`#vm-olvidar`),d=[],f=0,p=0,m=null,h=`vmcons:maquinas-ocultas`,g=()=>{try{return new Set(JSON.parse(window.localStorage.getItem(h)??`[]`))}catch{return new Set}},_=()=>{try{window.localStorage.setItem(h,JSON.stringify([...v]))}catch{}},v=g(),y=e=>e.administrable===!1&&e.estado===`running`&&v.has(e.nombre),b=()=>d.filter(e=>!y(e)),x=e=>{if(m=e,!e){r.textContent=`Elegir máquina`,i.className=`w-1.5 h-1.5 rounded-full bg-vm-muted shrink-0`,l.classList.add(`hidden`),u.classList.add(`hidden`);return}let t=I(e.estado);r.textContent=e.nombre,i.className=`w-1.5 h-1.5 rounded-full ${t.punto} shrink-0`,n.title=`Máquina activa: ${e.nombre} (${t.texto})`,l.classList.toggle(`hidden`,e.estado===`running`),l.disabled=!1,u.classList.toggle(`hidden`,!e.registrada)},S=()=>{let e=b();e.length===0?s.innerHTML=`
        <li class="px-2 py-3 text-center text-[11px] text-vm-fgSecondary">
          ${d.length>0?`Todas las máquinas encendidas están ocultas por falta de Guest Additions`:`No hay máquinas administrables que coincidan con la búsqueda`}
        </li>
      `:(s.innerHTML=e.map(e=>{let t=I(e.estado),n=e.administrable!==!1,r=e.advertencia||`Guest Additions no está disponible`;return n?`
        <li>
          <button type="button" role="option" aria-selected="${e.activa}"
            ${n?`data-maquina="${F(e.nombre)}"`:`disabled aria-disabled="true"`}
            class="w-full text-left px-2 py-1.5 flex items-center gap-2 ${n?`hover:bg-[#fdf5f4]`:`opacity-60 cursor-not-allowed`} ${e.activa?`border-l-2 border-[#C0392B] bg-[#faf9f9]`:`border-l-2 border-transparent`}">
            <span class="w-1.5 h-1.5 rounded-full shrink-0 ${t.punto}"></span>
            <span class="flex-1 min-w-0">
              <span class="block text-[12px] truncate">${F(e.nombre)}</span>
              <span class="block text-[10px] font-mono text-vm-fgSecondary truncate">${F(e.uuid)}</span>
            </span>
            ${e.registrada?``:`<span class="text-[10px] text-vm-accent shrink-0">Sin registrar</span>`}
            <span class="text-[10px] text-vm-fgSecondary shrink-0">${t.texto}</span>
            ${e.activa?`<i data-lucide="check" class="w-3.5 h-3.5 text-[#C0392B] shrink-0"></i>`:``}
          </button>
        </li>
      `:`
          <li class="px-2 py-1.5 flex items-center gap-2 opacity-60 border-l-2 border-transparent">
            <span class="w-1.5 h-1.5 rounded-full shrink-0 ${t.punto}"></span>
            <span class="flex-1 min-w-0">
              <span class="block text-[12px] truncate">${F(e.nombre)}</span>
              <span class="block text-[10px] font-mono text-vm-fgSecondary truncate">${F(e.uuid)}</span>
            </span>
            <span class="text-[10px] text-vm-accent shrink-0 max-w-[88px] truncate" title="${F(r)}">Sin Guest Additions</span>
            <button type="button" data-ocultar="${F(e.nombre)}"
              class="text-[10px] underline text-vm-accent shrink-0">Ocultar</button>
          </li>
        `}).join(``),j(s))},C=()=>{let e=b().length,t=d.filter(y).length;t>0?(c.innerHTML=`${e} máquina(s) · ${t} oculta(s) por falta de Guest Additions
        <button id="vm-mostrar-ocultas" type="button" class="underline text-vm-accent">Mostrar</button>`,c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary`,c.querySelector(`#vm-mostrar-ocultas`)?.addEventListener(`click`,()=>{v.clear(),_(),S(),C()})):(c.textContent=`${e} máquina(s) · las encendidas sin Guest Additions muestran un aviso`,c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary`)},w=async(e=``)=>{let t=++f;c.textContent=`Consultando…`,c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary`;try{let n=await P.maquinas.listar(e);if(t!==f)return;d=n.maquinas??[];let r=d.find(e=>e.activa);x(r??m),S(),C()}catch(e){if(t!==f)return;let n=R(e);d.length===0&&(s.innerHTML=`
          <li class="px-2 py-3 text-center text-[11px] text-vm-accent">${F(n)}</li>
        `),c.textContent=d.length===0?`No se pudo leer el inventario`:`No se pudo actualizar el inventario: ${n}`,c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent`}},ee=()=>{a.classList.remove(`hidden`),n.setAttribute(`aria-expanded`,`true`),o.value=``,o.focus(),w(``)},T=()=>{a.classList.add(`hidden`),n.setAttribute(`aria-expanded`,`false`)},te=async t=>{c.textContent=`Vinculando con ${t}…`;try{let n={...await P.sesion.vincular(t),activa:!0};x(n),T(),e&&e(n)}catch(e){c.textContent=R(e),c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent`}};n.addEventListener(`click`,()=>{a.classList.contains(`hidden`)?ee():T()});let E=null,D=!1,O=()=>{E?.overlay.remove(),E=null},ne=e=>{O(),D=!1;let t=document.createElement(`div`);t.id=`vm-power-overlay`,t.className=`fixed inset-0 z-50 bg-black/40 flex items-center justify-center px-3`,t.innerHTML=`
      <section class="card w-[440px] max-w-[94vw] p-3 flex flex-col gap-2" role="dialog" aria-modal="true" aria-labelledby="vm-power-titulo">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="power" class="w-4 h-4"></i><span id="vm-power-titulo" class="text-[13px] font-medium">Encendiendo ${F(e)}…</span></div>
          <button id="vm-power-cerrar" type="button" class="btn h-vm-btn px-1.5" title="Cerrar"><i data-lucide="x" class="w-4 h-4"></i></button>
        </div>
        <div class="h-1 w-full bg-vm-bg border border-vm-border overflow-hidden">
          <div class="barra-indeterminada h-full bg-vm-accent"></div>
        </div>
        <p id="vm-power-estado" class="text-[12px] text-vm-fgSecondary" aria-live="polite">La máquina está arrancando…</p>
        <p id="vm-power-motivo" class="text-[11px] text-vm-accent hidden"></p>
        <div id="vm-power-acciones" class="hidden gap-2 justify-end">
          <button id="vm-power-ocultar" type="button" class="btn">Ocultar de la lista</button>
          <button id="vm-power-mantener" type="button" class="btn btn-primary">Mantener en la lista</button>
        </div>
      </section>
    `,document.body.appendChild(t),j(t);let n={overlay:t,titulo:t.querySelector(`#vm-power-titulo`),estado:t.querySelector(`#vm-power-estado`),motivo:t.querySelector(`#vm-power-motivo`),acciones:t.querySelector(`#vm-power-acciones`),cerrar:t.querySelector(`#vm-power-cerrar`),ocultar:t.querySelector(`#vm-power-ocultar`),mantener:t.querySelector(`#vm-power-mantener`)};return n.cerrar.addEventListener(`click`,()=>{D=!0,O()}),n.ocultar.addEventListener(`click`,()=>{v.add(e),_(),D=!0,O(),S(),C()}),n.mantener.addEventListener(`click`,()=>{n.acciones.classList.add(`hidden`),n.acciones.classList.remove(`flex`),n.estado.textContent=`La máquina seguirá en la lista. Puedes cerrar este aviso cuando termine de arrancar.`}),E=n,n},k=({estado:e,motivo:t,acciones:n=!1})=>{E&&(e&&(E.estado.textContent=e),t?(E.motivo.textContent=t,E.motivo.classList.remove(`hidden`)):(E.motivo.textContent=``,E.motivo.classList.add(`hidden`)),E.acciones.classList.toggle(`hidden`,!n),E.acciones.classList.toggle(`flex`,n))},A=async()=>{try{let e=await P.sesion.diagnostico(),t=e.problemas??[];return!((e.guest_additions_run_level??0)<2)&&t.length===0?``:t.length>0?t.join(` `):`Guest Additions no está listo.`}catch{return``}},re=async e=>{let t=Date.now(),n=!1;for(;Date.now()-t<9e4;){if(D)return null;try{let e=await P.sesion.obtener();if(e.estado===`running`)return e}catch{}try{await w(``)}catch{}let r=Math.floor((Date.now()-t)/1e3),i=d.find(t=>t.nombre===e);if(!n&&r>=30&&i?.administrable===!1){let t=i.advertencia||await A();n=!0,k({estado:`${e} sigue arrancando. Aún no se detecta Guest Additions; puedes esperar, ocultarla o mantenerla en la lista.`,motivo:t,acciones:!0})}n||(r>=30?k({estado:`Esperando Guest Additions o SSH…`}):r>=8&&k({estado:`Esperando a que arranque el sistema…`})),await new Promise(e=>window.setTimeout(e,2e3))}return null};return l.addEventListener(`click`,async()=>{if(m){l.disabled=!0,l.innerHTML=`<i data-lucide="loader-2" class="w-4 h-4 animate-spin"></i>`,j(l),ne(m.nombre),c.textContent=`La máquina está arrancando…`,c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary`;try{let t=await P.maquinas.encender(m.nombre);if(x({...t,activa:!0}),k({estado:`${t.nombre} encendida. Esperando la sesión…`}),c.textContent=`${t.nombre} está en ejecución; esperando Guest Additions o SSH…`,await re(t.nombre))k({estado:`${t.nombre} en ejecución y sesión disponible.`}),c.textContent=`${t.nombre} está en ejecución`,e&&await Promise.resolve(e(t)),await w(``),window.setTimeout(O,900);else if(!D){let e=await A();k({estado:e?`${t.nombre} está encendida, pero no se puede administrar todavía.`:`${t.nombre} está encendida, pero Guest Additions/SSH aún no responde.`,motivo:e,acciones:!0}),c.textContent=e?`${t.nombre} está encendida; revisa el aviso de Guest Additions.`:`La máquina está encendida; actualiza en unos segundos.`,c.className=e?`px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent`:`px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary`}}catch(e){k({estado:R(e),acciones:!0}),c.textContent=R(e),c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent`}finally{l.innerHTML=`<i data-lucide="power" class="w-4 h-4"></i>`,j(l),l.disabled=!1}}}),u.addEventListener(`click`,async()=>{if(!m)return;let e=m.nombre;if(window.confirm(`¿Olvidar el registro de ${e}? Tendrás que registrar la máquina otra vez para administrarla.`)){u.disabled=!0;try{await P.sesion.olvidar(),x({...m,registrada:!1}),await w(``),c.textContent=`Registro de ${e} olvidado; vuelve a registrarla para administrarla.`,c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary`}catch(e){c.textContent=R(e),c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent`}finally{u.disabled=!1}}}),o.addEventListener(`input`,()=>{window.clearTimeout(p),p=window.setTimeout(()=>w(o.value),Oe)}),o.addEventListener(`keydown`,e=>{e.key===`Escape`&&(T(),n.focus())}),s.addEventListener(`click`,e=>{let t=e.target.closest(`[data-ocultar]`);if(t){v.add(t.dataset.ocultar),_(),S(),C();return}let n=e.target.closest(`[data-maquina]`);n&&te(n.dataset.maquina)}),document.addEventListener(`click`,e=>{t.contains(e.target)||T()}),document.addEventListener(`keydown`,e=>{e.key===`Escape`&&T()}),w(``),t}function Ae({usuario:e=``,alSalir:t}={}){let n=document.createElement(`header`);n.className=`h-vm-header shrink-0 bg-vm-surface border-b border-vm-border flex items-center justify-between px-3`,n.innerHTML=`
    <div class="flex items-center gap-4">
      <div class="flex items-center gap-2.5 pr-4 border-r border-vm-border">
        <span class="w-2.5 h-2.5 rounded-full bg-vm-accent animate-pulse"></span>
        <span class="font-semibold text-vm-fg tracking-tight text-[16px]">vmcons</span>
      </div>
      <div class="flex items-center gap-2.5 text-[12px]">
        <span id="vm-slot"></span>
        <span class="h-4 w-px bg-vm-border"></span>
        <span id="vm-ip" class="font-mono text-vm-fg">—</span>
      </div>
    </div>
    <div class="flex items-center gap-3 text-[12px]">
      <span class="flex items-center gap-1.5" title="Estado de la sesión">
        <span id="vm-punto" class="w-1.5 h-1.5 rounded-full bg-vm-muted"></span>
        <span id="vm-estado" class="text-vm-fgSecondary">Consultando…</span>
      </span>
      <button id="btn-refresh" class="btn h-vm-btn px-1.5" title="Consultar la sesión">
        <i data-lucide="refresh-cw" class="w-4 h-4"></i>
      </button>
      <span id="vm-usuario" class="hidden items-center gap-1 text-vm-fgSecondary"></span>
      <button id="btn-salir" type="button" class="btn h-vm-btn px-1.5 hidden" title="Cerrar la sesión de la consola">
        <i data-lucide="log-out" class="w-4 h-4"></i>
      </button>
    </div>
  `,j(n);let r=()=>je(n);if(n.querySelector(`#vm-slot`).appendChild(ke({alVincular:e=>{r(),document.dispatchEvent(new CustomEvent(`vmcons:maquina`,{detail:e}))}})),n.querySelector(`#btn-refresh`).addEventListener(`click`,r),e){let r=n.querySelector(`#vm-usuario`),i=n.querySelector(`#btn-salir`);r.textContent=e,r.classList.remove(`hidden`),r.classList.add(`flex`),i.classList.remove(`hidden`),i.addEventListener(`click`,()=>{t&&t()})}return r(),n}async function je(e){let t=e.querySelector(`#vm-ip`),n=e.querySelector(`#vm-punto`),r=e.querySelector(`#vm-estado`),i=e.querySelector(`#btn-refresh`);if(t&&n&&r){i.disabled=!0,r.textContent=`Consultando…`,r.className=`text-vm-fgSecondary`,n.className=`w-1.5 h-1.5 rounded-full bg-vm-muted`;try{let e=await P.sesion.obtener(),i=I(e.estado);t.textContent=e.ip||`—`,n.className=`w-1.5 h-1.5 rounded-full ${i.punto}`,r.textContent=i.texto,r.className=e.estado===`running`?`text-vm-fg`:`text-vm-fgSecondary`,r.title=`Conector: ${e.conector}`}catch(e){t.textContent=`—`,n.className=`w-1.5 h-1.5 rounded-full bg-vm-accent`,r.textContent=`Sin sesión`,r.className=`text-vm-accent`,r.title=R(e)}finally{i.disabled=!1}}}var Me=[{id:`inicio`,etiqueta:`Inicio`,icono:`home`},{id:`paquetes`,etiqueta:`Paquetes`,icono:`package`},{id:`llaves`,etiqueta:`Llaves SSH`,icono:`key-round`},{id:`issue`,etiqueta:`/etc/issue`,icono:`file-text`},{id:`hostname`,etiqueta:`Hostname`,icono:`server`},{id:`usuarios`,etiqueta:`Usuarios`,icono:`users`},{id:`systemd`,etiqueta:`systemd`,icono:`activity`},{id:`haproxy`,etiqueta:`HAProxy`,icono:`git-merge`},{id:`historial`,etiqueta:`Historial`,icono:`history`}];function Ne(e=`inicio`){let t=document.createElement(`nav`);t.className=`bg-vm-surface border-b border-vm-border flex items-center gap-0.5 px-2`,t.setAttribute(`role`,`tablist`),t.setAttribute(`aria-label`,`Módulos de la consola`),t.innerHTML=Me.map(t=>`
    <button type="button" role="tab" class="tab ${t.id===e?`active`:``}" data-tab="${t.id}"
      aria-selected="${t.id===e}" tabindex="${t.id===e?0:-1}">
      <i data-lucide="${t.icono}"></i>
      <span>${t.etiqueta}</span>
    </button>
  `).join(``),j(t);let n=[...t.querySelectorAll(`.tab`)],r=e=>{window.location.hash=`#/${e}`},i=e=>{let t=n[(e+n.length)%n.length];n.forEach(e=>{e.tabIndex=e===t?0:-1}),t.focus(),r(t.dataset.tab)};return n.forEach((e,t)=>{e.addEventListener(`click`,()=>r(e.dataset.tab)),e.addEventListener(`keydown`,e=>{switch(e.key){case`ArrowRight`:e.preventDefault(),i(t+1);break;case`ArrowLeft`:e.preventDefault(),i(t-1);break;case`Home`:e.preventDefault(),i(0);break;case`End`:e.preventDefault(),i(n.length-1)}})}),t}var Pe=`vmcons:resultados`,Fe=[];function Ie(){try{return window.localStorage}catch{return null}}function Le(e){return!!e&&typeof e==`object`&&typeof e.mensaje==`string`}function Re(){let e=Ie();if(!e)return[...Fe];try{let t=JSON.parse(e.getItem(Pe)??`[]`);return Array.isArray(t)?t.filter(Le):[]}catch{return[]}}function ze(e){return Ve([...Re(),e].slice(-50))}function Be(){return Ve([])}function Ve(e){Fe=e;let t=Ie();if(t)try{t.setItem(Pe,JSON.stringify(e))}catch{}return e}var He=`bg-[#2B2B2B] border-[#3C3C3C]`,H=`bg-[#333333] border-[#404040]`,U=`text-[#E5E5E5]`,W=`text-[#999999]`,G=`bg-[#242424] border-[#404040] text-[#D4D4D4] scroll-oscuro`;function Ue(){let e=document.createElement(`section`);return e.className=`flex flex-col min-h-[280px] max-h-[calc(100vh-8rem)] overflow-hidden border rounded-[3px] sticky top-0 ${He}`,e.innerHTML=`
    <header class="px-3 py-1.5 border-b flex items-center justify-between ${H}">
      <div class="flex items-center gap-1.5">
        <i data-lucide="info" class="w-4 h-4 ${U}"></i>
        <span class="text-[12px] font-medium ${U}">Resultado</span>
        <span id="result-cuenta" class="text-[10px] text-[#888888]"></span>
      </div>
      <button id="result-limpiar" type="button"
        class="h-vm-btn px-1.5 inline-flex items-center gap-1 rounded-[3px] border border-[#4A4A4A] bg-[#3A3A3A] text-[#E5E5E5] text-[11px] transition-colors duration-100 hover:border-vm-accent hover:bg-[#454545]"
        title="Borrar el historial de resultados">
        <i data-lucide="trash-2" class="w-4 h-4"></i>
        <span>Limpiar</span>
      </button>
    </header>
    <div id="result-body" class="flex-1 min-h-0 p-2 overflow-auto flex flex-col gap-2 scroll-oscuro ${He}"></div>
    <footer class="px-3 py-1 border-t text-[10px] font-mono text-[#888888] ${H}">
      <span id="result-pie">Sin operaciones ejecutadas</span>
    </footer>
  `,j(e),e.querySelector(`#result-limpiar`).addEventListener(`click`,()=>{Be(),q(e)}),q(e),e}function K({ok:e=!0,mensaje:t=``,operacion:n=``,codigo:r=0,salida:i=``,stderr:a=``,tecnico:o=``}={}){ze({ok:e,mensaje:t||Xe(e,n),operacion:n,codigo:r,salida:i,stderr:a,tecnico:o,hora:new Date().toISOString()}),q()}function q(e=document){let t=e.querySelector(`#result-body`),n=e.querySelector(`#result-cuenta`),r=e.querySelector(`#result-pie`);if(!t)return;let i=Re();if(n&&(n.textContent=i.length>0?`(${i.length})`:``),i.length===0){t.innerHTML=`
      <div class="flex-1 shrink-0 flex items-center justify-center text-center">
        <div>
          <div class="text-[12px] text-[#AAAAAA]">Sin operaciones ejecutadas aún</div>
          <div class="text-[11px] text-[#777777] mt-1">Los resultados se conservan al cambiar de módulo.</div>
        </div>
      </div>
    `,r&&(r.textContent=`Sin operaciones ejecutadas`);return}t.innerHTML=i.map((e,t)=>Ye(e,t===i.length-1)).join(``),j(t),We(t);let a=i[i.length-1];r&&(r.textContent=`${Ze(a.hora)} · código de salida: ${a.codigo??`-`}`)}function We(e){let t=()=>{e.scrollTop=e.scrollHeight};t(),typeof requestAnimationFrame==`function`&&requestAnimationFrame(t)}function Ge(e){return/(^|\n)\s*(Loaded|Active):\s/.test(e)}function Ke(e){return e.split(`
`).map(e=>{if(e.trim()===``)return`<div class="h-2"></div>`;let t=e.match(/^(\s*)●\s+(.*)$/);if(t)return`<div class="whitespace-pre-wrap">${t[1]}<span class="text-vm-accent">●</span> <span class="font-medium text-[#E8E8E8]">${F(t[2])}</span></div>`;let n=e.match(/^(\s*)([A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+([^:]+):\s*(.*)$/);if(n)return`<div class="whitespace-pre-wrap">${n[1]}<span class="text-vm-fgSecondary">${F(n[2])}</span> <span class="text-vm-fgSecondary">${F(n[3])}</span> <span class="text-vm-fgSecondary">${F(n[4])}:</span> ${qe(n[5])}</div>`;let r=e.match(/^(\s*)([A-Z][A-Za-z0-9 ]*?):(.*)$/);return r?`<div class="whitespace-pre-wrap">${r[1]}<span class="text-vm-fgSecondary">${F(r[2])}:</span>${qe(r[3])}</div>`:`<div class="whitespace-pre-wrap">${qe(e)}</div>`}).join(``)}function qe(e){let t={active:`text-vm-success`,running:`text-vm-success`,loaded:`text-vm-success`,enabled:`text-vm-success`,listening:`text-vm-success`,plugged:`text-vm-success`,mounted:`text-vm-success`,exited:`text-vm-success`,inactive:`text-vm-warning`,dead:`text-vm-warning`,disabled:`text-vm-warning`,static:`text-vm-warning`,waiting:`text-vm-warning`,skipped:`text-vm-warning`,activating:`text-vm-warning`,deactivating:`text-vm-warning`,reloading:`text-vm-warning`,failed:`text-vm-accent`,error:`text-vm-accent`};return e.split(/(\b(?:active|running|loaded|enabled|listening|plugged|mounted|exited|inactive|dead|disabled|static|waiting|skipped|activating|deactivating|reloading|failed|error)\b)/gi).map(e=>{let n=t[e.toLowerCase()];return n?`<span class="${n}">${F(e)}</span>`:F(e)}).join(``)}function Je(e){return Ge(e)?`<div class="salida-systemctl border p-2 overflow-auto text-[11px] font-mono max-h-[180px] whitespace-pre-wrap ${G}">${Ke(e)}</div>`:`<pre class="border p-2 overflow-auto text-[11px] font-mono max-h-[120px] whitespace-pre-wrap ${G}">${F(e)}</pre>`}function Ye(e,t){let n=e.ok!==!1;return`
    <article class="border rounded-[3px] overflow-hidden shrink-0 ${n?H:`bg-[#3A2A28] border-vm-accent`} fade-slide">
      <div class="flex items-start gap-2 px-2 py-1.5 ${n?``:`bg-[#3A2A28]`}">
        <i data-lucide="${n?`check-circle-2`:`x-circle`}" class="w-4 h-4 shrink-0 ${n?`text-vm-success`:`text-vm-accent`}"></i>
        <div class="flex-1 min-w-0">
          <div class="flex items-start justify-between gap-2">
            <span class="text-[12px] break-words ${U}">${F(e.mensaje)}</span>
            <span class="text-[10px] font-mono text-[#888888] shrink-0">${Ze(e.hora)}</span>
          </div>
        </div>
      </div>
      <details class="border-t border-[#404040]" ${t?`open`:``}>
        <summary class="px-2 py-1 cursor-pointer text-[11px] ${W}">Detalle técnico</summary>
        <div class="px-2 py-2 flex flex-col gap-1 text-[11px] ${U}">
          <div><span class="${W}">Orden ejecutada:</span> <span class="font-mono break-all">${F(e.operacion)||`-`}</span></div>
          <div><span class="${W}">Código de salida:</span> <span class="font-mono">${F(e.codigo)}</span></div>
          ${e.salida?`<div><span class="${W}">Salida:</span></div>${Je(e.salida)}`:``}
          ${e.stderr?`<div><span class="${W}">stderr:</span></div>
                 <pre class="border p-2 overflow-auto text-[11px] font-mono max-h-[120px] whitespace-pre-wrap ${G}">${F(e.stderr)}</pre>`:``}
          ${!e.salida&&!e.stderr?`<div class="${W}">La orden no produjo salida.</div>`:``}
          ${e.tecnico?`<div><span class="${W}">Diagnóstico:</span> <span class="font-mono break-all">${F(e.tecnico)}</span></div>`:``}
        </div>
      </details>
    </article>
  `}function Xe(e,t){let n=t?` «${t}»`:``;return e?`Se completó la orden${n}.`:`Falló la orden${n}.`}function Ze(e){if(!e)return``;let t=new Date(e);return Number.isNaN(t.getTime())?``:t.toLocaleTimeString(`es-CO`,{hour12:!1})}var Qe=/^[a-z0-9][a-z0-9+.-]*$/,$e=/^[a-z_][a-z0-9_-]{0,31}$/,et=/^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/,tt=/^[A-Za-z0-9_.@-]+\.service$/,nt=/^[0-7]{3,4}$/;function rt(e){return typeof e!=`string`||!Qe.test(e)?`El nombre del paquete solo admite minúsculas, dígitos y los signos + . -`:null}function it(e){return typeof e!=`string`||!$e.test(e)?`El nombre de usuario debe iniciar con minúscula o guion bajo y usar hasta 32 caracteres`:null}function at(e){return typeof e!=`string`||!et.test(e)?`El nombre de host debe cumplir RFC 1123: minúsculas, dígitos y guiones internos`:null}function ot(e){return typeof e!=`string`||!tt.test(e)?`La unidad debe tener una extensión .service y usar letras, dígitos o _ . @ -`:null}function st(e){if(typeof e!=`string`)return!1;let t=e.split(`.`);return t.length===4&&t.every(e=>/^\d{1,3}$/.test(e)&&Number(e)<=255)}function ct(e){return typeof e==`number`&&Number.isInteger(e)&&e>=1&&e<=65535}function lt(e){return typeof e==`string`&&nt.test(e)}function ut(e){return typeof e==`string`&&e.startsWith(`/`)&&e.length>1}function dt(e,t){return`${e}:${t}`}var ft=/^[a-z_][a-z0-9_-]{2,31}$/;function pt(e){return typeof e!=`string`||!ft.test(e)?`El usuario debe tener entre 3 y 32 caracteres y empezar por minúscula o guion bajo`:null}function mt(e){return typeof e!=`string`||e.length<8||e.length>128?`La contraseña debe tener entre 8 y 128 caracteres`:e.includes(`\0`)?`La contraseña no puede contener el carácter NUL`:null}function ht({configurada:e=!1,alEntrar:t}={}){let n=document.createElement(`div`);n.className=`min-h-dvh flex items-center justify-center bg-vm-bg px-4`,n.innerHTML=`
    <div class="w-full max-w-[380px] card p-0">
      <header class="px-3 py-2 border-b border-vm-border bg-[#f7f7f7] flex items-center gap-2">
        <span class="w-2.5 h-2.5 rounded-full bg-vm-accent"></span>
        <span class="font-semibold text-[15px] tracking-tight">vmcons</span>
      </header>
      <form id="acceso-form" class="p-3 flex flex-col gap-2.5" novalidate>
        <div class="flex flex-col gap-0.5">
          <h1 class="text-[14px] font-medium">${e?`Iniciar sesión`:`Crear la cuenta de la consola`}</h1>
          <p class="text-[11px] text-vm-fgSecondary">${e?`La consola pide usuario y contraseña para administrar la máquina virtual.`:`Es la primera vez que se abre la consola: define la cuenta con la que se administrará.`}</p>
        </div>
        <div class="flex flex-col gap-1">
          <label class="label" for="acceso-usuario">Usuario</label>
          <input id="acceso-usuario" name="usuario" class="input font-mono" autocomplete="username"
            spellcheck="false" maxlength="32">
        </div>
        <div class="flex flex-col gap-1">
          <label class="label" for="acceso-password">Contraseña</label>
          <input id="acceso-password" name="password" type="password" class="input"
            autocomplete="${e?`current-password`:`new-password`}" maxlength="128">
        </div>
        ${e?``:`<div class="flex flex-col gap-1">
          <label class="label" for="acceso-repetir">Repetir la contraseña</label>
          <input id="acceso-repetir" name="repetir" type="password" class="input"
            autocomplete="new-password" maxlength="128">
        </div>`}
        <p id="acceso-aviso" class="text-[11px] text-vm-accent min-h-[14px]" role="alert"></p>
        <button id="acceso-enviar" type="submit" class="btn btn-primary justify-center">
          ${e?`Entrar`:`Crear cuenta y entrar`}
        </button>
      </form>
    </div>
  `,j(n);let r=n.querySelector(`#acceso-form`),i=n.querySelector(`#acceso-aviso`),a=n.querySelector(`#acceso-enviar`),o=e=>{i.textContent=e};return r.addEventListener(`submit`,async n=>{n.preventDefault();let i=new FormData(r),s=String(i.get(`usuario`)??``).trim(),c=String(i.get(`password`)??``),l=String(i.get(`repetir`)??``),u=pt(s);if(u){o(u);return}let d=mt(c);if(d)o(d);else if(!e&&c!==l)o(`Las contraseñas no coinciden.`);else{o(``),a.disabled=!0;try{let n=e?await P.auth.entrar(s,c):await P.auth.configurar(s,c);t&&t(n)}catch(e){o(R(e))}finally{a.disabled=!1}}}),n}function gt({maquina:e,alRegistrar:t}){let n=document.createElement(`div`);n.className=`fixed inset-0 z-50 bg-black/30 flex items-center justify-center p-4`,n.innerHTML=`
    <section class="card w-[400px] max-w-full" role="dialog" aria-modal="true" aria-labelledby="registro-titulo">
      <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
        <span id="registro-titulo" class="text-[12px] font-medium">Registrar ${F(e.nombre)}</span>
        <button id="registro-cerrar" type="button" class="btn h-vm-btn px-1" title="Cerrar" aria-label="Cerrar">
          <i data-lucide="x" class="w-4 h-4"></i>
        </button>
      </header>
      <form id="registro-form" class="p-3 flex flex-col gap-2">
        <p class="text-[11px] text-vm-fgSecondary">
          La máquina aún no está registrada. Ingresa un usuario con permisos para preparar SSH; las
          credenciales se guardan solo en este equipo.
        </p>
        <label class="flex flex-col gap-1">
          <span class="label">Usuario</span>
          <input name="usuario" value="root" class="input font-mono" autocomplete="username" required>
        </label>
        <label class="flex flex-col gap-1">
          <span class="label">Contraseña</span>
          <input name="password" type="password" class="input" autocomplete="current-password" required>
        </label>
        <div id="registro-progreso" class="hidden flex-col gap-1.5 pt-1" aria-live="polite">
          <div class="flex items-center justify-between text-[11px]">
            <span class="flex items-center gap-1.5">
              <i data-lucide="loader-2" class="w-3.5 h-3.5 text-vm-accent animate-spin"></i>
              <span id="registro-paso">Guardando credenciales…</span>
            </span>
            <span id="registro-porcentaje" class="font-mono text-vm-fgSecondary">0 %</span>
          </div>
          <div class="h-1.5 w-full bg-vm-bg border border-vm-border overflow-hidden">
            <div id="registro-barra" class="h-full bg-vm-accent transition-all duration-300" style="width: 0%"></div>
          </div>
          <div id="registro-ayuda" class="text-[10px] text-vm-fgSecondary">
            La consola está conectando con la máquina y preparando SSH. Esto puede tardar unos segundos.
          </div>
        </div>
        <div id="registro-error" class="text-[11px] text-vm-accent min-h-[14px]"></div>
        <footer class="flex justify-end gap-2 pt-1">
          <button id="registro-cancelar" type="button" class="btn">Cancelar</button>
          <button type="submit" class="btn btn-primary"><i data-lucide="key-round"></i> Registrar</button>
        </footer>
      </form>
    </section>
  `,j(n);let r=()=>n.remove(),i=n.querySelector(`#registro-form`),a=n.querySelector(`#registro-error`),o=i.querySelector(`button[type=submit]`),s=n.querySelector(`#registro-progreso`),c=n.querySelector(`#registro-barra`),l=n.querySelector(`#registro-porcentaje`),u=n.querySelector(`#registro-paso`),d=n.querySelector(`#registro-ayuda`),f=n.querySelector(`#registro-cerrar`),p=n.querySelector(`#registro-cancelar`),m=!1,h=0,g=()=>{m||r()},_=e=>{m=e;for(let t of i.querySelectorAll(`input, button`))t.disabled=e;f.disabled=e,p.disabled=e,o.innerHTML=e?`Registrando…`:`<i data-lucide="key-round"></i> Registrar`,e||j(o)},v=()=>{s.classList.remove(`hidden`),s.classList.add(`flex`);let e=6,t=()=>{c.style.width=`${e}%`,l.textContent=`${e} %`,e>=82?u.textContent=`Verificando el acceso SSH…`:e>=54?u.textContent=`Preparando SSH y la llave de root…`:e>=26&&(u.textContent=`Conectando con Guest Additions…`)};t(),h=setInterval(()=>{e=Math.min(e+5,88),t()},600)},y=(e,t)=>{clearInterval(h),c.style.width=`100%`,l.textContent=`100 %`,c.className=`h-full transition-all duration-300 ${e?`bg-vm-success`:`bg-vm-accent`}`,u.textContent=t,d.textContent=e?`La máquina quedó registrada.`:`Revisa el detalle técnico e inténtalo de nuevo.`};f.addEventListener(`click`,g),p.addEventListener(`click`,g),n.addEventListener(`click`,e=>{e.target===n&&g()}),n.addEventListener(`keydown`,e=>{e.key===`Escape`&&g()}),i.addEventListener(`submit`,async e=>{e.preventDefault(),a.textContent=``;let n=new FormData(i);_(!0),v();try{await P.sesion.registrar(String(n.get(`usuario`)||``),String(n.get(`password`)||``)),y(!0,`Máquina registrada.`),setTimeout(()=>{r(),t&&t()},650)}catch(e){if(y(!1,`No fue posible registrar la máquina.`),a.textContent=R(e),e.tecnico){let t=document.createElement(`pre`);t.className=`mt-1 p-1.5 bg-vm-bg border border-vm-border text-[10px] font-mono whitespace-pre-wrap max-h-[120px] overflow-auto`,t.textContent=e.tecnico,a.appendChild(t)}if(e.solicitud){let t=document.createElement(`div`);t.className=`mt-1 text-[10px] font-mono text-vm-fgSecondary`,t.textContent=`Solicitud: ${e.solicitud}`,a.appendChild(t)}_(!1)}}),document.body.appendChild(n),i.querySelector(`input[name=usuario]`).focus()}var _t=[[`package`,`Paquetes`,`Gestión de paquetes APT/dpkg`,`#/paquetes`],[`key-round`,`Llaves SSH`,`Gestión de authorized_keys`,`#/llaves`],[`file-text`,`/etc/issue`,`Banner pre-login del sistema`,`#/issue`],[`server`,`Hostname`,`Identificador y FQDN del host`,`#/hostname`],[`users`,`Usuarios`,`Cuentas locales y privilegios`,`#/usuarios`],[`activity`,`systemd`,`Servicios, timers y unidades`,`#/systemd`],[`git-merge`,`HAProxy`,`Balanceo y configuración L4/L7`,`#/haproxy`],[`history`,`Historial`,`Registro de auditoría y tareas`,`#/historial`]];function vt(){let e=document.createElement(`div`);return e.className=`grid grid-cols-[58%_42%] gap-vm-gap`,e.innerHTML=`
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="monitor" class="w-4 h-4"></i><span class="text-[12px] font-medium">Sesión con la máquina virtual</span></div>
          <button id="sesion-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
        </header>
        <div id="sesion-cuerpo" class="p-3 flex flex-col gap-1.5 text-[12px]">
          <div class="text-vm-fgSecondary">Consultando la sesión…</div>
        </div>
      </section>
      <p class="text-[11px] text-vm-fgSecondary px-1">
        Se muestran las máquinas registradas, las que publican Guest Additions y las apagadas para poder encenderlas. Una encendida sin Guest Additions aparece con un aviso y puedes ocultarla o mantenerla en la lista.
      </p>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="layout-grid" class="w-4 h-4"></i><span class="text-[12px] font-medium">Módulos</span></div></header>
        <div class="p-3 grid grid-cols-2 gap-2">
          ${_t.map(([e,t,n,r])=>`
            <a href="${r}" class="border border-vm-border p-2 rounded-sm hover:border-[#C0392B] hover:bg-[#fdf5f4] transition-colors duration-100">
              <div class="flex items-center gap-1.5"><i data-lucide="${e}"></i><span class="text-[12px] font-medium">${t}</span></div>
              <div class="text-[11px] text-vm-fgSecondary mt-0.5">${n}</div>
            </a>
          `).join(``)}
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `,j(e),e.querySelector(`#sesion-refresh`).addEventListener(`click`,()=>J(e)),J(e),e}async function yt(e){try{let{maquinas:t}=await P.maquinas.listar(),n=t.find(e=>e.activa);if(!n){await J(e);return}gt({maquina:n,alRegistrar:()=>J(e)})}catch{await J(e)}}async function J(e){let t=e.querySelector(`#sesion-cuerpo`),n=e.querySelector(`#sesion-refresh`);if(t){n.disabled=!0,t.innerHTML=`
    <div class="text-vm-fgSecondary">Consultando la sesión con la máquina…</div>
    <div class="h-1 w-full bg-vm-bg border border-vm-border overflow-hidden mt-1">
      <div class="barra-indeterminada h-full bg-vm-accent"></div>
    </div>
  `;try{let n=await P.sesion.obtener(),r=I(n.estado),i=new Date().toLocaleTimeString(`es-CO`,{hour12:!1});t.innerHTML=`
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Máquina</span>
        <span class="font-mono">${F(n.vm)||`sin vincular`}</span>
      </div>
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Estado</span>
        <span class="px-1.5 py-0.5 border text-[11px] rounded-sm ${r.clase}">${r.texto}</span>
      </div>
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Conector</span>
        <span class="flex items-center gap-1"><i data-lucide="key-round" class="w-4 h-4"></i>${F(n.conector)}</span>
      </div>
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Dirección IP</span>
        <span class="flex items-center gap-1.5">
          <span class="font-mono">${F(n.ip)}</span>
          <button id="copiar-ip" class="btn h-vm-btn px-1" title="Copiar la dirección"><i data-lucide="copy" class="w-4 h-4"></i></button>
        </span>
      </div>
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Última consulta</span>
        <span class="text-vm-fgSecondary font-mono">${i}</span>
      </div>
      ${n.registrada?``:`
        <button id="registrar-sesion" type="button" class="btn btn-primary self-start mt-1">
          <i data-lucide="key-round"></i> Registrar esta máquina
        </button>
      `}
    `,j(t),t.querySelector(`#copiar-ip`).addEventListener(`click`,()=>navigator.clipboard?.writeText(n.ip)),t.querySelector(`#registrar-sesion`)?.addEventListener(`click`,()=>yt(e))}catch(n){t.innerHTML=`
      <div class="flex items-start gap-2 text-[12px]">
        <i data-lucide="alert-triangle" class="w-4 h-4 text-vm-accent shrink-0"></i>
        <div>
          <div class="text-vm-fg">No fue posible consultar la sesión</div>
          <div class="text-[11px] text-vm-fgSecondary mt-0.5">${F(R(n))}</div>
        </div>
      </div>
      <button id="abrir-registro" type="button" class="btn btn-primary self-start">
        <i data-lucide="key-round"></i> Registrar esta máquina
      </button>
    `,j(t),t.querySelector(`#abrir-registro`)?.addEventListener(`click`,()=>yt(e))}finally{n.disabled=!1}}}function bt(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap`,e.innerHTML=`
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="package" class="w-4 h-4"></i><span class="text-[12px] font-medium">Instalación y eliminación de paquetes</span></div></header>
        <form id="form-paquetes" class="p-3 flex flex-col gap-2" novalidate>
          <div class="flex gap-4 text-[12px]">
            <label class="flex items-center gap-1.5"><input type="radio" name="accion" value="instalar" checked> Instalar</label>
            <label class="flex items-center gap-1.5"><input type="radio" name="accion" value="eliminar"> Eliminar</label>
          </div>
          <div class="flex flex-col gap-1">
            <label class="label" for="paquete-nombre">Nombre del paquete</label>
            <input id="paquete-nombre" name="nombre" class="input font-mono" placeholder="nano" autocomplete="off" spellcheck="false" required>
            <span id="paquete-error" class="text-[11px] text-vm-accent hidden"></span>
          </div>
          <div class="flex gap-2">
            <button type="submit" class="btn btn-primary"><i data-lucide="play"></i> Ejecutar</button>
            <button type="reset" class="btn">Limpiar</button>
          </div>
        </form>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="list" class="w-4 h-4"></i><span class="text-[12px] font-medium">Paquetes instalados</span></div>
          <div class="flex items-center gap-1.5">
            <span id="paquetes-cuenta" class="text-[11px] text-vm-fgSecondary"></span>
            <button id="paquetes-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
          </div>
        </header>
        <div class="p-2 border-b border-vm-border">
          <div class="flex items-center gap-1 border border-vm-border px-2 h-vm-input">
            <i data-lucide="search"></i>
            <input id="paquetes-filtro" class="flex-1 outline-none text-[12px] bg-transparent" placeholder="Filtrar por nombre…">
          </div>
        </div>
        <div class="overflow-auto max-h-[320px]">
          <table class="w-full text-[12px]">
            <thead class="bg-vm-bg sticky top-0">
              <tr>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Nombre</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Versión</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Estado</th>
              </tr>
            </thead>
            <tbody id="paquetes-filas"></tbody>
          </table>
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `,j(e);let t=[],n=e.querySelector(`#paquetes-filas`),r=e.querySelector(`#paquetes-cuenta`),i=e.querySelector(`#paquetes-filtro`),a=()=>{let e=i.value.trim().toLowerCase(),a=e?t.filter(t=>t.nombre.toLowerCase().includes(e)):t;r.textContent=`${a.length} de ${t.length}`,a.length===0?n.innerHTML=z(3,t.length===0?`No hay paquetes para mostrar`:`Ningún paquete coincide con el filtro`):n.innerHTML=a.map(e=>`
        <tr class="border-t border-vm-border hover:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${F(e.nombre)}</td>
          <td class="px-3 py-1.5 font-mono text-vm-fgSecondary">${F(e.version)||`—`}</td>
          <td class="px-3 py-1.5">
            <span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${e.estado===`installed`?`text-vm-success border-vm-success bg-[#e8f5e9]`:`text-vm-fgSecondary border-vm-border bg-vm-bg`}">${F(e.estado)}</span>
          </td>
        </tr>
      `).join(``)},o=async()=>{n.innerHTML=z(3,`Consultando los paquetes…`);try{t=(await P.paquetes.listar()).paquetes??[],a()}catch(e){t=[],r.textContent=``,n.innerHTML=z(3,R(e),`text-vm-accent`)}};i.addEventListener(`input`,a),e.querySelector(`#paquetes-refresh`).addEventListener(`click`,o);let s=e.querySelector(`#form-paquetes`),c=e.querySelector(`#paquete-nombre`),l=e.querySelector(`#paquete-error`);return s.addEventListener(`submit`,async e=>{e.preventDefault();let t=s.elements.accion.value,n=c.value.trim(),r=rt(n);r?(l.textContent=r,l.classList.remove(`hidden`),c.focus()):(l.classList.add(`hidden`),B(s,!0),await V({accion:()=>t===`instalar`?P.paquetes.instalar(n):P.paquetes.eliminar(n),operacion:t===`instalar`?`apt-get install -y ${n}`:`apt-get remove -y ${n}`,informar:e=>{t===`eliminar`&&e.ok&&(e.mensaje=e.mensaje||`Se eliminó el paquete «${n}».`),K(e)},recargar:o}),B(s,!1))}),s.addEventListener(`reset`,()=>{l.classList.add(`hidden`),setTimeout(()=>c.focus(),0)}),o(),e}var xt=`id_ed25519_vmcons`;function St(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap`,e.innerHTML=`
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="key-round" class="w-4 h-4"></i><span id="llave-titulo" class="text-[12px] font-medium">Par de llaves SSH</span></div></header>
        <div class="p-3 flex flex-col gap-3">
          <div class="flex items-center gap-2">
            <label class="label shrink-0" for="llave-usuario">Usuario</label>
            <select id="llave-usuario" class="input flex-1"></select>
          </div>
          <span id="llave-usuarios-error" class="text-[11px] text-vm-accent hidden"></span>
          <div class="flex justify-between items-center text-[12px]">
            <span class="text-vm-fgSecondary">Estado</span>
            <span id="llave-estado" class="px-1.5 py-0.5 border text-[11px] rounded-sm text-vm-fgSecondary border-vm-border bg-vm-bg">sin verificar</span>
          </div>
          <p id="llave-descripcion" class="text-[11px] text-vm-fgSecondary"></p>
          <div class="flex gap-2">
            <button id="llave-generar" class="btn btn-primary"><i data-lucide="settings"></i> Generar par de llaves</button>
            <button id="llave-descargar" class="btn"><i data-lucide="download"></i> Descargar llave privada</button>
          </div>
          <div id="llave-confirmacion" class="hidden flex-col gap-2 border border-vm-accent bg-[#fdf1ef] p-2 text-[11px]">
            <span id="llave-confirmacion-texto"></span>
            <div class="flex gap-2 justify-end">
              <button id="llave-cancelar-regenerar" type="button" class="btn">Cancelar</button>
              <button id="llave-confirmar-regenerar" type="button" class="btn btn-primary">Regenerar</button>
            </div>
          </div>
        </div>
      </section>
      <section class="card border border-vm-accent">
        <div class="p-3 flex gap-2">
          <i data-lucide="alert-triangle" class="text-vm-accent shrink-0"></i>
          <div id="llave-aviso" class="text-[12px]"></div>
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `,j(e);let t=e.querySelector(`#llave-titulo`),n=e.querySelector(`#llave-usuario`),r=e.querySelector(`#llave-usuarios-error`),i=e.querySelector(`#llave-estado`),a=e.querySelector(`#llave-descripcion`),o=e.querySelector(`#llave-aviso`),s=e.querySelector(`#llave-generar`),c=e.querySelector(`#llave-descargar`),l=e.querySelector(`#llave-confirmacion`),u=e.querySelector(`#llave-confirmacion-texto`),d=e.querySelector(`#llave-confirmar-regenerar`),f=e.querySelector(`#llave-cancelar-regenerar`),p=null,m=[],h=!1,g=(e,t)=>{i.textContent=e,i.className=`px-1.5 py-0.5 border text-[11px] rounded-sm `+(t===`ok`?`text-vm-success border-vm-success bg-[#e8f5e9]`:t===`error`?`text-vm-accent border-vm-accent bg-[#fdf1ef]`:`text-vm-fgSecondary border-vm-border bg-vm-bg`)},_=()=>{let e=n.value;n.innerHTML=`<option value="root">root (conexión)</option>`+m.map(e=>`<option value="${F(e.nombre)}">${F(e.nombre)}</option>`).join(``),(e===`root`||m.some(t=>t.nombre===e))&&(n.value=e)},v=()=>{let e=n.value||`root`,r=m.find(t=>t.nombre===e),i=e===`root`,l=i?`/root/.ssh/authorized_keys`:`/home/${e}/.ssh/authorized_keys`;if(t.textContent=i?`Par de llaves SSH para root (conexión)`:`Par de llaves SSH para ${e}`,a.innerHTML=`La generación crea el par en el anfitrión, instala la llave pública en <span class="font-mono">${F(l)}</span> y deshabilita el acceso SSH por contraseña; el login local sigue disponible.`,o.textContent=i?`Guarda la llave privada en un lugar seguro y no la compartas; quien la posea tendrá acceso root a la máquina.`:`Guarda la llave privada en un lugar seguro y no la compartas; quien la posea tendrá acceso a la cuenta «${e}».`,i){g(h?`activa`:`sin llave`,h?`ok`:`neutro`),s.disabled=h,s.title=h?`La llave de root es la llave de conexión compartida; no se regenera desde este módulo.`:``,s.innerHTML=h?`<i data-lucide="lock"></i> La llave de conexión ya existe`:`<i data-lucide="settings"></i> Generar par de llaves`,c.disabled=!h,j(s);return}let u=!!r?.llaves;g(u?`activa`:`sin llave`,u?`ok`:`neutro`),s.disabled=!1,s.title=``,s.innerHTML=`<i data-lucide="${u?`refresh-cw`:`settings`}"></i> ${u?`Regenerar par de llaves`:`Generar par de llaves`}`,c.disabled=!u,j(s)},y=async()=>{r.classList.add(`hidden`);try{m=(await P.usuarios.listar()).usuarios??[]}catch{m=[],r.textContent=`No se pudieron listar los usuarios; solo está disponible root.`,r.classList.remove(`hidden`)}try{h=!!(await P.llaves.estadoRoot())?.existe}catch{h=!1}_(),v()},b=()=>{l.classList.add(`hidden`),l.classList.remove(`flex`),p=null},x=async(t,n)=>{let r=m.find(e=>e.nombre===t);B(e,!0);let i=await V({accion:()=>t===`root`?P.llaves.generarRoot():P.usuarios.generarLlave(t,{regenerar:n}),operacion:t===`root`?`ssh-keygen -t ed25519 + instalación en /root/.ssh/authorized_keys`:`ssh-keygen -t ed25519 + instalación en /home/${t}/.ssh/authorized_keys`,informar:K});B(e,!1),i&&(r&&(r.llaves=!0),t===`root`&&(h=!0)),v()};return n.addEventListener(`change`,()=>{b(),v()}),s.addEventListener(`click`,()=>{let e=n.value||`root`,t=m.find(t=>t.nombre===e);e!==`root`&&t?.llaves?(p=e,u.textContent=`El usuario «${e}» ya tiene llaves. ¿Regenerar el par? La llave anterior dejará de funcionar.`,l.classList.remove(`hidden`),l.classList.add(`flex`)):x(e,!1)}),d.addEventListener(`click`,()=>{let e=p;e&&(b(),x(e,!0))}),f.addEventListener(`click`,b),c.addEventListener(`click`,async()=>{let t=n.value||`root`,r=t===`root`?xt:`id_ed25519_${t}`;B(e,!0);let i=await V({accion:async()=>(Te(t===`root`?await P.llaves.descargarRoot():await P.usuarios.descargarLlave(t),r),{mensaje:`Se descargó la llave privada de «${t}» como «${r}».`,operacion:t===`root`?`GET /api/keys/root`:`GET /api/users/${t}/key`,salida:`Archivo descargado: ${r}`}),operacion:t===`root`?`GET /api/keys/root`:`GET /api/users/${t}/key`,informar:K});B(e,!1),g(i?`activa`:`sin llave`,i?`ok`:`error`)}),y(),e}var Y=[[`nombre_host`,`Mostrar nombre del host`,`Hostname: \\n`],[`fecha`,`Mostrar fecha`,`Fecha: \\d`],[`hora`,`Mostrar hora`,`Hora: \\t`],[`ip`,`Mostrar dirección IP`,`IP: \\4`]];function Ct(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap`,e.innerHTML=`
    <section class="card">
      <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
        <div class="flex items-center gap-1.5"><i data-lucide="file-text" class="w-4 h-4"></i><span class="text-[12px] font-medium">Configuración de /etc/issue</span></div>
        <button id="issue-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
      </header>
      <form id="form-issue" class="p-3 flex flex-col gap-2" novalidate>
        <div class="flex flex-col gap-1">
          <label class="label" for="issue-mensaje">Mensaje personalizado</label>
          <textarea id="issue-mensaje" name="mensaje" class="input h-auto py-1 font-mono" rows="2" maxlength="512" placeholder="Debian GNU/Linux 13"></textarea>
          <span class="text-[11px] text-vm-fgSecondary">Hasta 512 caracteres. Se permiten saltos de línea.</span>
        </div>
        <div class="flex flex-col gap-1 text-[12px]">
          ${Y.map(([e,t])=>`<label class="flex items-center gap-1.5"><input type="checkbox" name="${e}"> ${t}</label>`).join(``)}
          <span class="text-[11px] text-vm-fgSecondary">Al marcar una casilla se agrega su bloque al mensaje; puedes editarlo o quitarlo.</span>
        </div>
        <section class="card">
          <header class="px-3 py-1 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="eye" class="w-3 h-3"></i><span class="text-[11px]">Vista previa</span></div></header>
          <pre id="issue-preview" class="p-2 font-mono text-[11px] whitespace-pre-wrap"></pre>
        </section>
        <div class="flex gap-2">
          <button type="submit" class="btn btn-primary"><i data-lucide="save"></i> Guardar</button>
          <button type="button" id="issue-restablecer" class="btn"><i data-lucide="rotate-ccw"></i> Restablecer</button>
        </div>
      </form>
    </section>
    <div id="result-slot"></div>
  `,j(e);let t=e.querySelector(`#form-issue`),n=e.querySelector(`#issue-mensaje`),r=e.querySelector(`#issue-preview`),i={mensaje:``,nombre_host:!1,fecha:!1,hora:!1,ip:!1,contenido:``},a={nombre:``,ip:``},o=(e,t)=>e.replace(/\\([a-zA-Z0-9])/g,(e,n)=>t[n]??e),s=()=>({n:a.nombre||`nombre-host`,d:new Intl.DateTimeFormat(`es-CO`).format(new Date),t:new Date().toLocaleTimeString(`es-CO`,{hour12:!1}),4:a.ip||`192.168.1.10`,l:`tty1`,s:`Linux`,S:`Debian GNU/Linux`}),c=(e,t)=>e.split(`
`).map(e=>e.trim()).includes(t.trim())?e:e.trimEnd()?`${e.trimEnd()}\n${t}`:t,l=(e,t)=>e.split(`
`).filter(e=>e.trim()!==t.trim()).join(`
`),u=()=>{let e=n.value.split(`
`).map(e=>e.trim());for(let[n,,r]of Y)t.elements[n].checked=e.includes(r.trim())},d=()=>({mensaje:n.value,nombre_host:!1,fecha:!1,hora:!1,ip:!1}),f=()=>{let e=d(),t=s(),n=e.mensaje.replace(/\r\n/g,`
`).split(`
`).map(e=>o(e,t));r.textContent=n.some(e=>e.trim())?n.join(`
`):`(banner vacío)`},p=e=>{i={...i,...e};let t=e.mensaje??``;for(let[n,,r]of Y)e[n]&&(t=c(t,r));n.value=t,u(),f()},m=async()=>{B(t,!0),r.textContent=`Consultando…`;try{let[e,t]=await Promise.all([P.issue.obtener(),P.hostname.obtener().catch(()=>null)]);a={nombre:t?.nombre??``,ip:t?.ip??``},p(e)}catch(e){r.textContent=R(e)}finally{B(t,!1)}};return t.addEventListener(`input`,e=>{e.target===n&&u(),f()}),t.addEventListener(`change`,e=>{let t=Y.find(([t])=>t===e.target.name);if(t&&e.target.type===`checkbox`){let[,,r]=t;n.value=e.target.checked?c(n.value,r):l(n.value,r)}u(),f()}),e.querySelector(`#issue-refresh`).addEventListener(`click`,m),e.querySelector(`#issue-restablecer`).addEventListener(`click`,()=>p(i)),t.addEventListener(`submit`,async e=>{e.preventDefault(),B(t,!0);let n=d(),r=await V({accion:async()=>(await P.issue.reemplazar(n),{mensaje:`Se actualizó el banner de /etc/issue.`}),operacion:`PUT /api/issue`,informar:K});B(t,!1),r&&(i={...n,contenido:``},m())}),m(),e}function wt(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap`,e.innerHTML=`
    <section class="card">
      <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
        <div class="flex items-center gap-1.5"><i data-lucide="server" class="w-4 h-4"></i><span class="text-[12px] font-medium">Nombre de host</span></div>
        <button id="hostname-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
      </header>
      <form id="form-hostname" class="p-3 flex flex-col gap-2" novalidate>
        <div class="flex flex-col gap-1">
          <label class="label" for="hostname-nombre">Nombre de host</label>
          <input id="hostname-nombre" name="nombre" class="input font-mono" placeholder="debian13" autocomplete="off" spellcheck="false">
          <span id="hostname-error-nombre" class="text-[11px] text-vm-accent hidden"></span>
          <span class="text-[11px] text-vm-fgSecondary">Minúsculas, dígitos y guiones internos. Máximo 63 caracteres.</span>
        </div>
        <div class="flex flex-col gap-1">
          <label class="label" for="hostname-ip">Dirección IP</label>
          <input id="hostname-ip" name="ip" class="input font-mono" placeholder="192.168.1.105" autocomplete="off" spellcheck="false">
          <span id="hostname-error-ip" class="text-[11px] text-vm-accent hidden"></span>
        </div>
        <label class="flex items-center gap-1.5 text-[12px]"><input type="checkbox" name="actualizar_hosts" checked> Actualizar también <span class="font-mono">/etc/hosts</span></label>
        <div class="flex gap-2">
          <button type="submit" class="btn btn-primary"><i data-lucide="play"></i> Aplicar</button>
          <button type="button" id="hostname-restablecer" class="btn"><i data-lucide="rotate-ccw"></i> Restablecer</button>
        </div>
      </form>
    </section>
    <div id="result-slot"></div>
  `,j(e);let t=e.querySelector(`#form-hostname`),n=e.querySelector(`#hostname-nombre`),r=e.querySelector(`#hostname-ip`),i=e.querySelector(`#hostname-error-nombre`),a=e.querySelector(`#hostname-error-ip`),o={nombre:``,ip:``,actualizar_hosts:!0},s=(e,t)=>{e.textContent=t??``,e.classList.toggle(`hidden`,!t)},c=async()=>{B(t,!0);try{let e=await P.hostname.obtener();o={...o,...e},n.value=e.nombre??``,r.value=e.ip??``,t.elements.actualizar_hosts.checked=e.actualizar_hosts!==!1}catch(e){s(i,R(e))}finally{B(t,!1)}};return t.addEventListener(`input`,()=>{s(i,null),s(a,null)}),e.querySelector(`#hostname-refresh`).addEventListener(`click`,c),e.querySelector(`#hostname-restablecer`).addEventListener(`click`,()=>{n.value=o.nombre??``,r.value=o.ip??``,t.elements.actualizar_hosts.checked=o.actualizar_hosts!==!1,s(i,null),s(a,null)}),t.addEventListener(`submit`,async e=>{e.preventDefault();let o=at(n.value.trim()),l=st(r.value.trim())?null:`La dirección debe ser una IPv4 válida, por ejemplo 192.168.1.105`;if(s(i,o),s(a,l),o){n.focus();return}if(l){r.focus();return}let u={nombre:n.value.trim(),ip:r.value.trim(),actualizar_hosts:t.elements.actualizar_hosts.checked};B(t,!0),await V({accion:async()=>(await P.hostname.reemplazar(u),{mensaje:`Se aplicó el nombre de host «${u.nombre}».`}),operacion:`PUT /api/hostname`,informar:K,recargar:c}),B(t,!1)}),c(),e}function Tt(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap`,e.innerHTML=`
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="user-plus" class="w-4 h-4"></i><span class="text-[12px] font-medium">Crear usuario</span></div></header>
        <form id="form-crear" class="p-3 flex flex-col gap-2" novalidate>
          <div class="flex flex-col gap-1">
            <label class="label" for="usuario-nombre">Nombre de usuario</label>
            <input id="usuario-nombre" name="nombre" class="input font-mono" placeholder="jorge" autocomplete="off" spellcheck="false">
            <span id="usuario-error" class="text-[11px] text-vm-accent hidden"></span>
          </div>
          <div class="flex flex-col gap-1">
            <label class="label" for="usuario-password">Contraseña</label>
            <input id="usuario-password" name="password" type="password" class="input" autocomplete="new-password" placeholder="Opcional">
            <span class="text-[11px] text-vm-fgSecondary">Permite el login local y <span class="font-mono">sudo</span>; el acceso SSH sigue usando la llave.</span>
          </div>
          <label class="flex items-center gap-1.5 text-[12px]"><input type="checkbox" name="sudo"> Otorgar privilegios sudo</label>
          <label class="flex items-center gap-1.5 text-[12px]"><input type="checkbox" name="llaves"> Generar par de llaves e instalar la pública</label>
          <div class="flex gap-2">
            <button type="submit" class="btn btn-primary"><i data-lucide="play"></i> Crear usuario</button>
            <button type="button" id="usuario-descargar-nueva" class="btn hidden"><i data-lucide="download"></i> Descargar llave privada</button>
          </div>
        </form>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="user-minus" class="w-4 h-4"></i><span class="text-[12px] font-medium">Eliminar usuario</span></div></header>
        <form id="form-eliminar" class="p-3 flex flex-col gap-2">
          <div class="flex flex-col gap-1">
            <label class="label" for="eliminar-nombre">Nombre de usuario</label>
            <select id="eliminar-nombre" class="input"></select>
          </div>
          <label class="flex items-center gap-1.5 text-[12px]"><input type="checkbox" name="archivos"> Eliminar también todos sus archivos</label>
          <button type="submit" class="btn btn-danger-outline w-fit"><i data-lucide="trash-2"></i> Eliminar usuario</button>
        </form>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="users" class="w-4 h-4"></i><span class="text-[12px] font-medium">Usuarios de la máquina</span></div>
          <div class="flex items-center gap-1.5">
            <span id="usuarios-cuenta" class="text-[11px] text-vm-fgSecondary"></span>
            <button id="usuarios-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
          </div>
        </header>
        <div class="overflow-auto max-h-[260px]">
          <table class="w-full text-[12px]">
            <thead class="bg-vm-bg">
              <tr>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Usuario</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">sudo</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">llaves</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">UID</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Home</th>
                <th class="px-3 py-1"></th>
              </tr>
            </thead>
            <tbody id="usuarios-filas"></tbody>
          </table>
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `,j(e);let t=[],n=e.querySelector(`#usuarios-filas`),r=e.querySelector(`#usuarios-cuenta`),i=e.querySelector(`#eliminar-nombre`),a=e.querySelector(`#form-crear`),o=e.querySelector(`#form-eliminar`),s=(e,n)=>{let r=e.value;e.innerHTML=`<option value="">${n}</option>`+t.map(e=>`<option value="${F(e.nombre)}">${F(e.nombre)}</option>`).join(``),t.some(e=>e.nombre===r)&&(e.value=r)},c=()=>{r.textContent=String(t.length),s(i,t.length?`Seleccionar usuario`:`Sin usuarios`),t.length===0?n.innerHTML=z(6,`No hay usuarios para mostrar`):(n.innerHTML=t.map(e=>`
        <tr class="border-t border-vm-border hover:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${F(e.nombre)}</td>
          <td class="px-3 py-1.5">${e.sudo?`<i data-lucide="check" class="w-4 h-4 text-vm-success"></i>`:`<span class="text-vm-fgSecondary">—</span>`}</td>
          <td class="px-3 py-1.5">${e.llaves?`<i data-lucide="key-round" class="w-4 h-4 text-vm-fg"></i>`:`<span class="text-vm-fgSecondary">—</span>`}</td>
          <td class="px-3 py-1.5 font-mono text-vm-fgSecondary">${F(e.uid)||`—`}</td>
          <td class="px-3 py-1.5 font-mono text-vm-fgSecondary">${F(e.home)||`—`}</td>
          <td class="px-3 py-1.5 text-right whitespace-nowrap">
            ${e.llaves?`<button class="btn h-vm-btn px-1" data-descargar="${F(e.nombre)}" title="Descargar llave privada"><i data-lucide="download" class="w-4 h-4"></i></button>`:``}
            <button class="btn h-vm-btn px-1" data-eliminar="${F(e.nombre)}" title="Eliminar usuario"><i data-lucide="trash-2" class="w-4 h-4"></i></button>
          </td>
        </tr>
      `).join(``),j(n))},l=async()=>{n.innerHTML=z(6,`Consultando los usuarios…`);try{t=(await P.usuarios.listar()).usuarios??[],c()}catch(e){t=[],r.textContent=``,n.innerHTML=z(6,R(e),`text-vm-accent`)}},u=async e=>{await V({accion:async()=>(Te(await P.usuarios.descargarLlave(e),`id_ed25519_${e}`),{mensaje:`Se descargó la llave privada de «${e}».`,operacion:`GET /api/users/${e}/key`,salida:`Archivo descargado: id_ed25519_${e}`}),operacion:`GET /api/users/${e}/key`,informar:K})},d=async(e,t)=>{e&&await V({accion:async()=>(await P.usuarios.eliminar(e,{archivos:t}),{mensaje:`Se eliminó el usuario «${e}».`}),operacion:`DELETE /api/users/${e}${t?`?files=true`:``}`,informar:K,recargar:l})};n.addEventListener(`click`,e=>{let t=e.target.closest(`[data-descargar]`),n=e.target.closest(`[data-eliminar]`);t&&u(t.dataset.descargar),n&&window.confirm(`¿Eliminar el usuario «${n.dataset.eliminar}»?`)&&d(n.dataset.eliminar,!1)}),e.querySelector(`#usuarios-refresh`).addEventListener(`click`,l),o.addEventListener(`submit`,e=>{e.preventDefault();let t=i.value;t?window.confirm(`¿Eliminar el usuario «${t}»?`)&&d(t,o.elements.archivos.checked):K({ok:!1,mensaje:`Selecciona el usuario que se va a eliminar.`,operacion:`DELETE /api/users/{name}`})});let f=e.querySelector(`#usuario-nombre`),p=e.querySelector(`#usuario-error`),m=e.querySelector(`#usuario-descargar-nueva`),h=``;return a.addEventListener(`input`,()=>{p.classList.add(`hidden`),m.classList.add(`hidden`)}),a.addEventListener(`submit`,async t=>{t.preventDefault();let n=f.value.trim(),r=e.querySelector(`#usuario-password`).value,i=it(n);if(i){p.textContent=i,p.classList.remove(`hidden`),f.focus();return}let o={nombre:n,sudo:a.elements.sudo.checked,llaves:a.elements.llaves.checked,password:r};B(a,!0);let s=await V({accion:async()=>(await P.usuarios.crear(o),{mensaje:`Se creó el usuario «${n}».`}),operacion:`useradd ${n}${o.sudo?` + grupo sudo`:``}${o.llaves?` + par de llaves`:``}${o.password?` + contraseña`:``}`,informar:K,recargar:l});B(a,!1),s&&(h=o.llaves?n:``,m.classList.toggle(`hidden`,!o.llaves),a.reset())}),m.addEventListener(`click`,()=>u(h)),l(),e}function Et(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap`,e.innerHTML=`
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="upload" class="w-4 h-4"></i><span class="text-[12px] font-medium">Despliegue de unidad</span></div></header>
        <form id="form-despliegue" class="p-3 flex flex-col gap-2" novalidate>
          <div class="flex flex-col gap-1">
            <label class="label" for="unidad-nombre">Nombre de unidad (.service)</label>
            <input id="unidad-nombre" name="unidad" class="input font-mono" placeholder="catalogo.service" autocomplete="off" spellcheck="false">
            <span id="unidad-error" class="text-[11px] text-vm-accent hidden"></span>
          </div>
          <div class="flex flex-col gap-2">
            <div class="flex items-center justify-between">
              <span class="label">Archivos que se despliegan</span>
              <button type="button" id="agregar-archivo" class="btn h-vm-btn px-1.5"><i data-lucide="plus" class="w-4 h-4"></i> Añadir archivo</button>
            </div>
            <div id="archivos" class="flex flex-col gap-2"></div>
          </div>
          <button type="submit" class="btn btn-primary w-fit"><i data-lucide="upload"></i> Desplegar</button>
        </form>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="sliders-horizontal" class="w-4 h-4"></i><span class="text-[12px] font-medium">Control de servicios</span></div>
          <button id="unidades-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
        </header>
        <div class="p-3 flex flex-col gap-2">
          <div class="flex items-center gap-2">
            <select id="unidad-seleccion" class="input flex-1"></select>
            <span id="unidad-estado" class="px-1.5 py-0.5 border text-[11px] rounded-sm text-vm-fgSecondary border-vm-border bg-vm-bg">sin datos</span>
          </div>
          <div class="flex flex-wrap gap-1">
            <button class="btn" data-accion="iniciar"><i data-lucide="play"></i> Iniciar</button>
            <button class="btn" data-accion="detener"><i data-lucide="square"></i> Detener</button>
            <button class="btn" data-accion="reiniciar"><i data-lucide="rotate-ccw"></i> Reiniciar</button>
            <button class="btn" data-accion="estado"><i data-lucide="info"></i> Estado</button>
            <button class="btn" data-accion="habilitar"><i data-lucide="power"></i> Habilitar</button>
            <button class="btn" data-accion="deshabilitar"><i data-lucide="power-off"></i> Deshabilitar</button>
          </div>
        </div>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="activity" class="w-4 h-4"></i><span class="text-[12px] font-medium">Unidades administradas</span></div>
          <span id="unidades-cuenta" class="text-[11px] text-vm-fgSecondary"></span>
        </header>
        <div class="overflow-auto max-h-[220px]">
          <table class="w-full text-[12px]">
            <thead class="bg-vm-bg">
              <tr>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Unidad</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Estado</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Habilitada</th>
              </tr>
            </thead>
            <tbody id="unidades-filas"></tbody>
          </table>
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `,j(e);let t=[],n=e.querySelector(`#archivos`),r=e.querySelector(`#form-despliegue`),i=e.querySelector(`#unidad-seleccion`),a=e.querySelector(`#unidad-estado`),o=e.querySelector(`#unidades-filas`),s=e.querySelector(`#unidades-cuenta`),c=(e={})=>{let t=document.createElement(`div`);t.className=`grid grid-cols-[1fr_1fr_72px_28px] gap-1 items-center`,t.innerHTML=`
      <input class="input font-mono" data-campo="origen" placeholder="./app/catalogo" value="${F(e.origen)}" spellcheck="false">
      <input class="input font-mono" data-campo="destino" placeholder="/opt/catalogo/app" value="${F(e.destino)}" spellcheck="false">
      <input class="input font-mono" data-campo="modo" placeholder="755" value="${F(e.modo)||`755`}" maxlength="4" spellcheck="false">
      <button type="button" class="btn h-vm-btn px-1" data-quitar title="Quitar el archivo"><i data-lucide="x" class="w-4 h-4"></i></button>
    `,t.querySelector(`[data-quitar]`).addEventListener(`click`,()=>{n.children.length>1&&t.remove()}),n.appendChild(t),j(t)},l=e=>{if(!e){a.textContent=`sin datos`,a.className=`px-1.5 py-0.5 border text-[11px] rounded-sm text-vm-fgSecondary border-vm-border bg-vm-bg`;return}let t=L(e.estado);a.textContent=t.texto,a.className=`px-1.5 py-0.5 border text-[11px] rounded-sm ${t.clase}`,a.title=`activa: ${e.activo} · habilitada: ${e.habilitado}`},u=e=>{for(let t of o.querySelectorAll(`tr[data-unidad]`)){let n=t.dataset.unidad===e;t.classList.toggle(`bg-vm-bg`,n),t.setAttribute(`aria-selected`,String(n))}},d=()=>{s.textContent=String(t.length);let e=t.find(e=>e.unidad===i.value)?.unidad??t[0]?.unidad??``;i.innerHTML=t.length?t.map(e=>`<option value="${F(e.unidad)}">${F(e.unidad)}</option>`).join(``):`<option value="">Sin unidades</option>`,i.value=e,t.length===0?(o.innerHTML=z(3,`No hay unidades para mostrar`),l(null)):(o.innerHTML=t.map(e=>{let t=L(e.estado);return`
        <tr data-unidad="${F(e.unidad)}" tabindex="0" aria-selected="false"
          class="border-t border-vm-border hover:bg-vm-bg cursor-pointer outline-none focus:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${F(e.unidad)}</td>
          <td class="px-3 py-1.5"><span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${t.clase}">${t.texto}</span></td>
          <td class="px-3 py-1.5">${e.habilitado?`<i data-lucide="check" class="w-4 h-4 text-vm-success"></i>`:`<span class="text-vm-fgSecondary">—</span>`}</td>
        </tr>
      `}).join(``),j(o),l(t.find(t=>t.unidad===e)??null),u(e))},f=async()=>{o.innerHTML=z(3,`Consultando las unidades…`);try{t=(await P.unidades.listar()).unidades??[],d()}catch(e){t=[],s.textContent=``,i.innerHTML=`<option value="">Sin unidades</option>`,o.innerHTML=z(3,R(e),`text-vm-accent`),l(null)}},p=async()=>{let e=i.value;if(e)try{l(await P.unidades.obtener(e))}catch{l(null)}},m=e=>{e&&(i.value=e,l(t.find(t=>t.unidad===e)??null),u(e),p())};e.querySelector(`#agregar-archivo`).addEventListener(`click`,()=>c()),e.querySelector(`#unidades-refresh`).addEventListener(`click`,f),i.addEventListener(`change`,()=>{u(i.value),p()}),o.addEventListener(`click`,e=>{let t=e.target.closest(`tr[data-unidad]`);t&&m(t.dataset.unidad)}),o.addEventListener(`keydown`,e=>{if(e.key!==`Enter`&&e.key!==` `)return;let t=e.target.closest(`tr[data-unidad]`);t&&(e.preventDefault(),m(t.dataset.unidad))});let h={iniciar:{verbo:`start`,cambio:{activo:!0},mensaje:e=>`Se activó la unidad «${e}».`},detener:{verbo:`stop`,cambio:{activo:!1},mensaje:e=>`Se desactivó la unidad «${e}».`},reiniciar:{verbo:`restart`,mensaje:e=>`Se reinició la unidad «${e}».`},habilitar:{verbo:`enable`,cambio:{habilitado:!0},mensaje:e=>`Se habilitó el arranque de «${e}».`},deshabilitar:{verbo:`disable`,cambio:{habilitado:!1},mensaje:e=>`Se deshabilitó el arranque de «${e}».`},estado:{verbo:`status`}};return e.querySelectorAll(`[data-accion]`).forEach(t=>{t.addEventListener(`click`,async()=>{let n=i.value,r=h[t.dataset.accion];!n||!r?K({ok:!1,mensaje:`Selecciona una unidad para operar.`,operacion:`systemctl`}):(B(e,!0),await V({accion:async()=>{if(r.verbo===`status`){let[e,t]=await Promise.all([P.unidades.obtener(n),P.unidades.estadoDetallado(n)]);l(e);let r=e.habilitado?`habilitada al arranque`:`deshabilitada al arranque`;return{mensaje:`Estado de «${n}»: ${L(e.estado).texto}, ${r}.`,operacion:`systemctl status ${n}`,codigo:0,salida:t.salida,stderr:t.stderr}}r.verbo===`restart`?await P.unidades.reiniciar(n):await P.unidades.cambiarEstado(n,r.cambio);let e=await P.unidades.estadoDetallado(n).catch(()=>null);return{mensaje:r.mensaje(n),operacion:`systemctl ${r.verbo} ${n}`,codigo:0,salida:e?.salida??``,stderr:e?.stderr??``}},operacion:`systemctl ${r.verbo} ${n}`,informar:K,recargar:r.verbo===`status`?void 0:f}),B(e,!1))})}),r.addEventListener(`submit`,async t=>{t.preventDefault();let i=e.querySelector(`#unidad-nombre`).value.trim(),a=e.querySelector(`#unidad-error`),o=ot(i);if(o){a.textContent=o,a.classList.remove(`hidden`);return}a.classList.add(`hidden`);let s=[...n.querySelectorAll(`[data-campo="origen"]`)].map(e=>{let t=e.parentElement;return{origen:e.value.trim(),destino:t.querySelector(`[data-campo="destino"]`).value.trim(),modo:t.querySelector(`[data-campo="modo"]`).value.trim()}});s.find(e=>!e.origen||!ut(e.destino)||!lt(e.modo))?K({ok:!1,mensaje:`Revisa los archivos: el origen es obligatorio, el destino debe ser una ruta absoluta y el modo octal de tres o cuatro dígitos.`,operacion:`PUT /api/units/${i}`}):(B(r,!0),await V({accion:async()=>({...await P.unidades.desplegar(i,{archivos:s}),mensaje:`Se desplegó la unidad «${i}» con ${s.length} archivo(s).`}),operacion:`PUT /api/units/${i}`,informar:K,recargar:f}),B(r,!1))}),c(),f(),e}function Dt(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap`,e.innerHTML=`
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
          <div class="flex items-center gap-1.5"><i data-lucide="server" class="w-4 h-4"></i><span class="text-[12px] font-medium">Servidores en el back-end</span></div>
          <div class="flex items-center gap-1.5">
            <span id="backend-cuenta" class="text-[11px] text-vm-fgSecondary"></span>
            <button id="backend-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
          </div>
        </header>
        <div class="overflow-auto">
          <table class="w-full text-[12px]">
            <thead class="bg-vm-bg">
              <tr>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Dirección</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Puerto</th>
                <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">Estado</th>
                <th class="px-3 py-1"></th>
              </tr>
            </thead>
            <tbody id="backend-filas"></tbody>
          </table>
        </div>
      </section>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="plus-circle" class="w-4 h-4"></i><span class="text-[12px] font-medium">Agregar servidor al back-end</span></div></header>
        <form id="form-backend" class="p-3 flex items-end gap-2" novalidate>
          <div class="flex flex-col gap-1 flex-1">
            <label class="label" for="backend-direccion">Dirección</label>
            <input id="backend-direccion" name="direccion" class="input font-mono" placeholder="192.168.1.111" autocomplete="off" spellcheck="false">
          </div>
          <div class="flex flex-col gap-1 w-[100px]">
            <label class="label" for="backend-puerto">Puerto</label>
            <input id="backend-puerto" name="puerto" class="input font-mono" placeholder="8080" inputmode="numeric" maxlength="5" autocomplete="off">
          </div>
          <button type="submit" class="btn btn-primary"><i data-lucide="plus"></i> Agregar</button>
        </form>
      </section>
    </div>
    <div id="result-slot"></div>
  `,j(e);let t=[],n=e.querySelector(`#backend-filas`),r=e.querySelector(`#backend-cuenta`),i=e.querySelector(`#form-backend`),a=()=>{r.textContent=String(t.length),t.length===0?n.innerHTML=z(4,`El back-end no tiene servidores`):(n.innerHTML=t.map(e=>{let t=Ee(e.estado),n=dt(e.direccion,e.puerto);return`
        <tr class="border-t border-vm-border hover:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${F(e.direccion)}</td>
          <td class="px-3 py-1.5 font-mono">${F(e.puerto)}</td>
          <td class="px-3 py-1.5"><span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${t.clase}">${t.texto}</span></td>
          <td class="px-3 py-1.5 text-right"><button class="btn h-vm-btn px-1" data-eliminar="${F(n)}" title="Quitar del back-end"><i data-lucide="trash-2" class="w-4 h-4"></i></button></td>
        </tr>
      `}).join(``),j(n))},o=async()=>{n.innerHTML=z(4,`Consultando los servidores…`);try{t=(await P.haproxy.listar()).servidores??[],a()}catch(e){t=[],r.textContent=``,n.innerHTML=z(4,R(e),`text-vm-accent`)}},s=async e=>{window.confirm(`¿Quitar ${e} del back-end?`)&&await V({accion:async()=>(await P.haproxy.eliminar(e),{mensaje:`Se quitó ${e} del back-end.`}),operacion:`DELETE /api/haproxy/backend/servers/${e}`,informar:K,recargar:o})};return n.addEventListener(`click`,e=>{let t=e.target.closest(`[data-eliminar]`);t&&s(t.dataset.eliminar)}),e.querySelector(`#backend-refresh`).addEventListener(`click`,o),i.addEventListener(`submit`,async e=>{e.preventDefault();let t=i.elements.direccion.value.trim(),n=Number(i.elements.puerto.value.trim());if(!st(t)){K({ok:!1,mensaje:`La dirección debe ser una IPv4 válida, por ejemplo 192.168.1.111.`,operacion:`POST /api/haproxy/backend/servers`});return}if(!ct(n)){K({ok:!1,mensaje:`El puerto debe ser un número entre 1 y 65535.`,operacion:`POST /api/haproxy/backend/servers`});return}B(i,!0);let r=await V({accion:async()=>(await P.haproxy.agregar({direccion:t,puerto:n}),{mensaje:`Se agregó ${t}:${n} al back-end.`}),operacion:`POST /api/haproxy/backend/servers ${t}:${n}`,informar:K,recargar:o});B(i,!1),r&&i.reset()}),o(),e}var Ot=[`paquetes`,`llaves`,`issue`,`hostname`,`usuarios`,`systemd`,`haproxy`,`sesion`,`maquinas`];function kt(){let e=document.createElement(`div`);e.className=`flex flex-col gap-vm-gap`,e.innerHTML=`
    <section class="card">
      <form id="filtros-historial" class="p-3 flex flex-wrap items-center gap-2">
        <div class="flex items-center gap-1 border border-vm-border px-2 h-vm-input flex-1 min-w-[200px]">
          <i data-lucide="search"></i>
          <input name="q" class="flex-1 outline-none text-[12px]" placeholder="Buscar: paquete, unidad, usuario, dirección…">
        </div>
        <input name="desde" class="input" type="date" title="Desde">
        <span class="text-vm-fgSecondary">→</span>
        <input name="hasta" class="input" type="date" title="Hasta">
        <input name="hora_desde" class="input w-[90px]" type="time" title="Hora desde">
        <span class="text-vm-fgSecondary">→</span>
        <input name="hora_hasta" class="input w-[90px]" type="time" title="Hora hasta">
        <input name="vm" class="input w-[140px] font-mono" placeholder="VM" title="Nombre de la VM">
        <select name="modulo" class="input">
          <option value="">Módulo: todos</option>
          ${Ot.map(e=>`<option value="${e}">${e}</option>`).join(``)}
        </select>
        <button type="submit" class="btn btn-primary"><i data-lucide="search"></i> Aplicar</button>
        <button id="limpiar-historial" type="button" class="btn">Limpiar</button>
      </form>
    </section>
    <section class="card">
      <header class="px-3 py-1.5 border-b border-vm-border flex items-center justify-between bg-[#f7f7f7]">
        <div class="flex items-center gap-1.5"><i data-lucide="history" class="w-4 h-4"></i><span class="text-[12px] font-medium">Historial de operaciones</span></div>
        <button id="exportar-historial" type="button" class="btn" disabled><i data-lucide="file-down"></i> Exportar CSV</button>
      </header>
      <div class="overflow-auto">
        <table class="w-full text-[12px]">
          <thead class="bg-vm-bg">
            <tr>
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">FECHA Y HORA</th>
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">VM</th>
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">MÓDULO</th>
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">DESCRIPCIÓN</th>
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">OPERACIÓN</th>
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">RESULTADO</th>
            </tr>
          </thead>
          <tbody id="historial-filas">
            <tr><td colspan="6" class="px-3 py-6 text-center text-vm-fgSecondary">Consultando…</td></tr>
          </tbody>
        </table>
      </div>
      <footer class="px-3 py-1.5 border-t border-vm-border text-[11px] text-vm-fgSecondary flex items-center justify-between">
        <span id="historial-estado">Consultando…</span>
        <span id="historial-total" class="font-mono">0</span>
      </footer>
    </section>
  `,j(e);let t=e.querySelector(`#filtros-historial`),n=e.querySelector(`#historial-filas`),r=e.querySelector(`#historial-estado`),i=e.querySelector(`#historial-total`),a=e.querySelector(`#exportar-historial`),o=[],s=()=>{let e=new FormData(t),n={};for(let t of[`q`,`desde`,`hasta`,`hora_desde`,`hora_hasta`,`vm`,`modulo`]){let r=String(e.get(t)||``).trim();r&&(n[t]=r)}return n},c=e=>{let t=new Date(e);return Number.isNaN(t.getTime())?e:t.toLocaleString(`es-CO`)},l=()=>{o.length===0?(n.innerHTML=`
        <tr><td colspan="6" class="px-3 py-6 text-center text-vm-fgSecondary">
          No hay operaciones que coincidan con los filtros.
        </td></tr>
      `,a.disabled=!0):(n.innerHTML=o.map(e=>`
        <tr class="border-t border-vm-border hover:bg-[#faf9f9]">
          <td class="px-3 py-1 font-mono text-[11px] whitespace-nowrap">${F(c(e.fecha_hora))}</td>
          <td class="px-3 py-1 font-mono text-[11px]">${F(e.vm||`—`)}</td>
          <td class="px-3 py-1">${F(e.modulo)}</td>
          <td class="px-3 py-1">${F(e.descripcion||e.operacion)}</td>
          <td class="px-3 py-1 font-mono text-[11px]">${F(e.operacion)}</td>
          <td class="px-3 py-1">
            <span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${e.resultado===`ok`?`border-vm-success text-vm-success`:`border-vm-accent text-vm-accent`}">
              ${e.resultado===`ok`?`OK`:`Error`}
            </span>
          </td>
        </tr>
      `).join(``),a.disabled=!1)},u=async()=>{r.textContent=`Consultando…`,r.className=`text-vm-fgSecondary`;try{o=(await P.historial.listar(s())).operaciones??[],l(),r.textContent=o.length===0?`Sin operaciones`:`Historial actualizado`,i.textContent=String(o.length)}catch(e){o=[],n.innerHTML=`
        <tr><td colspan="6" class="px-3 py-6 text-center text-vm-accent">${F(R(e))}</td></tr>
      `,r.textContent=`No fue posible consultar el historial`,r.className=`text-vm-accent`,i.textContent=`0`,a.disabled=!0}};return t.addEventListener(`submit`,e=>{e.preventDefault(),u()}),e.querySelector(`#limpiar-historial`).addEventListener(`click`,()=>{t.reset(),u()}),a.addEventListener(`click`,()=>{let e=o.map(e=>[e.fecha_hora,e.vm,e.modulo,e.descripcion,e.operacion,e.resultado,e.detalle].map(e=>`"${String(e??``).replaceAll(`"`,`""`)}"`).join(`,`)).join(`
`),t=document.createElement(`a`);t.href=URL.createObjectURL(new Blob([`fecha_hora,vm,modulo,descripcion,operacion,resultado,detalle
`+e],{type:`text/csv;charset=utf-8`})),t.download=`historial-vmcons.csv`,t.click(),URL.revokeObjectURL(t.href)}),u(),e}var At={inicio:vt,paquetes:bt,llaves:St,issue:Ct,hostname:wt,usuarios:Tt,systemd:Et,haproxy:Dt,historial:kt},jt=`inicio`,X=null,Z=``;function Mt(){let e=window.location.hash.replace(/^#\/?/,``);return Object.hasOwn(At,e)?e:jt}function Nt(){let e=document.getElementById(`app`);e.innerHTML=``,e.className=`flex h-dvh flex-col overflow-hidden`;let t=document.createElement(`div`);t.className=`shrink-0`;let n=document.createElement(`main`);n.className=`flex-1 min-h-0 overflow-y-auto`;let r=document.createElement(`div`);return r.className=`max-w-vm mx-auto px-3 py-3`,n.appendChild(r),e.append(Ae({usuario:Z,alSalir:Ft}),t,n),X={barra:t,principal:r},X}function Q(){let{barra:e,principal:t}=X??Nt(),n=Mt();e.replaceChildren(Ne(n));let r=At[n]();r.classList.add(`fade-slide`),t.replaceChildren(r);let i=document.getElementById(`result-slot`);if(i&&!document.getElementById(`resultado-global`)){let e=Ue();e.id=`resultado-global`,i.appendChild(e)}}window.addEventListener(`hashchange`,()=>{X&&Q()}),document.addEventListener(`vmcons:maquina`,()=>{X&&Q()});function Pt(e,t){let n=document.getElementById(`app`);X=null,n.className=``,t?(n.innerHTML=`
      <div class="min-h-dvh flex items-center justify-center bg-vm-bg px-4">
        <div class="card p-3 max-w-[420px] flex flex-col gap-2">
          <p class="text-[13px] font-medium">No fue posible comprobar la sesión de la consola</p>
          <p class="text-[11px] text-vm-accent">${F(t)}</p>
          <button id="acceso-reintentar" type="button" class="btn btn-primary justify-center">
            Reintentar
          </button>
        </div>
      </div>
    `,n.querySelector(`#acceso-reintentar`).addEventListener(`click`,$)):n.appendChild(ht({configurada:!!(e&&e.configurada),alEntrar:e=>{Z=e&&e.usuario||Z,$()}}))}async function Ft(){try{await P.auth.salir()}catch{}Z=``,$()}async function $(){try{let e=await P.auth.estado();if(!e.autenticado){Pt(e);return}Z=e.usuario||``,Q()}catch(e){Pt(null,e.message)}}$();