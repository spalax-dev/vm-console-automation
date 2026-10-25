(function(){let e=document.createElement(`link`).relList;if(e&&e.supports&&e.supports(`modulepreload`))return;for(let e of document.querySelectorAll(`link[rel="modulepreload"]`))n(e);new MutationObserver(e=>{for(let t of e)if(t.type===`childList`)for(let e of t.addedNodes)e.tagName===`LINK`&&e.rel===`modulepreload`&&n(e)}).observe(document,{childList:!0,subtree:!0});function t(e){let t={};return e.integrity&&(t.integrity=e.integrity),e.referrerPolicy&&(t.referrerPolicy=e.referrerPolicy),t.credentials=e.crossOrigin===`use-credentials`?`include`:e.crossOrigin===`anonymous`?`omit`:`same-origin`,t}function n(e){if(e.ep)return;e.ep=!0;let n=t(e);fetch(e.href,n)}})();var e={xmlns:`http://www.w3.org/2000/svg`,width:24,height:24,viewBox:`0 0 24 24`,fill:`none`,stroke:`currentColor`,"stroke-width":2,"stroke-linecap":`round`,"stroke-linejoin":`round`},t=([e,n,r])=>{let i=document.createElementNS(`http://www.w3.org/2000/svg`,e);return Object.keys(n).forEach(e=>{i.setAttribute(e,String(n[e]))}),r?.length&&r.forEach(e=>{let n=t(e);i.appendChild(n)}),i},n=(n,r={})=>t([`svg`,{...e,...r},n]),r=(...e)=>e.filter((e,t,n)=>!!e&&e.trim()!==``&&n.indexOf(e)===t).join(` `).trim(),i=e=>{for(let t in e)if(t.startsWith(`aria-`)||t===`role`||t===`title`)return!0;return!1},a=e=>{let t=``,n=!1;for(let r of e)r===`-`||r===`_`||r<=` `?n=t.length>0:(t.length===0?t+=r.toLowerCase():t+=n?r.toUpperCase():r,n=!1);return t},o=e=>{let t=a(e);return t.charAt(0).toUpperCase()+t.slice(1)},s=e=>Array.from(e.attributes).reduce((e,t)=>(e[t.name]=t.value,e),{}),c=e=>typeof e==`string`?e:!e||!e.class?``:e.class&&typeof e.class==`string`?e.class.split(` `):e.class&&Array.isArray(e.class)?e.class:``,l=(t,{nameAttr:a,icons:l,attrs:u})=>{let d=t.getAttribute(a);if(d==null)return;let f=l[o(d)];if(!f)return console.warn(`${t.outerHTML} icon name was not found in the provided icons object.`);let p=s(t),m=i(p)?{}:{"aria-hidden":`true`},h={...e,"data-lucide":d,...m,...u,...p},g=c(p),_=c(u),v=r(`lucide`,`lucide-${d}`,...g,..._);v&&Object.assign(h,{class:v});let y=n(f,h);return t.parentNode?.replaceChild(y,t)},u=[[`path`,{d:`M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2`}]],d=[[`path`,{d:`M20 6 9 17l-5-5`}]],f=[[`path`,{d:`m6 9 6 6 6-6`}]],p=[[`circle`,{cx:`12`,cy:`12`,r:`10`}],[`path`,{d:`m16 9-5.5 5.5L8 12`}]],m=[[`circle`,{cx:`12`,cy:`12`,r:`10`}],[`path`,{d:`M8 12h8`}],[`path`,{d:`M12 8v8`}]],h=[[`circle`,{cx:`12`,cy:`12`,r:`10`}],[`path`,{d:`m15 9-6 6`}],[`path`,{d:`m9 9 6 6`}]],g=[[`rect`,{width:`14`,height:`14`,x:`8`,y:`8`,rx:`2`,ry:`2`}],[`path`,{d:`M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2`}]],_=[[`path`,{d:`M12 15V3`}],[`path`,{d:`M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4`}],[`path`,{d:`m7 10 5 5 5-5`}]],v=[[`path`,{d:`M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0`}],[`circle`,{cx:`12`,cy:`12`,r:`3`}]],y=[[`path`,{d:`M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z`}],[`path`,{d:`M14 2v5a1 1 0 0 0 1 1h5`}],[`path`,{d:`M12 18v-6`}],[`path`,{d:`m9 15 3 3 3-3`}]],b=[[`path`,{d:`M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z`}],[`path`,{d:`M14 2v5a1 1 0 0 0 1 1h5`}],[`path`,{d:`M10 9H8`}],[`path`,{d:`M16 13H8`}],[`path`,{d:`M16 17H8`}]],x=[[`circle`,{cx:`18`,cy:`18`,r:`3`}],[`circle`,{cx:`6`,cy:`6`,r:`3`}],[`path`,{d:`M6 21V9a9 9 0 0 0 9 9`}]],S=[[`path`,{d:`M15 21v-8a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v8`}],[`path`,{d:`M3 10a2 2 0 0 1 .709-1.528l7-6a2 2 0 0 1 2.582 0l7 6A2 2 0 0 1 21 10v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z`}]],C=[[`circle`,{cx:`12`,cy:`12`,r:`10`}],[`path`,{d:`M12 16v-4`}],[`path`,{d:`M12 8h.01`}]],w=[[`path`,{d:`M2.586 17.414A2 2 0 0 0 2 18.828V21a1 1 0 0 0 1 1h3a1 1 0 0 0 1-1v-1a1 1 0 0 1 1-1h1a1 1 0 0 0 1-1v-1a1 1 0 0 1 1-1h.172a2 2 0 0 0 1.414-.586l.814-.814a6.5 6.5 0 1 0-4-4z`}],[`circle`,{cx:`16.5`,cy:`7.5`,r:`.5`,fill:`currentColor`}]],T=[[`rect`,{width:`7`,height:`7`,x:`3`,y:`3`,rx:`1`}],[`rect`,{width:`7`,height:`7`,x:`14`,y:`3`,rx:`1`}],[`rect`,{width:`7`,height:`7`,x:`14`,y:`14`,rx:`1`}],[`rect`,{width:`7`,height:`7`,x:`3`,y:`14`,rx:`1`}]],E=[[`path`,{d:`M3 5h.01`}],[`path`,{d:`M3 12h.01`}],[`path`,{d:`M3 19h.01`}],[`path`,{d:`M8 5h13`}],[`path`,{d:`M8 12h13`}],[`path`,{d:`M8 19h13`}]],ee=[[`rect`,{width:`20`,height:`14`,x:`2`,y:`3`,rx:`2`}],[`line`,{x1:`8`,x2:`16`,y1:`21`,y2:`21`}],[`line`,{x1:`12`,x2:`12`,y1:`17`,y2:`21`}]],te=[[`path`,{d:`M11 21.73a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73z`}],[`path`,{d:`M12 22V12`}],[`polyline`,{points:`3.29 7 12 12 20.71 7`}],[`path`,{d:`m7.5 4.27 9 5.15`}]],D=[[`path`,{d:`M5 5a2 2 0 0 1 3.008-1.728l11.997 6.998a2 2 0 0 1 .003 3.458l-12 7A2 2 0 0 1 5 19z`}]],ne=[[`path`,{d:`M5 12h14`}],[`path`,{d:`M12 5v14`}]],re=[[`path`,{d:`M18.36 6.64A9 9 0 0 1 20.77 15`}],[`path`,{d:`M6.16 6.16a9 9 0 1 0 12.68 12.68`}],[`path`,{d:`M12 2v4`}],[`path`,{d:`m2 2 20 20`}]],ie=[[`path`,{d:`M12 2v10`}],[`path`,{d:`M18.4 6.6a9 9 0 1 1-12.77.04`}]],ae=[[`path`,{d:`M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8`}],[`path`,{d:`M21 3v5h-5`}],[`path`,{d:`M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16`}],[`path`,{d:`M8 16H3v5`}]],oe=[[`path`,{d:`M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8`}],[`path`,{d:`M3 3v5h5`}],[`path`,{d:`M12 7v5l4 2`}]],se=[[`path`,{d:`M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8`}],[`path`,{d:`M3 3v5h5`}]],ce=[[`path`,{d:`M15.2 3a2 2 0 0 1 1.4.6l3.8 3.8a2 2 0 0 1 .6 1.4V19a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z`}],[`path`,{d:`M17 21v-7a1 1 0 0 0-1-1H8a1 1 0 0 0-1 1v7`}],[`path`,{d:`M7 3v4a1 1 0 0 0 1 1h7`}]],le=[[`path`,{d:`m21 21-4.34-4.34`}],[`circle`,{cx:`11`,cy:`11`,r:`8`}]],ue=[[`rect`,{width:`20`,height:`8`,x:`2`,y:`2`,rx:`2`,ry:`2`}],[`rect`,{width:`20`,height:`8`,x:`2`,y:`14`,rx:`2`,ry:`2`}],[`line`,{x1:`6`,x2:`6.01`,y1:`6`,y2:`6`}],[`line`,{x1:`6`,x2:`6.01`,y1:`18`,y2:`18`}]],de=[[`path`,{d:`M9.671 4.136a2.34 2.34 0 0 1 4.659 0 2.34 2.34 0 0 0 3.319 1.915 2.34 2.34 0 0 1 2.33 4.033 2.34 2.34 0 0 0 0 3.831 2.34 2.34 0 0 1-2.33 4.033 2.34 2.34 0 0 0-3.319 1.915 2.34 2.34 0 0 1-4.659 0 2.34 2.34 0 0 0-3.32-1.915 2.34 2.34 0 0 1-2.33-4.033 2.34 2.34 0 0 0 0-3.831A2.34 2.34 0 0 1 6.35 6.051a2.34 2.34 0 0 0 3.319-1.915`}],[`circle`,{cx:`12`,cy:`12`,r:`3`}]],fe=[[`path`,{d:`M10 5H3`}],[`path`,{d:`M12 19H3`}],[`path`,{d:`M14 3v4`}],[`path`,{d:`M16 17v4`}],[`path`,{d:`M21 12h-9`}],[`path`,{d:`M21 19h-5`}],[`path`,{d:`M21 5h-7`}],[`path`,{d:`M8 10v4`}],[`path`,{d:`M8 12H3`}]],pe=[[`rect`,{width:`18`,height:`18`,x:`3`,y:`3`,rx:`2`}]],me=[[`path`,{d:`M10 11v6`}],[`path`,{d:`M14 11v6`}],[`path`,{d:`M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6`}],[`path`,{d:`M3 6h18`}],[`path`,{d:`M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2`}]],he=[[`path`,{d:`m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3`}],[`path`,{d:`M12 9v4`}],[`path`,{d:`M12 17h.01`}]],ge=[[`path`,{d:`M12 3v12`}],[`path`,{d:`m17 8-5-5-5 5`}],[`path`,{d:`M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4`}]],_e=[[`path`,{d:`M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2`}],[`circle`,{cx:`9`,cy:`7`,r:`4`}],[`line`,{x1:`22`,x2:`16`,y1:`11`,y2:`11`}]],ve=[[`path`,{d:`M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2`}],[`circle`,{cx:`9`,cy:`7`,r:`4`}],[`line`,{x1:`19`,x2:`19`,y1:`8`,y2:`14`}],[`line`,{x1:`22`,x2:`16`,y1:`11`,y2:`11`}]],ye=[[`path`,{d:`M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2`}],[`path`,{d:`M16 3.128a4 4 0 0 1 0 7.744`}],[`path`,{d:`M22 21v-2a4 4 0 0 0-3-3.87`}],[`circle`,{cx:`9`,cy:`7`,r:`4`}]],be=[[`path`,{d:`M18 6 6 18`}],[`path`,{d:`m6 6 12 12`}]],O=({icons:e={},nameAttr:t=`data-lucide`,attrs:n={},root:r=document,inTemplates:i}={})=>{if(!Object.values(e).length)throw Error(`Please provide an icons object.
If you want to use all the icons you can import it like:
 \`import { createIcons, icons } from 'lucide';
lucide.createIcons({icons});\``);if(r===void 0)throw Error("`createIcons()` only works in a browser environment.");if(Array.from(r.querySelectorAll(`[${t}]`)).forEach(r=>l(r,{nameAttr:t,icons:e,attrs:n})),i&&Array.from(r.querySelectorAll(`template`)).forEach(r=>O({icons:e,nameAttr:t,attrs:n,root:r.content,inTemplates:i})),t===`data-lucide`){let t=r.querySelectorAll(`[icon-name]`);t.length>0&&(console.warn(`[Lucide] Some icons were found with the now deprecated icon-name attribute. These will still be replaced for backwards compatibility, but will no longer be supported in v1.0 and you should switch to data-lucide`),Array.from(t).forEach(t=>l(t,{nameAttr:`icon-name`,icons:e,attrs:n})))}},xe={Activity:u,AlertTriangle:he,Check:d,CheckCircle2:p,ChevronDown:f,Copy:g,Download:_,Eye:v,FileDown:y,FileText:b,GitMerge:x,History:oe,Home:S,Info:C,KeyRound:w,LayoutGrid:T,List:E,Monitor:ee,Package:te,Play:D,Plus:ne,PlusCircle:m,Power:ie,PowerOff:re,RefreshCw:ae,RotateCcw:se,Save:ce,Search:le,Server:ue,Settings:de,SlidersHorizontal:fe,Square:pe,Trash2:me,Upload:ge,UserMinus:_e,UserPlus:ve,Users:ye,X:be,XCircle:h};function k(e=document){e&&O({icons:xe,attrs:{"stroke-width":1.5,width:16,height:16},nameAttr:`data-lucide`,root:e})}var A=`/api`,j={Accept:`application/json`,"Content-Type":`application/json`},M=class extends Error{constructor(e,t){super(t&&t.mensaje||`La consola respondió con el código ${e}.`),this.name=`ApiError`,this.estado=e,this.mensaje=this.message,this.campo=t&&t.campo,this.tecnico=t&&t.tecnico,this.solicitud=``}get noImplementado(){return this.estado===501}};function Se(e,t){if(!t)return A+e;let n=Object.entries(t).filter(([,e])=>e!=null&&e!==!1).map(([e,t])=>`${encodeURIComponent(e)}=${encodeURIComponent(String(t))}`);return n.length>0?`${A}${e}?${n.join(`&`)}`:A+e}async function N(e,t,{cuerpo:n,consulta:r,descarga:i=!1}={}){let a={method:e,headers:{Accept:`application/json`}};n!==void 0&&(a.headers=j,a.body=JSON.stringify(n));let o=await fetch(Se(t,r),a),s=o.headers.get(`X-Request-ID`)||``;if(o.status===204)return null;let c=o.headers.get(`content-type`)||``;if(!o.ok){let e=null;if(c.includes(`application/json`))try{e=await o.json()}catch{e=null}let t=new M(o.status,e);throw t.solicitud=s,t}return i?o.blob():c.includes(`application/json`)?o.json():o.text()}function P(e,t){let n=URL.createObjectURL(e),r=document.createElement(`a`);r.href=n,r.download=t,document.body.appendChild(r),r.click(),r.remove(),URL.revokeObjectURL(n)}var F={sesion:{obtener:()=>N(`GET`,`/session`),vincular:e=>N(`PUT`,`/session`,{cuerpo:{vm:e}}),registrar:(e,t)=>N(`POST`,`/session/register`,{cuerpo:{usuario:e,password:t}}),olvidar:()=>N(`DELETE`,`/session/register`),diagnostico:()=>N(`GET`,`/session/diagnostico`)},historial:{listar:e=>N(`GET`,`/historial`,{consulta:e})},maquinas:{listar:e=>N(`GET`,`/vms`,{consulta:{q:e&&e.trim()?e.trim():void 0}}),encender:e=>N(`POST`,`/vms/${encodeURIComponent(e)}/poweron`)},paquetes:{listar:()=>N(`GET`,`/packages`),instalar:e=>N(`POST`,`/packages`,{cuerpo:{nombre:e}}),eliminar:e=>N(`DELETE`,`/packages/${encodeURIComponent(e)}`)},llaves:{generarRoot:()=>N(`POST`,`/keys/root`),descargarRoot:()=>N(`GET`,`/keys/root`,{descarga:!0})},issue:{obtener:()=>N(`GET`,`/issue`),reemplazar:e=>N(`PUT`,`/issue`,{cuerpo:e})},hostname:{obtener:()=>N(`GET`,`/hostname`),reemplazar:e=>N(`PUT`,`/hostname`,{cuerpo:e})},usuarios:{listar:()=>N(`GET`,`/users`),crear:e=>N(`POST`,`/users`,{cuerpo:e}),obtener:e=>N(`GET`,`/users/${encodeURIComponent(e)}`),eliminar:(e,t={})=>N(`DELETE`,`/users/${encodeURIComponent(e)}`,{consulta:{files:t.archivos?!0:void 0}}),descargarLlave:e=>N(`GET`,`/users/${encodeURIComponent(e)}/key`,{descarga:!0})},unidades:{listar:()=>N(`GET`,`/units`),obtener:e=>N(`GET`,`/units/${encodeURIComponent(e)}`),desplegar:(e,t)=>N(`PUT`,`/units/${encodeURIComponent(e)}`,{cuerpo:t}),cambiarEstado:(e,t)=>N(`PATCH`,`/units/${encodeURIComponent(e)}`,{cuerpo:t}),reiniciar:e=>N(`POST`,`/units/${encodeURIComponent(e)}/restarts`)},haproxy:{listar:()=>N(`GET`,`/haproxy/backend/servers`),agregar:e=>N(`POST`,`/haproxy/backend/servers`,{cuerpo:e}),eliminar:e=>N(`DELETE`,`/haproxy/backend/servers/${encodeURIComponent(e)}`)}};function I(e){return e==null?``:String(e).replaceAll(`&`,`&amp;`).replaceAll(`<`,`&lt;`).replaceAll(`>`,`&gt;`).replaceAll(`"`,`&quot;`).replaceAll(`'`,`&#39;`)}function L(e){switch(e){case`running`:return{texto:`Encendida`,clase:`text-vm-success border-vm-success bg-[#e8f5e9]`,punto:`bg-vm-success`};case`paused`:return{texto:`Pausada`,clase:`text-[#b9770e] border-vm-warning bg-[#fdf3e3]`,punto:`bg-vm-warning`};case`poweroff`:case`saved`:return{texto:`Apagada`,clase:`text-vm-muted border-vm-muted bg-[#f2f4f4]`,punto:`bg-vm-muted`};default:return{texto:`Desconocido`,clase:`text-vm-fgSecondary border-vm-border bg-vm-bg`,punto:`bg-vm-muted`}}}function R(e){switch(e){case`active`:return{texto:`activa`,clase:`text-vm-success border-vm-success bg-[#e8f5e9]`,punto:`bg-vm-success`};case`failed`:return{texto:`falló`,clase:`text-vm-accent border-vm-accent bg-[#fdf1ef]`,punto:`bg-vm-accent`};case`inactive`:return{texto:`inactiva`,clase:`text-vm-fgSecondary border-vm-border bg-vm-bg`,punto:`bg-vm-muted`};default:return{texto:`desconocida`,clase:`text-vm-fgSecondary border-vm-border bg-vm-bg`,punto:`bg-vm-muted`}}}function Ce(e){switch(e){case`up`:return{texto:`UP`,clase:`text-vm-success border-vm-success bg-[#e8f5e9]`,punto:`bg-vm-success`};case`down`:return{texto:`DOWN`,clase:`text-vm-accent border-vm-accent bg-[#fdf1ef]`,punto:`bg-vm-accent`};default:return{texto:`DESCONOCIDO`,clase:`text-vm-fgSecondary border-vm-border bg-vm-bg`,punto:`bg-vm-muted`}}}function z(e){return e&&e.noImplementado?`Este módulo aún no está conectado con la máquina.`:e&&e.mensaje?e.mensaje:e instanceof TypeError?`No fue posible contactar con la consola. Verifica que el servidor esté en ejecución.`:`Ocurrió un error inesperado al ejecutar la operación.`}function we(e,t=``){return{ok:!1,mensaje:z(e),operacion:t,codigo:e&&e.codigo,stderr:e&&e.stderr||``,tecnico:e&&(e.tecnico||e.campo)||``,solicitud:e&&e.solicitud||``}}function B(e,t,n=`text-vm-fgSecondary`){return`<tr><td colspan="${e}" class="px-3 py-3 text-center text-[12px] ${n}">${I(t)}</td></tr>`}function V(e,t){e.querySelectorAll(`button, input, select, textarea`).forEach(e=>{e.disabled=t})}async function H({accion:e,operacion:t,recargar:n,informar:r}){try{let i=await e();return r({ok:!0,mensaje:i&&i.mensaje||`La operación terminó correctamente.`,operacion:i&&i.operacion||t,codigo:i&&i.codigo||0,salida:i&&i.salida||``,stderr:i&&i.stderr||``,tecnico:i&&i.tecnico||``}),n&&n(),!0}catch(e){return r(we(e,t)),!1}}function U({maquina:e,alRegistrar:t}){let n=document.createElement(`div`);n.className=`fixed inset-0 z-50 bg-black/30 flex items-center justify-center p-4`,n.innerHTML=`
    <section class="card w-[400px] max-w-full" role="dialog" aria-modal="true" aria-labelledby="registro-titulo">
      <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
        <span id="registro-titulo" class="text-[12px] font-medium">Registrar ${I(e.nombre)}</span>
        <button id="registro-cerrar" type="button" class="btn h-vm-btn px-1" title="Cerrar" aria-label="Cerrar">
          <i data-lucide="x" class="w-4 h-4"></i>
        </button>
      </header>
      <form id="registro-form" class="p-3 flex flex-col gap-2">
        <p class="text-[11px] text-vm-fgSecondary">
          La máquina tiene Guest Additions pero aún no está registrada. Ingresa un usuario con permisos
          para preparar SSH; las credenciales se guardan solo en este equipo.
        </p>
        <div id="registro-diagnostico" class="border border-vm-border p-2 flex flex-col gap-1 text-[11px]">
          <div class="flex items-center justify-between">
            <span class="text-vm-fgSecondary">Guest Additions</span>
            <span id="diag-ga" class="font-mono">consultando…</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-vm-fgSecondary">SSH (puerto 22)</span>
            <span id="diag-ssh" class="font-mono">consultando…</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-vm-fgSecondary">Usuario sugerido</span>
            <span id="diag-usuario" class="font-mono">—</span>
          </div>
          <ul id="diag-problemas" class="list-disc pl-4 text-vm-accent"></ul>
          <ul id="diag-sugerencias" class="list-disc pl-4 text-vm-fgSecondary"></ul>
        </div>
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
  `,k(n);let r=()=>n.remove(),i=n.querySelector(`#registro-form`),a=n.querySelector(`#registro-error`),o=i.querySelector(`button[type=submit]`),s=n.querySelector(`#registro-progreso`),c=n.querySelector(`#registro-barra`),l=n.querySelector(`#registro-porcentaje`),u=n.querySelector(`#registro-paso`),d=n.querySelector(`#registro-ayuda`),f=n.querySelector(`#registro-cerrar`),p=n.querySelector(`#registro-cancelar`),m=n.querySelector(`#diag-ga`),h=n.querySelector(`#diag-ssh`),g=n.querySelector(`#diag-usuario`),_=n.querySelector(`#diag-problemas`),v=n.querySelector(`#diag-sugerencias`),y=i.querySelector(`input[name=usuario]`),b=!1,x=0,S=()=>{b||r()},C=async()=>{try{let e=await F.sesion.diagnostico();m.textContent=e.guest_additions_version||`no informa`,h.textContent=e.ssh_abierto?`abierto`:`cerrado`,h.className=`font-mono ${e.ssh_abierto?`text-vm-success`:`text-vm-accent`}`,g.textContent=e.usuario_sugerido||`—`,_.innerHTML=(e.problemas??[]).map(e=>`<li>${I(e)}</li>`).join(``),v.innerHTML=(e.sugerencias??[]).map(e=>`<li>${I(e)}</li>`).join(``),e.usuario_sugerido&&y.value===`root`&&(y.value=e.usuario_sugerido)}catch{m.textContent=`no disponible`,h.textContent=`no disponible`,h.className=`font-mono text-vm-fgSecondary`}},w=e=>{b=e;for(let t of i.querySelectorAll(`input, button`))t.disabled=e;f.disabled=e,p.disabled=e,o.innerHTML=e?`Registrando…`:`<i data-lucide="key-round"></i> Registrar`,e||k(o)},T=()=>{s.classList.remove(`hidden`),s.classList.add(`flex`);let e=6,t=()=>{c.style.width=`${e}%`,l.textContent=`${e} %`,e>=82?u.textContent=`Verificando el acceso SSH…`:e>=54?u.textContent=`Preparando SSH y la llave de root…`:e>=26&&(u.textContent=`Conectando con Guest Additions…`)};t(),x=setInterval(()=>{e=Math.min(e+5,88),t()},600)},E=(e,t)=>{clearInterval(x),c.style.width=`100%`,l.textContent=`100 %`,c.className=`h-full transition-all duration-300 ${e?`bg-vm-success`:`bg-vm-accent`}`,u.textContent=t,d.textContent=e?`La máquina quedó registrada.`:`Revisa el detalle técnico e inténtalo de nuevo.`};f.addEventListener(`click`,S),p.addEventListener(`click`,S),n.addEventListener(`click`,e=>{e.target===n&&S()}),n.addEventListener(`keydown`,e=>{e.key===`Escape`&&S()}),i.addEventListener(`submit`,async e=>{e.preventDefault(),a.textContent=``;let n=new FormData(i);w(!0),T();try{await F.sesion.registrar(String(n.get(`usuario`)||``),String(n.get(`password`)||``)),E(!0,`Máquina registrada.`),setTimeout(()=>{r(),t&&t()},650)}catch(e){if(E(!1,`No fue posible registrar la máquina.`),a.textContent=z(e),await C(),e.tecnico){let t=document.createElement(`pre`);t.className=`mt-1 p-1.5 bg-vm-bg border border-vm-border text-[10px] font-mono whitespace-pre-wrap max-h-[120px] overflow-auto`,t.textContent=e.tecnico,a.appendChild(t)}if(e.solicitud){let t=document.createElement(`div`);t.className=`mt-1 text-[10px] font-mono text-vm-fgSecondary`,t.textContent=`Solicitud: ${e.solicitud}`,a.appendChild(t)}w(!1)}}),document.body.appendChild(n),i.querySelector(`input[name=usuario]`).focus(),C()}var Te=200;function Ee({alVincular:e}={}){let t=document.createElement(`div`);t.className=`flex items-center gap-1.5`,t.innerHTML=`
    <div class="relative">
      <button id="vm-boton" type="button" class="btn h-vm-btn pr-1.5 max-w-[260px]"
        aria-haspopup="listbox" aria-expanded="false" aria-controls="vm-panel" title="Elegir la máquina virtual (Guest Additions y red puente/host-only)">
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
        <div id="vm-progreso" class="hidden flex-col gap-1 px-2 py-1.5 border-t border-vm-border" aria-live="polite">
          <div class="flex items-center justify-between text-[10px] text-vm-fgSecondary">
            <span id="vm-progreso-texto">Encendiendo la máquina…</span>
            <i data-lucide="loader-2" class="w-3 h-3 text-vm-accent animate-spin"></i>
          </div>
          <div class="h-1 w-full bg-vm-bg border border-vm-border overflow-hidden">
            <div class="barra-indeterminada h-full bg-vm-accent"></div>
          </div>
        </div>
        <footer id="vm-pie" class="px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary"></footer>
      </div>
    </div>
    <button id="vm-encender" type="button" class="btn h-vm-btn px-1.5 hidden" title="Encender la máquina virtual">
      <i data-lucide="power" class="w-4 h-4"></i>
    </button>
  `,k(t);let n=t.querySelector(`#vm-boton`),r=t.querySelector(`#vm-nombre`),i=t.querySelector(`#vm-punto`),a=t.querySelector(`#vm-panel`),o=t.querySelector(`#vm-busqueda`),s=t.querySelector(`#vm-lista`),c=t.querySelector(`#vm-pie`),l=t.querySelector(`#vm-encender`),u=t.querySelector(`#vm-progreso`),d=t.querySelector(`#vm-progreso-texto`),f=[],p=0,m=0,h=null,g=e=>{if(h=e,!e){r.textContent=`Elegir máquina`,i.className=`w-1.5 h-1.5 rounded-full bg-vm-muted shrink-0`,l.classList.add(`hidden`);return}let t=L(e.estado);r.textContent=e.nombre,i.className=`w-1.5 h-1.5 rounded-full ${t.punto} shrink-0`,n.title=`Máquina activa: ${e.nombre} (${t.texto})`,l.classList.toggle(`hidden`,e.estado===`running`),l.disabled=!1},_=()=>{f.length===0?s.innerHTML=`
        <li class="px-2 py-3 text-center text-[11px] text-vm-fgSecondary">
          No hay máquinas administrables que coincidan con la búsqueda
        </li>
      `:(s.innerHTML=f.map(e=>{let t=L(e.estado);return`
        <li>
          <button type="button" role="option" aria-selected="${e.activa}"
            class="w-full text-left px-2 py-1.5 flex items-center gap-2 hover:bg-[#fdf5f4] ${e.activa?`border-l-2 border-[#C0392B] bg-[#faf9f9]`:`border-l-2 border-transparent`}"
            data-maquina="${I(e.nombre)}">
            <span class="w-1.5 h-1.5 rounded-full shrink-0 ${t.punto}"></span>
            <span class="flex-1 min-w-0">
              <span class="block text-[12px] truncate">${I(e.nombre)}</span>
              <span class="block text-[10px] font-mono text-vm-fgSecondary truncate">${I(e.uuid)}</span>
            </span>
            ${e.registrada?``:`<span class="text-[10px] text-vm-accent shrink-0">Sin registrar</span>`}
            <span class="text-[10px] text-vm-fgSecondary shrink-0">${t.texto}</span>
            ${e.activa?`<i data-lucide="check" class="w-3.5 h-3.5 text-[#C0392B] shrink-0"></i>`:``}
          </button>
        </li>
      `}).join(``),k(s))},v=async(e=``)=>{let t=++p;c.textContent=`Consultando…`,c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary`;try{let n=await F.maquinas.listar(e);if(t!==p)return;f=n.maquinas??[];let r=f.find(e=>e.activa);g(r??h),_(),c.textContent=`${f.length} máquina(s) administrable(s) · Guest Additions y red puente/host-only`,c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary`}catch(e){if(t!==p)return;f=[],s.innerHTML=`
        <li class="px-2 py-3 text-center text-[11px] text-vm-accent">${I(z(e))}</li>
      `,c.textContent=`No se pudo leer el inventario`,c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent`}},y=()=>{a.classList.remove(`hidden`),n.setAttribute(`aria-expanded`,`true`),o.value=``,o.focus(),v(``)},b=()=>{a.classList.add(`hidden`),n.setAttribute(`aria-expanded`,`false`)},x=async t=>{c.textContent=`Vinculando con ${t}…`;try{let n=await F.sesion.vincular(t),r={...n,activa:!0};g(r),b(),e&&e(r),n.registrada||U({maquina:r,alRegistrar:()=>{v(``),e&&e(r)}})}catch(e){c.textContent=z(e),c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent`}};n.addEventListener(`click`,()=>{a.classList.contains(`hidden`)?y():b()});let S=async e=>{let t=Date.now();for(;Date.now()-t<9e4;){await new Promise(e=>window.setTimeout(e,2e3));try{let e=await F.sesion.obtener();if(e.estado===`running`)return e}catch{}try{await v(``)}catch{}let e=Math.floor((Date.now()-t)/1e3);e>=30?d.textContent=`Esperando Guest Additions o SSH…`:e>=8&&(d.textContent=`Esperando a que arranque el sistema…`)}return null};return l.addEventListener(`click`,async()=>{if(h){l.disabled=!0,u.classList.remove(`hidden`),u.classList.add(`flex`),d.textContent=`Encendiendo ${h.nombre}…`,c.textContent=`La máquina está arrancando…`,c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary`;try{let t=await F.maquinas.encender(h.nombre);g({...t,activa:!0}),d.textContent=`${t.nombre} encendida. Esperando la sesión…`,c.textContent=`${t.nombre} está en ejecución; esperando Guest Additions o SSH…`,await S(t.nombre)?(d.textContent=`${t.nombre} en ejecución y sesión disponible.`,c.textContent=`${t.nombre} está en ejecución`,e&&await Promise.resolve(e(t)),await v(``),await new Promise(e=>window.setTimeout(e,800))):(d.textContent=`${t.nombre} está encendida, pero Guest Additions/SSH aún no responde.`,c.textContent=`La máquina está encendida; actualiza en unos segundos.`,c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-fgSecondary`,a.classList.remove(`hidden`),n.setAttribute(`aria-expanded`,`true`),l.disabled=!1)}catch(e){c.textContent=z(e),c.className=`px-2 py-1 border-t border-vm-border text-[10px] text-vm-accent`,a.classList.remove(`hidden`),n.setAttribute(`aria-expanded`,`true`),l.disabled=!1}finally{u.classList.add(`hidden`),u.classList.remove(`flex`)}}}),o.addEventListener(`input`,()=>{window.clearTimeout(m),m=window.setTimeout(()=>v(o.value),Te)}),o.addEventListener(`keydown`,e=>{e.key===`Escape`&&(b(),n.focus())}),s.addEventListener(`click`,e=>{let t=e.target.closest(`[data-maquina]`);t&&x(t.dataset.maquina)}),document.addEventListener(`click`,e=>{t.contains(e.target)||b()}),document.addEventListener(`keydown`,e=>{e.key===`Escape`&&b()}),v(``),t}function De(){let e=document.createElement(`header`);e.className=`h-vm-header bg-vm-surface border-b border-vm-border flex items-center justify-between px-3`,e.innerHTML=`
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
    </div>
  `,k(e);let t=()=>Oe(e);return e.querySelector(`#vm-slot`).appendChild(Ee({alVincular:e=>{t(),document.dispatchEvent(new CustomEvent(`vmcons:maquina`,{detail:e}))}})),e.querySelector(`#btn-refresh`).addEventListener(`click`,t),t(),e}async function Oe(e){let t=e.querySelector(`#vm-ip`),n=e.querySelector(`#vm-punto`),r=e.querySelector(`#vm-estado`),i=e.querySelector(`#btn-refresh`);if(t&&n&&r){i.disabled=!0,r.textContent=`Consultando…`,r.className=`text-vm-fgSecondary`,n.className=`w-1.5 h-1.5 rounded-full bg-vm-muted`;try{let e=await F.sesion.obtener(),i=L(e.estado);t.textContent=e.ip||`—`,n.className=`w-1.5 h-1.5 rounded-full ${i.punto}`,r.textContent=i.texto,r.className=e.estado===`running`?`text-vm-fg`:`text-vm-fgSecondary`,r.title=`Conector: ${e.conector}`}catch(e){t.textContent=`—`,n.className=`w-1.5 h-1.5 rounded-full bg-vm-accent`,r.textContent=`Sin sesión`,r.className=`text-vm-accent`,r.title=z(e)}finally{i.disabled=!1}}}var ke=[{id:`inicio`,etiqueta:`Inicio`,icono:`home`},{id:`paquetes`,etiqueta:`Paquetes`,icono:`package`},{id:`llaves`,etiqueta:`Llaves SSH`,icono:`key-round`},{id:`issue`,etiqueta:`/etc/issue`,icono:`file-text`},{id:`hostname`,etiqueta:`Hostname`,icono:`server`},{id:`usuarios`,etiqueta:`Usuarios`,icono:`users`},{id:`systemd`,etiqueta:`systemd`,icono:`activity`},{id:`haproxy`,etiqueta:`HAProxy`,icono:`git-merge`},{id:`historial`,etiqueta:`Historial`,icono:`history`}];function Ae(e=`inicio`){let t=document.createElement(`nav`);t.className=`bg-vm-surface border-b border-vm-border flex items-center gap-0.5 px-2`,t.setAttribute(`role`,`tablist`),t.setAttribute(`aria-label`,`Módulos de la consola`),t.innerHTML=ke.map(t=>`
    <button type="button" role="tab" class="tab ${t.id===e?`active`:``}" data-tab="${t.id}"
      aria-selected="${t.id===e}" tabindex="${t.id===e?0:-1}">
      <i data-lucide="${t.icono}"></i>
      <span>${t.etiqueta}</span>
    </button>
  `).join(``),k(t);let n=[...t.querySelectorAll(`.tab`)],r=e=>{window.location.hash=`#/${e}`},i=e=>{let t=n[(e+n.length)%n.length];n.forEach(e=>{e.tabIndex=e===t?0:-1}),t.focus(),r(t.dataset.tab)};return n.forEach((e,t)=>{e.addEventListener(`click`,()=>r(e.dataset.tab)),e.addEventListener(`keydown`,e=>{switch(e.key){case`ArrowRight`:e.preventDefault(),i(t+1);break;case`ArrowLeft`:e.preventDefault(),i(t-1);break;case`Home`:e.preventDefault(),i(0);break;case`End`:e.preventDefault(),i(n.length-1)}})}),t}function je(){let e=document.createElement(`section`);return e.className=`card flex flex-col h-full min-h-[280px]`,e.innerHTML=`
    <header class="px-3 py-1.5 border-b border-vm-border flex items-center justify-between bg-[#f7f7f7]">
      <div class="flex items-center gap-1.5"><i data-lucide="info" class="w-4 h-4"></i><span class="text-[12px] font-medium">Resultado</span></div>
      <span id="result-estado" class="text-[11px] text-vm-fgSecondary">En espera</span>
    </header>
    <div id="result-body" class="flex-1 p-3 flex items-center justify-center text-center tty">
      <div>
        <div class="text-[12px] tty-muted">Sin operaciones ejecutadas aún</div>
        <div class="text-[11px] tty-muted mt-1">Selecciona un módulo para ejecutar tareas sobre la máquina.</div>
      </div>
    </div>
    <footer class="px-3 py-1 border-t border-[#333] text-[10px] tty-muted font-mono tty">
      <span id="result-pie">Código de salida: -</span>
    </footer>
  `,e}function W({ok:e=!0,mensaje:t=``,operacion:n=``,codigo:r=0,salida:i=``,stderr:a=``,tecnico:o=``,solicitud:s=``}={}){let c=document.getElementById(`result-body`),l=document.getElementById(`result-estado`),u=document.getElementById(`result-pie`);c&&(c.className=`flex-1 p-3 overflow-auto tty`,c.innerHTML=`
    <div class="flex flex-col gap-2 fade-slide">
      <div class="flex items-start gap-2 px-2 py-1.5 border ${e?`border-[#3a3a3a] bg-[#242424]`:`border-vm-accent bg-[#2a1a1a]`}">
        <i data-lucide="${e?`check-circle-2`:`x-circle`}" class="w-4 h-4 shrink-0 ${e?`text-vm-success`:`text-vm-accent`}"></i>
        <span class="text-[12px] text-[#e6e6e6]">${I(t||(e?`Operación exitosa`:`Error en la operación`))}</span>
      </div>
      <details class="border border-[#3a3a3a]" ${e?``:`open`}>
        <summary class="px-2 py-1 cursor-pointer text-[11px] tty-muted">Detalle técnico</summary>
        <div class="px-2 py-2 flex flex-col gap-1 text-[11px]">
          <div><span class="tty-muted">Orden ejecutada:</span> <span class="font-mono break-all">${I(n)||`-`}</span></div>
          <div><span class="tty-muted">Código de salida:</span> <span class="font-mono">${I(r)}</span></div>
          <div><span class="tty-muted">Salida:</span></div>
          <pre class="bg-[#161616] border border-[#333] p-2 overflow-auto text-[11px] font-mono max-h-[140px] whitespace-pre-wrap text-[#d6d6d6]">${I(i)||`(vacío)`}</pre>
          <div><span class="tty-muted">stderr:</span></div>
          <pre class="bg-[#161616] border border-[#333] p-2 overflow-auto text-[11px] font-mono max-h-[140px] whitespace-pre-wrap text-[#d6d6d6]">${I(a)||`(vacío)`}</pre>
          ${o?`<div><span class="tty-muted">Diagnóstico:</span> <span class="font-mono break-all">${I(o)}</span></div>`:``}
          ${s?`<div><span class="tty-muted">Solicitud:</span> <span class="font-mono">${I(s)}</span></div>`:``}
        </div>
      </details>
    </div>
  `,l&&(l.textContent=e?`Correcto`:`Con error`,l.className=e?`text-[11px] text-vm-success`:`text-[11px] text-vm-accent`),u&&(u.textContent=`Código de salida: ${r??`-`}`),k(c))}var Me=[[`package`,`Paquetes`,`Gestión de paquetes APT/dpkg`,`#/paquetes`],[`key-round`,`Llaves SSH`,`Gestión de authorized_keys`,`#/llaves`],[`file-text`,`/etc/issue`,`Banner pre-login del sistema`,`#/issue`],[`server`,`Hostname`,`Identificador y FQDN del host`,`#/hostname`],[`users`,`Usuarios`,`Cuentas locales y privilegios`,`#/usuarios`],[`activity`,`systemd`,`Servicios, timers y unidades`,`#/systemd`],[`git-merge`,`HAProxy`,`Balanceo y configuración L4/L7`,`#/haproxy`],[`history`,`Historial`,`Registro de auditoría y tareas`,`#/historial`]];function Ne(){let e=document.createElement(`div`);return e.className=`grid grid-cols-[58%_42%] gap-vm-gap items-start`,e.innerHTML=`
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
        Solo se muestran las máquinas con Guest Additions y red puente/host-only (o apagadas, para poder encenderlas).
      </p>
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="layout-grid" class="w-4 h-4"></i><span class="text-[12px] font-medium">Módulos</span></div></header>
        <div class="p-3 grid grid-cols-2 gap-2">
          ${Me.map(([e,t,n,r])=>`
            <a href="${r}" class="border border-vm-border p-2 rounded-sm hover:border-[#C0392B] hover:bg-[#fdf5f4] transition-colors duration-100">
              <div class="flex items-center gap-1.5"><i data-lucide="${e}"></i><span class="text-[12px] font-medium">${t}</span></div>
              <div class="text-[11px] text-vm-fgSecondary mt-0.5">${n}</div>
            </a>
          `).join(``)}
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `,k(e),e.querySelector(`#sesion-refresh`).addEventListener(`click`,()=>K(e)),K(e),e}async function G(e){try{let{maquinas:t}=await F.maquinas.listar(),n=t.find(e=>e.activa);if(!n){await K(e);return}U({maquina:n,alRegistrar:()=>K(e)})}catch{await K(e)}}async function K(e){let t=e.querySelector(`#sesion-cuerpo`),n=e.querySelector(`#sesion-refresh`);if(t){n.disabled=!0,t.innerHTML=`
    <div class="text-vm-fgSecondary">Consultando la sesión con la máquina…</div>
    <div class="h-1 w-full bg-vm-bg border border-vm-border overflow-hidden mt-1">
      <div class="barra-indeterminada h-full bg-vm-accent"></div>
    </div>
  `;try{let n=await F.sesion.obtener(),r=L(n.estado),i=new Date().toLocaleTimeString(`es-CO`,{hour12:!1});t.innerHTML=`
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Máquina</span>
        <span class="font-mono">${I(n.vm)||`sin vincular`}</span>
      </div>
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Estado</span>
        <span class="px-1.5 py-0.5 border text-[11px] rounded-sm ${r.clase}">${r.texto}</span>
      </div>
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Conector</span>
        <span class="flex items-center gap-1"><i data-lucide="key-round" class="w-4 h-4"></i>${I(n.conector)}</span>
      </div>
      <div class="flex justify-between items-center">
        <span class="text-vm-fgSecondary">Dirección IP</span>
        <span class="flex items-center gap-1.5">
          <span class="font-mono">${I(n.ip)}</span>
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
    `,k(t),t.querySelector(`#copiar-ip`).addEventListener(`click`,()=>navigator.clipboard?.writeText(n.ip)),t.querySelector(`#registrar-sesion`)?.addEventListener(`click`,()=>G(e))}catch(n){t.innerHTML=`
      <div class="flex items-start gap-2 text-[12px]">
        <i data-lucide="alert-triangle" class="w-4 h-4 text-vm-accent shrink-0"></i>
        <div>
          <div class="text-vm-fg">No fue posible consultar la sesión</div>
          <div class="text-[11px] text-vm-fgSecondary mt-0.5">${I(z(n))}</div>
        </div>
      </div>
      <button id="abrir-registro" type="button" class="btn btn-primary self-start">
        <i data-lucide="key-round"></i> Registrar esta máquina
      </button>
    `,k(t),t.querySelector(`#abrir-registro`)?.addEventListener(`click`,()=>G(e))}finally{n.disabled=!1}}}var Pe=/^[a-z0-9][a-z0-9+.-]*$/,Fe=/^[a-z_][a-z0-9_-]{0,31}$/,Ie=/^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/,Le=/^[A-Za-z0-9_.@-]+\.service$/,Re=/^[0-7]{3,4}$/;function ze(e){return typeof e!=`string`||!Pe.test(e)?`El nombre del paquete solo admite minúsculas, dígitos y los signos + . -`:null}function Be(e){return typeof e!=`string`||!Fe.test(e)?`El nombre de usuario debe iniciar con minúscula o guion bajo y usar hasta 32 caracteres`:null}function Ve(e){return typeof e!=`string`||!Ie.test(e)?`El nombre de host debe cumplir RFC 1123: minúsculas, dígitos y guiones internos`:null}function He(e){return typeof e!=`string`||!Le.test(e)?`La unidad debe tener una extensión .service y usar letras, dígitos o _ . @ -`:null}function q(e){if(typeof e!=`string`)return!1;let t=e.split(`.`);return t.length===4&&t.every(e=>/^\d{1,3}$/.test(e)&&Number(e)<=255)}function Ue(e){return typeof e==`number`&&Number.isInteger(e)&&e>=1&&e<=65535}function We(e){return typeof e==`string`&&Re.test(e)}function Ge(e){return typeof e==`string`&&e.startsWith(`/`)&&e.length>1}function Ke(e,t){return`${e}:${t}`}function qe(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap items-start`,e.innerHTML=`
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
  `,k(e);let t=[],n=e.querySelector(`#paquetes-filas`),r=e.querySelector(`#paquetes-cuenta`),i=e.querySelector(`#paquetes-filtro`),a=()=>{let e=i.value.trim().toLowerCase(),a=e?t.filter(t=>t.nombre.toLowerCase().includes(e)):t;r.textContent=`${a.length} de ${t.length}`,a.length===0?n.innerHTML=B(3,t.length===0?`No hay paquetes para mostrar`:`Ningún paquete coincide con el filtro`):n.innerHTML=a.map(e=>`
        <tr class="border-t border-vm-border hover:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${I(e.nombre)}</td>
          <td class="px-3 py-1.5 font-mono text-vm-fgSecondary">${I(e.version)||`—`}</td>
          <td class="px-3 py-1.5">
            <span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${e.estado===`installed`?`text-vm-success border-vm-success bg-[#e8f5e9]`:`text-vm-fgSecondary border-vm-border bg-vm-bg`}">${I(e.estado)}</span>
          </td>
        </tr>
      `).join(``)},o=async()=>{n.innerHTML=B(3,`Consultando los paquetes…`);try{t=(await F.paquetes.listar()).paquetes??[],a()}catch(e){t=[],r.textContent=``,n.innerHTML=B(3,z(e),`text-vm-accent`)}};i.addEventListener(`input`,a),e.querySelector(`#paquetes-refresh`).addEventListener(`click`,o);let s=e.querySelector(`#form-paquetes`),c=e.querySelector(`#paquete-nombre`),l=e.querySelector(`#paquete-error`);return s.addEventListener(`submit`,async e=>{e.preventDefault();let t=s.elements.accion.value,n=c.value.trim(),r=ze(n);r?(l.textContent=r,l.classList.remove(`hidden`),c.focus()):(l.classList.add(`hidden`),V(s,!0),await H({accion:()=>t===`instalar`?F.paquetes.instalar(n):F.paquetes.eliminar(n),operacion:t===`instalar`?`apt-get install -y ${n}`:`apt-get remove -y ${n}`,informar:e=>{t===`eliminar`&&e.ok&&(e.mensaje=e.mensaje||`Se eliminó el paquete «${n}».`),W(e)},recargar:o}),V(s,!1))}),s.addEventListener(`reset`,()=>{l.classList.add(`hidden`),setTimeout(()=>c.focus(),0)}),o(),e}var J=`id_ed25519_vmcons`;function Je(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap items-start`,e.innerHTML=`
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="key-round" class="w-4 h-4"></i><span class="text-[12px] font-medium">Par de llaves SSH para root</span></div></header>
        <div class="p-3 flex flex-col gap-3">
          <div class="flex justify-between items-center text-[12px]">
            <span class="text-vm-fgSecondary">Estado</span>
            <span id="llave-estado" class="px-1.5 py-0.5 border text-[11px] rounded-sm text-vm-fgSecondary border-vm-border bg-vm-bg">sin verificar</span>
          </div>
          <p class="text-[11px] text-vm-fgSecondary">
            La generación crea el par en el anfitrión, instala la llave pública en
            <span class="font-mono">/root/.ssh/authorized_keys</span> y deshabilita
            el acceso por contraseña.
          </p>
          <div class="flex gap-2">
            <button id="llave-generar" class="btn btn-primary"><i data-lucide="settings"></i> Generar par de llaves</button>
            <button id="llave-descargar" class="btn"><i data-lucide="download"></i> Descargar llave privada</button>
          </div>
        </div>
      </section>
      <section class="card border border-vm-accent">
        <div class="p-3 flex gap-2">
          <i data-lucide="alert-triangle" class="text-vm-accent shrink-0"></i>
          <div class="text-[12px]">Guarda la llave privada en un lugar seguro y no la compartas; quien la posea tendrá acceso root a la máquina.</div>
        </div>
      </section>
    </div>
    <div id="result-slot"></div>
  `,k(e);let t=e.querySelector(`#llave-estado`),n=e.querySelector(`#llave-generar`),r=e.querySelector(`#llave-descargar`),i=(e,n)=>{t.textContent=e,t.className=`px-1.5 py-0.5 border text-[11px] rounded-sm `+(n===`ok`?`text-vm-success border-vm-success bg-[#e8f5e9]`:n===`error`?`text-vm-accent border-vm-accent bg-[#fdf1ef]`:`text-vm-fgSecondary border-vm-border bg-vm-bg`)};return n.addEventListener(`click`,async()=>{V(e,!0);let t=await H({accion:()=>F.llaves.generarRoot(),operacion:`ssh-keygen -t ed25519 + instalación en authorized_keys`,informar:W});V(e,!1),t?i(`generada`,`ok`):i(`sin verificar`,`neutro`)}),r.addEventListener(`click`,async()=>{V(e,!0);let t=await H({accion:async()=>(P(await F.llaves.descargarRoot(),J),{mensaje:`Se descargó la llave privada como «${J}».`}),operacion:`GET /api/keys/root`,informar:W});V(e,!1),i(t?`generada`:`no disponible`,t?`ok`:`error`)}),e}var Y=[[`nombre_host`,`Mostrar nombre del host`,`\\n`],[`fecha`,`Mostrar fecha`,`\\d`],[`hora`,`Mostrar hora`,`\\t`],[`ip`,`Mostrar dirección IP`,`\\4`]];function Ye(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap items-start`,e.innerHTML=`
    <section class="card">
      <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7] flex items-center justify-between">
        <div class="flex items-center gap-1.5"><i data-lucide="file-text" class="w-4 h-4"></i><span class="text-[12px] font-medium">Configuración de /etc/issue</span></div>
        <button id="issue-refresh" class="btn h-vm-btn px-1.5" title="Consultar de nuevo"><i data-lucide="refresh-cw" class="w-4 h-4"></i></button>
      </header>
      <form id="form-issue" class="p-3 flex flex-col gap-2" novalidate>
        <div class="flex flex-col gap-1">
          <label class="label" for="issue-mensaje">Mensaje personalizado</label>
          <textarea id="issue-mensaje" name="mensaje" class="input h-auto py-1 font-mono" rows="2" maxlength="512" placeholder="Debian GNU/Linux 13"></textarea>
          <span class="text-[11px] text-vm-fgSecondary">Hasta 512 caracteres. No se admite el carácter nulo.</span>
        </div>
        <div class="flex flex-col gap-1 text-[12px]">
          ${Y.map(([e,t])=>`<label class="flex items-center gap-1.5"><input type="checkbox" name="${e}"> ${t}</label>`).join(``)}
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
  `,k(e);let t=e.querySelector(`#form-issue`),n=e.querySelector(`#issue-mensaje`),r=e.querySelector(`#issue-preview`),i={mensaje:``,nombre_host:!1,fecha:!1,hora:!1,ip:!1,contenido:``},a=()=>({mensaje:n.value,nombre_host:t.elements.nombre_host.checked,fecha:t.elements.fecha.checked,hora:t.elements.hora.checked,ip:t.elements.ip.checked}),o=()=>{let e=a(),t=Y.filter(([t])=>e[t]).map(([,,e])=>e),n=[];e.mensaje.trim()&&n.push(e.mensaje),t.length>0&&n.push(t.join(` `)),r.textContent=n.length>0?n.join(`
`):`(banner vacío)`},s=e=>{i={...i,...e},n.value=e.mensaje??``,t.elements.nombre_host.checked=!!e.nombre_host,t.elements.fecha.checked=!!e.fecha,t.elements.hora.checked=!!e.hora,t.elements.ip.checked=!!e.ip,e.contenido?r.textContent=e.contenido:o()},c=async()=>{V(t,!0),r.textContent=`Consultando…`;try{s(await F.issue.obtener())}catch(e){r.textContent=z(e)}finally{V(t,!1)}};return t.addEventListener(`input`,o),t.addEventListener(`change`,o),e.querySelector(`#issue-refresh`).addEventListener(`click`,c),e.querySelector(`#issue-restablecer`).addEventListener(`click`,()=>s(i)),t.addEventListener(`submit`,async e=>{e.preventDefault(),V(t,!0);let n=a(),r=await H({accion:async()=>(await F.issue.reemplazar(n),{mensaje:`Se actualizó el banner de /etc/issue.`}),operacion:`PUT /api/issue`,informar:W});V(t,!1),r&&(i={...n,contenido:``},c())}),c(),e}function Xe(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap items-start`,e.innerHTML=`
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
  `,k(e);let t=e.querySelector(`#form-hostname`),n=e.querySelector(`#hostname-nombre`),r=e.querySelector(`#hostname-ip`),i=e.querySelector(`#hostname-error-nombre`),a=e.querySelector(`#hostname-error-ip`),o={nombre:``,ip:``,actualizar_hosts:!0},s=(e,t)=>{e.textContent=t??``,e.classList.toggle(`hidden`,!t)},c=async()=>{V(t,!0);try{let e=await F.hostname.obtener();o={...o,...e},n.value=e.nombre??``,r.value=e.ip??``,t.elements.actualizar_hosts.checked=e.actualizar_hosts!==!1}catch(e){s(i,z(e))}finally{V(t,!1)}};return t.addEventListener(`input`,()=>{s(i,null),s(a,null)}),e.querySelector(`#hostname-refresh`).addEventListener(`click`,c),e.querySelector(`#hostname-restablecer`).addEventListener(`click`,()=>{n.value=o.nombre??``,r.value=o.ip??``,t.elements.actualizar_hosts.checked=o.actualizar_hosts!==!1,s(i,null),s(a,null)}),t.addEventListener(`submit`,async e=>{e.preventDefault();let o=Ve(n.value.trim()),l=q(r.value.trim())?null:`La dirección debe ser una IPv4 válida, por ejemplo 192.168.1.105`;if(s(i,o),s(a,l),o){n.focus();return}if(l){r.focus();return}let u={nombre:n.value.trim(),ip:r.value.trim(),actualizar_hosts:t.elements.actualizar_hosts.checked};V(t,!0),await H({accion:async()=>(await F.hostname.reemplazar(u),{mensaje:`Se aplicó el nombre de host «${u.nombre}».`}),operacion:`PUT /api/hostname`,informar:W,recargar:c}),V(t,!1)}),c(),e}function Ze(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap items-start`,e.innerHTML=`
    <div class="flex flex-col gap-vm-gap">
      <section class="card">
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="user-plus" class="w-4 h-4"></i><span class="text-[12px] font-medium">Crear usuario</span></div></header>
        <form id="form-crear" class="p-3 flex flex-col gap-2" novalidate>
          <div class="flex flex-col gap-1">
            <label class="label" for="usuario-nombre">Nombre de usuario</label>
            <input id="usuario-nombre" name="nombre" class="input font-mono" placeholder="jorge" autocomplete="off" spellcheck="false">
            <span id="usuario-error" class="text-[11px] text-vm-accent hidden"></span>
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
        <header class="px-3 py-1.5 border-b border-vm-border bg-[#f7f7f7]"><div class="flex items-center gap-1.5"><i data-lucide="download" class="w-4 h-4"></i><span class="text-[12px] font-medium">Descarga de llaves</span></div></header>
        <div class="p-3 flex items-center gap-2">
          <select id="descarga-nombre" class="input w-[200px]"></select>
          <button id="descarga-boton" class="btn"><i data-lucide="download"></i> Descargar llave privada</button>
        </div>
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
  `,k(e);let t=[],n=e.querySelector(`#usuarios-filas`),r=e.querySelector(`#usuarios-cuenta`),i=e.querySelector(`#eliminar-nombre`),a=e.querySelector(`#descarga-nombre`),o=e.querySelector(`#form-crear`),s=e.querySelector(`#form-eliminar`),c=(e,n)=>{let r=e.value;e.innerHTML=`<option value="">${n}</option>`+t.map(e=>`<option value="${I(e.nombre)}">${I(e.nombre)}</option>`).join(``),t.some(e=>e.nombre===r)&&(e.value=r)},l=()=>{r.textContent=String(t.length),c(i,t.length?`Seleccionar usuario`:`Sin usuarios`),c(a,t.length?`Seleccionar usuario`:`Sin usuarios`),t.length===0?n.innerHTML=B(6,`No hay usuarios para mostrar`):(n.innerHTML=t.map(e=>`
        <tr class="border-t border-vm-border hover:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${I(e.nombre)}</td>
          <td class="px-3 py-1.5">${e.sudo?`<i data-lucide="check" class="w-4 h-4 text-vm-success"></i>`:`<span class="text-vm-fgSecondary">—</span>`}</td>
          <td class="px-3 py-1.5">${e.llaves?`<i data-lucide="key-round" class="w-4 h-4 text-vm-fg"></i>`:`<span class="text-vm-fgSecondary">—</span>`}</td>
          <td class="px-3 py-1.5 font-mono text-vm-fgSecondary">${I(e.uid)||`—`}</td>
          <td class="px-3 py-1.5 font-mono text-vm-fgSecondary">${I(e.home)||`—`}</td>
          <td class="px-3 py-1.5 text-right whitespace-nowrap">
            ${e.llaves?`<button class="btn h-vm-btn px-1" data-descargar="${I(e.nombre)}" title="Descargar llave privada"><i data-lucide="download" class="w-4 h-4"></i></button>`:``}
            <button class="btn h-vm-btn px-1" data-eliminar="${I(e.nombre)}" title="Eliminar usuario"><i data-lucide="trash-2" class="w-4 h-4"></i></button>
          </td>
        </tr>
      `).join(``),k(n))},u=async()=>{n.innerHTML=B(6,`Consultando los usuarios…`);try{t=(await F.usuarios.listar()).usuarios??[],l()}catch(e){t=[],r.textContent=``,n.innerHTML=B(6,z(e),`text-vm-accent`)}},d=async e=>{await H({accion:async()=>(P(await F.usuarios.descargarLlave(e),`id_ed25519_${e}`),{mensaje:`Se descargó la llave privada de «${e}».`}),operacion:`GET /api/users/${e}/key`,informar:W})},f=async(e,t)=>{e&&await H({accion:async()=>(await F.usuarios.eliminar(e,{archivos:t}),{mensaje:`Se eliminó el usuario «${e}».`}),operacion:`DELETE /api/users/${e}${t?`?files=true`:``}`,informar:W,recargar:u})};n.addEventListener(`click`,e=>{let t=e.target.closest(`[data-descargar]`),n=e.target.closest(`[data-eliminar]`);t&&d(t.dataset.descargar),n&&window.confirm(`¿Eliminar el usuario «${n.dataset.eliminar}»?`)&&f(n.dataset.eliminar,!1)}),e.querySelector(`#usuarios-refresh`).addEventListener(`click`,u),e.querySelector(`#descarga-boton`).addEventListener(`click`,()=>d(a.value)),s.addEventListener(`submit`,e=>{e.preventDefault();let t=i.value;t?window.confirm(`¿Eliminar el usuario «${t}»?`)&&f(t,s.elements.archivos.checked):W({ok:!1,mensaje:`Selecciona el usuario que se va a eliminar.`,operacion:`DELETE /api/users/{name}`})});let p=e.querySelector(`#usuario-nombre`),m=e.querySelector(`#usuario-error`),h=e.querySelector(`#usuario-descargar-nueva`),g=``;return o.addEventListener(`input`,()=>{m.classList.add(`hidden`),h.classList.add(`hidden`)}),o.addEventListener(`submit`,async e=>{e.preventDefault();let t=p.value.trim(),n=Be(t);if(n){m.textContent=n,m.classList.remove(`hidden`),p.focus();return}let r={nombre:t,sudo:o.elements.sudo.checked,llaves:o.elements.llaves.checked};V(o,!0);let i=await H({accion:async()=>(await F.usuarios.crear(r),{mensaje:`Se creó el usuario «${t}».`}),operacion:`useradd ${t}${r.sudo?` + grupo sudo`:``}${r.llaves?` + par de llaves`:``}`,informar:W,recargar:u});V(o,!1),i&&(g=r.llaves?t:``,h.classList.toggle(`hidden`,!r.llaves),o.reset())}),h.addEventListener(`click`,()=>d(g)),u(),e}function Qe(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap items-start`,e.innerHTML=`
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
  `,k(e);let t=[],n=e.querySelector(`#archivos`),r=e.querySelector(`#form-despliegue`),i=e.querySelector(`#unidad-seleccion`),a=e.querySelector(`#unidad-estado`),o=e.querySelector(`#unidades-filas`),s=e.querySelector(`#unidades-cuenta`),c=(e={})=>{let t=document.createElement(`div`);t.className=`grid grid-cols-[1fr_1fr_72px_28px] gap-1 items-center`,t.innerHTML=`
      <input class="input font-mono" data-campo="origen" placeholder="./app/catalogo" value="${I(e.origen)}" spellcheck="false">
      <input class="input font-mono" data-campo="destino" placeholder="/opt/catalogo/app" value="${I(e.destino)}" spellcheck="false">
      <input class="input font-mono" data-campo="modo" placeholder="755" value="${I(e.modo)||`755`}" maxlength="4" spellcheck="false">
      <button type="button" class="btn h-vm-btn px-1" data-quitar title="Quitar el archivo"><i data-lucide="x" class="w-4 h-4"></i></button>
    `,t.querySelector(`[data-quitar]`).addEventListener(`click`,()=>{n.children.length>1&&t.remove()}),n.appendChild(t),k(t)},l=e=>{if(!e){a.textContent=`sin datos`,a.className=`px-1.5 py-0.5 border text-[11px] rounded-sm text-vm-fgSecondary border-vm-border bg-vm-bg`;return}let t=R(e.estado);a.textContent=t.texto,a.className=`px-1.5 py-0.5 border text-[11px] rounded-sm ${t.clase}`,a.title=`activa: ${e.activo} · habilitada: ${e.habilitado}`},u=()=>{s.textContent=String(t.length),i.innerHTML=t.length?t.map(e=>`<option value="${I(e.unidad)}">${I(e.unidad)}</option>`).join(``):`<option value="">Sin unidades</option>`,t.length===0?(o.innerHTML=B(3,`No hay unidades para mostrar`),l(null)):(o.innerHTML=t.map(e=>{let t=R(e.estado);return`
        <tr class="border-t border-vm-border hover:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${I(e.unidad)}</td>
          <td class="px-3 py-1.5"><span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${t.clase}">${t.texto}</span></td>
          <td class="px-3 py-1.5">${e.habilitado?`<i data-lucide="check" class="w-4 h-4 text-vm-success"></i>`:`<span class="text-vm-fgSecondary">—</span>`}</td>
        </tr>
      `}).join(``),k(o),l(t[0]))},d=async()=>{o.innerHTML=B(3,`Consultando las unidades…`);try{t=(await F.unidades.listar()).unidades??[],u(),f()}catch(e){t=[],s.textContent=``,o.innerHTML=B(3,z(e),`text-vm-accent`),l(null)}},f=async()=>{let e=i.value;if(e)try{l(await F.unidades.obtener(e))}catch{l(null)}};return e.querySelector(`#agregar-archivo`).addEventListener(`click`,()=>c()),e.querySelector(`#unidades-refresh`).addEventListener(`click`,d),i.addEventListener(`change`,f),e.querySelectorAll(`[data-accion]`).forEach(t=>{t.addEventListener(`click`,async()=>{let n=i.value;if(!n){W({ok:!1,mensaje:`Selecciona una unidad para operar.`,operacion:`systemctl`});return}let r=t.dataset.accion;V(e,!0);let a=await H({accion:async()=>{switch(r){case`iniciar`:return await F.unidades.cambiarEstado(n,{activo:!0}),{mensaje:`Se activó la unidad «${n}».`};case`detener`:return await F.unidades.cambiarEstado(n,{activo:!1}),{mensaje:`Se desactivó la unidad «${n}».`};case`habilitar`:return await F.unidades.cambiarEstado(n,{habilitado:!0}),{mensaje:`Se habilitó el arranque de «${n}».`};case`deshabilitar`:return await F.unidades.cambiarEstado(n,{habilitado:!1}),{mensaje:`Se deshabilitó el arranque de «${n}».`};case`reiniciar`:return await F.unidades.reiniciar(n),{mensaje:`Se reinició la unidad «${n}».`};default:return F.unidades.obtener(n)}},operacion:`systemctl ${r} ${n}`,informar:W,recargar:r===`estado`?void 0:d});V(e,!1),r===`estado`&&a&&l(await F.unidades.obtener(n))})}),r.addEventListener(`submit`,async t=>{t.preventDefault();let i=e.querySelector(`#unidad-nombre`).value.trim(),a=e.querySelector(`#unidad-error`),o=He(i);if(o){a.textContent=o,a.classList.remove(`hidden`);return}a.classList.add(`hidden`);let s=[...n.querySelectorAll(`[data-campo="origen"]`)].map(e=>{let t=e.parentElement;return{origen:e.value.trim(),destino:t.querySelector(`[data-campo="destino"]`).value.trim(),modo:t.querySelector(`[data-campo="modo"]`).value.trim()}});s.find(e=>!e.origen||!Ge(e.destino)||!We(e.modo))?W({ok:!1,mensaje:`Revisa los archivos: el origen es obligatorio, el destino debe ser una ruta absoluta y el modo octal de tres o cuatro dígitos.`,operacion:`PUT /api/units/${i}`}):(V(r,!0),await H({accion:async()=>(await F.unidades.desplegar(i,{archivos:s}),{mensaje:`Se desplegó la unidad «${i}» con ${s.length} archivo(s).`}),operacion:`PUT /api/units/${i}`,informar:W,recargar:d}),V(r,!1))}),c(),d(),e}function $e(){let e=document.createElement(`div`);e.className=`grid grid-cols-[58%_42%] gap-vm-gap items-start`,e.innerHTML=`
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
  `,k(e);let t=[],n=e.querySelector(`#backend-filas`),r=e.querySelector(`#backend-cuenta`),i=e.querySelector(`#form-backend`),a=()=>{r.textContent=String(t.length),t.length===0?n.innerHTML=B(4,`El back-end no tiene servidores`):(n.innerHTML=t.map(e=>{let t=Ce(e.estado),n=Ke(e.direccion,e.puerto);return`
        <tr class="border-t border-vm-border hover:bg-vm-bg">
          <td class="px-3 py-1.5 font-mono">${I(e.direccion)}</td>
          <td class="px-3 py-1.5 font-mono">${I(e.puerto)}</td>
          <td class="px-3 py-1.5"><span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${t.clase}">${t.texto}</span></td>
          <td class="px-3 py-1.5 text-right"><button class="btn h-vm-btn px-1" data-eliminar="${I(n)}" title="Quitar del back-end"><i data-lucide="trash-2" class="w-4 h-4"></i></button></td>
        </tr>
      `}).join(``),k(n))},o=async()=>{n.innerHTML=B(4,`Consultando los servidores…`);try{t=(await F.haproxy.listar()).servidores??[],a()}catch(e){t=[],r.textContent=``,n.innerHTML=B(4,z(e),`text-vm-accent`)}},s=async e=>{window.confirm(`¿Quitar ${e} del back-end?`)&&await H({accion:async()=>(await F.haproxy.eliminar(e),{mensaje:`Se quitó ${e} del back-end.`}),operacion:`DELETE /api/haproxy/backend/servers/${e}`,informar:W,recargar:o})};return n.addEventListener(`click`,e=>{let t=e.target.closest(`[data-eliminar]`);t&&s(t.dataset.eliminar)}),e.querySelector(`#backend-refresh`).addEventListener(`click`,o),i.addEventListener(`submit`,async e=>{e.preventDefault();let t=i.elements.direccion.value.trim(),n=Number(i.elements.puerto.value.trim());if(!q(t)){W({ok:!1,mensaje:`La dirección debe ser una IPv4 válida, por ejemplo 192.168.1.111.`,operacion:`POST /api/haproxy/backend/servers`});return}if(!Ue(n)){W({ok:!1,mensaje:`El puerto debe ser un número entre 1 y 65535.`,operacion:`POST /api/haproxy/backend/servers`});return}V(i,!0);let r=await H({accion:async()=>(await F.haproxy.agregar({direccion:t,puerto:n}),{mensaje:`Se agregó ${t}:${n} al back-end.`}),operacion:`POST /api/haproxy/backend/servers ${t}:${n}`,informar:W,recargar:o});V(i,!1),r&&i.reset()}),o(),e}var et=[`paquetes`,`llaves`,`issue`,`hostname`,`usuarios`,`systemd`,`haproxy`];function X(){let e=document.createElement(`div`);e.className=`flex flex-col gap-vm-gap`,e.innerHTML=`
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
          ${et.map(e=>`<option value="${e}">${e}</option>`).join(``)}
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
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">OPERACIÓN</th>
              <th class="text-left px-3 py-1 font-medium text-vm-fgSecondary">RESULTADO</th>
            </tr>
          </thead>
          <tbody id="historial-filas">
            <tr><td colspan="5" class="px-3 py-6 text-center text-vm-fgSecondary">Consultando…</td></tr>
          </tbody>
        </table>
      </div>
      <footer class="px-3 py-1.5 border-t border-vm-border text-[11px] text-vm-fgSecondary flex items-center justify-between">
        <span id="historial-estado">Consultando…</span>
        <span id="historial-total" class="font-mono">0</span>
      </footer>
    </section>
  `,k(e);let t=e.querySelector(`#filtros-historial`),n=e.querySelector(`#historial-filas`),r=e.querySelector(`#historial-estado`),i=e.querySelector(`#historial-total`),a=e.querySelector(`#exportar-historial`),o=[],s=()=>{let e=new FormData(t),n={};for(let t of[`q`,`desde`,`hasta`,`hora_desde`,`hora_hasta`,`vm`,`modulo`]){let r=String(e.get(t)||``).trim();r&&(n[t]=r)}return n},c=e=>{let t=new Date(e);return Number.isNaN(t.getTime())?e:t.toLocaleString(`es-CO`)},l=()=>{o.length===0?(n.innerHTML=`
        <tr><td colspan="5" class="px-3 py-6 text-center text-vm-fgSecondary">
          No hay operaciones que coincidan con los filtros.
        </td></tr>
      `,a.disabled=!0):(n.innerHTML=o.map(e=>`
        <tr class="border-t border-vm-border hover:bg-[#faf9f9]">
          <td class="px-3 py-1 font-mono text-[11px] whitespace-nowrap">${I(c(e.fecha_hora))}</td>
          <td class="px-3 py-1 font-mono text-[11px]">${I(e.vm||`—`)}</td>
          <td class="px-3 py-1">${I(e.modulo)}</td>
          <td class="px-3 py-1 font-mono text-[11px]">${I(e.operacion)}</td>
          <td class="px-3 py-1">
            <span class="px-1.5 py-0.5 border text-[10px] rounded-sm ${e.resultado===`ok`?`border-vm-success text-vm-success`:`border-vm-accent text-vm-accent`}">
              ${e.resultado===`ok`?`OK`:`Error`}
            </span>
          </td>
        </tr>
      `).join(``),a.disabled=!1)},u=async()=>{r.textContent=`Consultando…`,r.className=`text-vm-fgSecondary`;try{o=(await F.historial.listar(s())).operaciones??[],l(),r.textContent=o.length===0?`Sin operaciones`:`Historial actualizado`,i.textContent=String(o.length)}catch(e){o=[],n.innerHTML=`
        <tr><td colspan="5" class="px-3 py-6 text-center text-vm-accent">${I(z(e))}</td></tr>
      `,r.textContent=`No fue posible consultar el historial`,r.className=`text-vm-accent`,i.textContent=`0`,a.disabled=!0}};return t.addEventListener(`submit`,e=>{e.preventDefault(),u()}),e.querySelector(`#limpiar-historial`).addEventListener(`click`,()=>{t.reset(),u()}),a.addEventListener(`click`,()=>{let e=o.map(e=>[e.fecha_hora,e.vm,e.modulo,e.operacion,e.resultado,e.detalle].map(e=>`"${String(e??``).replaceAll(`"`,`""`)}"`).join(`,`)).join(`
`),t=document.createElement(`a`);t.href=URL.createObjectURL(new Blob([`fecha_hora,vm,modulo,operacion,resultado,detalle
`+e],{type:`text/csv;charset=utf-8`})),t.download=`historial-vmcons.csv`,t.click(),URL.revokeObjectURL(t.href)}),u(),e}var Z={inicio:Ne,paquetes:qe,llaves:Je,issue:Ye,hostname:Xe,usuarios:Ze,systemd:Qe,haproxy:$e,historial:X},tt=`inicio`,Q=null;function nt(){let e=window.location.hash.replace(/^#\/?/,``);return Object.hasOwn(Z,e)?e:tt}function rt(){let e=document.getElementById(`app`);e.innerHTML=``;let t=document.createElement(`div`),n=document.createElement(`main`);return n.className=`max-w-vm mx-auto px-3 py-3`,e.append(De(),t,n),Q={barra:t,principal:n},Q}function $(){let{barra:e,principal:t}=Q??rt(),n=nt();e.replaceChildren(Ae(n));let r=Z[n]();r.classList.add(`fade-slide`),t.replaceChildren(r);let i=document.getElementById(`result-slot`);if(i&&!document.getElementById(`resultado-global`)){let e=je();e.id=`resultado-global`,i.appendChild(e)}}window.addEventListener(`hashchange`,$),document.addEventListener(`vmcons:maquina`,$),$();