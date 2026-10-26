#!/usr/bin/env python3
# Arnés de las pruebas de campo P01-P18.
#
# Requisitos: consola escuchando en 127.0.0.1:8093 con una copia de data/,
# cuenta jorge/pruebas-2026, VMs VM1 (.16), VM2 (.18) y balanceador (.21)
# encendidas, y sshpass instalado. El balanceador necesita un backend
# temporal y VM2 un servidor HTTP en el puerto 8080 para P15/P16.
#
# Uso: python3 scripts/pruebas-campo.py
# Escribe results.json y results.md en el directorio de trabajo.
import base64, json, os, shutil, sqlite3, subprocess, time
import requests

BASE=os.environ.get('VMCONS_URL','http://127.0.0.1:8093')
KEY=os.environ.get('VMCONS_KEY','/tmp/opencode/e2e/data/id_ed25519')
DB=os.environ.get('VMCONS_DB','/tmp/opencode/e2e/data/vmcons.db')
USER=os.environ.get('VMCONS_USER','jorge')
PASS=os.environ.get('VMCONS_PASS','pruebas-2026')
S=requests.Session(); S.trust_env=False
login=S.post(BASE+'/api/auth/login',json={'usuario':USER,'password':PASS},timeout=30)
assert login.status_code==200, login.text
results=[]

def api(method,path,**kw):
    kw.setdefault('timeout',180)
    r=S.request(method,BASE+path,**kw)
    print(f'{method} {path} -> {r.status_code} {r.text[:160]}')
    return r

def ssh(ip,cmd,key=KEY,timeout=30):
    p=subprocess.run(['ssh','-i',key,'-o','BatchMode=yes','-o','StrictHostKeyChecking=no',
                      '-o','ConnectTimeout=5',f'root@{ip}',cmd],capture_output=True,text=True,timeout=timeout)
    return p.returncode,p.stdout.strip(),p.stderr.strip()

def link(vm):
    r=api('PUT','/api/session',json={'vm':vm}); assert r.status_code==200,(r.status_code,r.text)
    return r.json()

def record(test,before,operation,after,ok,notes=''):
    results.append({'test':test,'before':before,'operation':operation,'after':after,'ok':ok,'notes':notes})
    print(('PASS' if ok else 'FAIL'),test)

# ---------- P01/P02 registro y diagnóstico de la máquina ----------
link('VM2')
antes=api('GET','/api/session/diagnostico')
r=api('DELETE','/api/session/register')
rc,after_reg,_=ssh('192.168.1.18','test -f /root/.ssh/authorized_keys; echo authorized=$?')
rreg=api('POST','/api/session/register',json={'usuario':'jorge','password':'1234'})
despues=api('GET','/api/session/diagnostico')
registrada=api('GET','/api/session').json()
record('P01 registrar la máquina','VM2 sin registrar (registro eliminado)',
       f'POST /api/session/register jorge -> {rreg.status_code}',
       f"registrada={registrada.get('registrada')} guest_additions={despues.json().get('guest_additions_version')}",
       rreg.status_code==201 and registrada.get('registrada') is True and rc==0, rreg.text[:250])
record('P02 diagnóstico de la máquina','registro previo de VM2',
       f'GET /api/session/diagnostico -> {antes.status_code}' if antes.status_code==200 else 'GET /api/session/diagnostico -> 503',
       json.dumps(despues.json(),ensure_ascii=False)[:260],
       despues.status_code==200 and despues.json().get('ssh_abierto') is True and 'guest_additions_version' in despues.json(),
       'Se conservan los avisos de Guest Additions sin ocultarlos')

# ---------- P03/P04/P05 paquetes ----------
link('VM1')
rc,before,_=ssh('192.168.1.16',"dpkg-query -W -f='${Status}' tree 2>&1")
r=api('POST','/api/packages',json={'nombre':'tree'})
rc2,after,_=ssh('192.168.1.16',"dpkg-query -W -f='${Status}' tree 2>&1")
record('P03 instalar paquete',before,f'POST /api/packages tree -> {r.status_code}',after,
       r.status_code==201 and 'install ok installed' in after, r.text[:300])

r=api('DELETE','/api/packages/tree')
rc3,after2,_=ssh('192.168.1.16',"dpkg-query -W -f='${Status}' tree 2>&1")
record('P04 eliminar paquete','install ok installed' in after,f'DELETE /api/packages/tree -> {r.status_code}',after2,
       r.status_code==204 and 'no packages found' in after2, r.text[:300])

r=api('POST','/api/packages',json={'nombre':'paquete-inexistente'})
record('P05 error de APT','tree eliminado',f'POST /api/packages paquete-inexistente -> {r.status_code}',
       r.text[:400],r.status_code==500 and 'E:' in r.text, 'Se conserva código y stderr')

# ---------- P08 /etc/issue ----------
rc,raw_issue,_=ssh('192.168.1.16','base64 -w0 /etc/issue')
antes=api('GET','/api/issue').json()
cuerpo={'mensaje':'Bienvenido a vmcons - pruebas automaticas','nombre_host':True,'fecha':True,'hora':True,'ip':True}
r=api('PUT','/api/issue',json=cuerpo)
rc,issue_txt,_=ssh('192.168.1.16','cat /etc/issue')
record('P08 /etc/issue',antes.get('contenido',''),f'PUT /api/issue -> {r.status_code}',issue_txt,
       r.status_code==200 and 'Bienvenido a vmcons' in issue_txt and '\\n' in issue_txt, r.text[:300])
# restaurar el archivo original
subprocess.run(['ssh','-i',KEY,'-o','BatchMode=yes','-o','StrictHostKeyChecking=no','root@192.168.1.16',
                f"printf %s '{raw_issue}' | base64 -d > /etc/issue"],check=True)

# ---------- P10/P11 usuarios ----------
antes=ssh('192.168.1.16','id alumno 2>&1; ls -ld /home/alumno 2>&1')[1]
r=api('POST','/api/users',json={'nombre':'alumno','sudo':True,'llaves':True})
rc,after,_=ssh('192.168.1.16','id alumno; groups alumno; passwd -S alumno; ls -l /home/alumno/.ssh; stat -c "%a %U" /home/alumno/.ssh /home/alumno/.ssh/authorized_keys 2>&1')
rkey=api('GET','/api/users/alumno/key')
record('P10 crear usuario',antes,f'POST /api/users alumno -> {r.status_code}',after,
       r.status_code==201 and 'sudo' in after and 'alumno' in after and rkey.status_code==200, r.text[:250])
r=api('DELETE','/api/users/alumno?files=true')
rc,after2,_=ssh('192.168.1.16','id alumno 2>&1; ls -ld /home/alumno 2>&1')
record('P11 eliminar usuario','alumno existente con home',f'DELETE /api/users/alumno?files=true -> {r.status_code}',after2,
       r.status_code==204 and 'no such user' in after2 and 'No such file' in after2, r.text[:250])

# ---------- P12/P13/P14 systemd ----------
os.makedirs('/tmp/opencode/e2e/unit',exist_ok=True)
script='/tmp/opencode/e2e/unit/vmcons-demo.sh'
unit='/tmp/opencode/e2e/unit/vmcons-demo.service'
open(script,'w').write('#!/bin/sh\nwhile true; do date -Is >> /tmp/vmcons-demo.log; sleep 2; done\n')
open(unit,'w').write('[Unit]\nDescription=vmcons demo\n[Service]\nType=simple\nExecStart=/usr/local/bin/vmcons-demo.sh\n[Install]\nWantedBy=multi-user.target\n')
payload={'archivos':[{'origen':script,'destino':'/usr/local/bin/vmcons-demo.sh','modo':'755'},
                     {'origen':unit,'destino':'/etc/systemd/system/vmcons-demo.service','modo':'644'}],
         'usuario':'root','grupo':'root'}
r=api('PUT','/api/units/vmcons-demo.service',json=payload)
rc,after,_=ssh('192.168.1.16','systemctl cat vmcons-demo.service; systemctl is-enabled vmcons-demo.service 2>&1')
record('P12 desplegar unidad','unidad inexistente',f'PUT /api/units/vmcons-demo.service -> {r.status_code}',after,
       r.status_code==200 and 'vmcons-demo' in after, r.text[:250])

r=api('PATCH','/api/units/vmcons-demo.service',json={'activo':True})
time.sleep(2)
rc,act,_=ssh('192.168.1.16','systemctl is-active vmcons-demo.service')
r2=api('PATCH','/api/units/vmcons-demo.service',json={'habilitado':True})
rc,en,_=ssh('192.168.1.16','systemctl is-enabled vmcons-demo.service')
r3=api('POST','/api/units/vmcons-demo.service/restarts')
time.sleep(2)
rc,log,_=ssh('192.168.1.16','wc -l < /tmp/vmcons-demo.log; systemctl is-active vmcons-demo.service')
r4=api('PATCH','/api/units/vmcons-demo.service',json={'activo':False})
rc,act2,_=ssh('192.168.1.16','systemctl is-active vmcons-demo.service')
record('P13 ciclo de vida unidad','inactiva/deshabilitada',
       f'PATCH activo/enable, restarts, PATCH stop -> {r.status_code}/{r2.status_code}/{r3.status_code}/{r4.status_code}',
       f'active={act} enabled={en} log={log} after_stop={act2}',
       r.status_code==200 and r2.status_code==200 and r3.status_code==202 and act=='active' and en=='enabled' and act2=='inactive', log)

# P14 script de la precondición e): el log ya se incrementó al iniciar/reiniciar
rc,log,_=ssh('192.168.1.16','cat /tmp/vmcons-demo.log | tail -3')
record('P14 script con fecha y hora','sin ejecuciones',f'start + restart -> {r3.status_code}',log,
       'T' in log and len(log.splitlines())>=1, 'El script agrega la fecha en cada ejecución')
# limpiar unidad
api('PATCH','/api/units/vmcons-demo.service',json={'activo':False})
api('PATCH','/api/units/vmcons-demo.service',json={'habilitado':False})
ssh('192.168.1.16','systemctl stop vmcons-demo.service 2>/dev/null; systemctl disable vmcons-demo.service 2>/dev/null; rm -f /etc/systemd/system/vmcons-demo.service /usr/local/bin/vmcons-demo.sh /tmp/vmcons-demo.log; systemctl daemon-reload; true')

# ---------- P15/P16 HAProxy ----------
link('balanceador')
antes=api('GET','/api/haproxy/backend/servers').json()
r=api('POST','/api/haproxy/backend/servers',json={'direccion':'192.168.1.18','puerto':8080})
rc,cfg,_=ssh('192.168.1.21','grep -n "192.168.1.18:8080" /etc/haproxy/haproxy.cfg; haproxy -c -f /etc/haproxy/haproxy.cfg')
curl=subprocess.run(['curl','-s','-o','/dev/null','-w','%{http_code}','http://192.168.1.21/'],capture_output=True,text=True)
lista=api('GET','/api/haproxy/backend/servers').json()
record('P15 agregar servidor al back-end','sin servidores',f'POST /api/haproxy/backend/servers -> {r.status_code}',
       f'{cfg} curl_balancer={curl.stdout}', r.status_code==201 and '192.168.1.18:8080' in cfg and curl.stdout in ('200','503'),
       json.dumps(lista)[:300])
r=api('DELETE','/api/haproxy/backend/servers/192.168.1.18:8080')
rc,cfg2,_=ssh('192.168.1.21','grep -c "192.168.1.18:8080" /etc/haproxy/haproxy.cfg; haproxy -c -f /etc/haproxy/haproxy.cfg')
curl2=subprocess.run(['curl','-s','-o','/dev/null','-w','%{http_code}','http://192.168.1.21/'],capture_output=True,text=True)
rdup=api('DELETE','/api/haproxy/backend/servers/192.168.1.18:8080')
record('P16 eliminar servidor del back-end','servidor .18:8080 presente',f'DELETE /api/haproxy/... -> {r.status_code}',
       f'lineas={cfg2} curl={curl2.stdout} repetir={rdup.status_code}',
       r.status_code==204 and cfg2.splitlines()[0]=='0' and rdup.status_code==404, '')

# ---------- P18 cambio de máquina ----------
link('VM1'); s1=api('GET','/api/session').json()
link('VM2'); s2=api('GET','/api/session').json()
record('P18 cambio de máquina vinculada',f'VM1 {s1.get("ip")}', 'PUT /api/session VM2',
       f'VM2 {s2.get("ip")}', s1.get('vm')=='VM1' and s2.get('vm')=='VM2' and s1.get('ip')!=s2.get('ip'), '')

# ---------- P17 historial ----------
r=api('GET','/api/historial?q=paquetes&limite=5')
hist=r.json()
record('P17 historial y filtros','operaciones registradas','GET /api/historial?q=paquetes&limite=5 -> %d'%r.status_code,
       json.dumps(hist)[:300], r.status_code==200 and len(hist.get('operaciones',[]))>0, '')

# ---------- P06/P07 llave de root en VM2 ----------
old=KEY+'.bak'; oldpub=KEY+'.pub.bak'
shutil.move(KEY,old); shutil.move(KEY+'.pub',oldpub)
con=sqlite3.connect(DB)
con.execute("update credenciales set usuario='root', registrada=0 where vm='VM2'")
con.commit(); con.close()
link('VM2')
antes='sin llave privada en el anfitrión'
r=api('POST','/api/keys/root')
rkey=api('GET','/api/keys/root')
open(KEY+'.nueva','wb').write(rkey.content)
os.chmod(KEY+'.nueva',0o600)
rc,sshout,_=ssh('192.168.1.18','hostname; id -u',key=KEY+'.nueva')
rc2,sshd,_=ssh('192.168.1.18',"sshd -T | grep -E 'passwordauthentication|permitrootlogin'",key=KEY+'.nueva')
record('P06 llave SSH de root',antes,f'POST/GET /api/keys/root -> {r.status_code}/{rkey.status_code}',
       f'{sshout}; {sshd}', r.status_code==201 and rkey.status_code==200 and 'passwordauthentication no' in sshd, r.text[:300])
# P07: intento por contraseña
p7=subprocess.run(['sshpass','-p','1234','ssh','-o','StrictHostKeyChecking=no','-o','PubkeyAuthentication=no',
                   '-o','PreferredAuthentications=password','-o','ConnectTimeout=5','root@192.168.1.18','true'],
                  capture_output=True,text=True,timeout=20)
record('P07 acceso por contraseña','autenticación por contraseña deshabilitada',
       'ssh -o PubkeyAuthentication=no root@VM2', f'rc={p7.returncode} {p7.stderr.strip()[:120]}',
       p7.returncode!=0, '')
# restaurar llave y credenciales
os.remove(KEY+'.nueva'); os.remove(KEY+'.nueva.pub') if os.path.exists(KEY+'.nueva.pub') else None
shutil.move(old,KEY); shutil.move(oldpub,KEY+'.pub')
con=sqlite3.connect(DB); con.execute("update credenciales set usuario='jorge', registrada=1 where vm='VM2'"); con.commit(); con.close()

# ---------- P09 hostname (al final) ----------
link('VM1')
oldname=ssh('192.168.1.16','hostname')[1]
oldhosts=ssh('192.168.1.16','base64 -w0 /etc/hosts')[1]
r=api('PUT','/api/hostname',json={'nombre':'vmcons-lab','ip':'192.168.1.16','actualizar_hosts':True})
newname=ssh('192.168.1.16','hostname')[1]
hosts=ssh('192.168.1.16','grep vmcons-lab /etc/hosts')[1]
record('P09 nombre de host y /etc/hosts',oldname,'PUT /api/hostname -> %d'%r.status_code,
       f'{newname}; {hosts}', r.status_code==200 and newname=='vmcons-lab' and 'vmcons-lab' in hosts, r.text[:250])
ssh('192.168.1.16',f'hostnamectl set-hostname {oldname}; printf %s {oldhosts} | base64 -d > /etc/hosts')

def texto(valor, limite):
    return str(valor).replace('|','\\|').replace('\n',' ')[:limite]

json.dump(results,open('results.json','w',encoding='utf-8'),ensure_ascii=False,indent=2)
with open('results.md','w',encoding='utf-8') as f:
    f.write('| Prueba | Antes | Operación | Después | Resultado | Notas |\n|---|---|---|---|---|---|\n')
    for x in results:
        f.write(f"| {texto(x['test'],80)} | {texto(x['before'],120)} | {texto(x['operation'],160)} | {texto(x['after'],180)} | {'Conforme' if x['ok'] else 'Revisar'} | {texto(x.get('notes',''),120)} |\n")
print('TOTAL',sum(1 for x in results if x['ok']),'/',len(results))
