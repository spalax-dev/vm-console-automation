# Resultados de las pruebas de campo (P01–P18)

Ejecución del **26 de octubre de 2026** sobre el escenario del parcial:

| Máquina | IP | Papel |
| --- | --- | --- |
| VM1 | `192.168.1.16` | Máquina de pruebas generales y sistema de archivos |
| VM2 | `192.168.1.18` | Segunda máquina; registro, llave de `root` y servidor HTTP del back-end |
| balanceador | `192.168.1.21` | HAProxy con un `backend` temporal `vmcons_app` |

La consola se ejecutó con una copia de `data/` en `/tmp/opencode/e2e/data`, de
modo que la base real no se modificó. Cada prueba siguió el mismo ciclo:
comprobar el estado anterior por SSH, ejecutar la operación por la API REST y
comprobar el estado posterior por SSH. Las comprobaciones se hicieron con la
llave de `root` instalada por la propia consola y, cuando la prueba lo exigía,
con `sshpass` para verificar que el acceso por contraseña estaba bloqueado.

El resultado fue **18 de 18 pruebas conformes**. Las pruebas restauran lo que
modifican: la llave privada del anfitrión, el nombre de host, `/etc/issue`, la
unidad temporal de systemd, el usuario temporal y los servidores del
balanceador quedaron como estaban antes de la ejecución. La clave pública que
P06 agregó a `authorized_keys` en VM2 queda inerte al retirar la llave privada
nueva; el acceso con la llave original se comprobó al final.

| Prueba | Estado anterior | Operación | Estado posterior | Resultado |
| --- | --- | --- | --- | --- |
| P01 | VM2 sin registrar (registro eliminado) | `POST /api/session/register` con `jorge` | `registrada=true`; Guest Additions 6.0.0 r127566 | Conforme |
| P02 | VM2 registrada | `GET /api/session/diagnostico` | Red puente, IP `192.168.1.18`, SSH abierto y aviso de Guest Additions desactualizado | Conforme |
| P03 | `tree` no instalado | `POST /api/packages` con `tree` | `install ok installed` | Conforme |
| P04 | `tree` instalado | `DELETE /api/packages/tree` | `dpkg-query`: no se encontró el paquete | Conforme |
| P05 | `tree` eliminado | `POST /api/packages` con `paquete-inexistente` | `500`, código 1 y `stderr` de APT con `E:` | Conforme |
| P06 | Sin llave privada en el anfitrión | `POST /api/keys/root` y `GET /api/keys/root` | SSH como `root` con la llave nueva; `passwordauthentication no` | Conforme |
| P07 | Contraseña deshabilitada por P06 | `ssh -o PubkeyAuthentication=no root@192.168.1.18` | `Permission denied (publickey)` | Conforme |
| P08 | `/etc/issue` original de la distribución | `PUT /api/issue` con mensaje, nombre, fecha, hora e IP | Archivo con `\n`, `\d`, `\t` y `\4`; restaurado al terminar | Conforme |
| P09 | Nombre de host `vm1` | `PUT /api/hostname` con `vmcons-lab` | `hostname` devuelve `vmcons-lab`; `/etc/hosts` incluye `192.168.1.16 vmcons-lab`; restaurado | Conforme |
| P10 | Usuario `alumno` inexistente | `POST /api/users` con `sudo` y llaves | UID 1001, grupo `sudo`, contraseña bloqueada, `authorized_keys` en 0600 y descarga de la llave privada | Conforme |
| P11 | `alumno` con directorio personal | `DELETE /api/users/alumno?files=true` | Usuario y `/home/alumno` eliminados | Conforme |
| P12 | Unidad `vmcons-demo.service` inexistente | `PUT /api/units/vmcons-demo.service` con script y unidad | Unidad creada y `systemctl cat` la muestra; `daemon-reload` | Conforme |
| P13 | Unidad inactiva y deshabilitada | `PATCH activo`, `PATCH habilitado`, `POST restarts` y `PATCH detener` | `active`, `enabled`, `202` e `inactive` | Conforme |
| P14 | Script sin ejecuciones | Iniciar y reiniciar la unidad | Cinco líneas con fecha y hora ISO-8601 | Conforme |
| P15 | Back-end `vmcons_app` sin servidores | `POST /api/haproxy/backend/servers` con `192.168.1.18:8080` | Línea `srv_192_168_1_18_8080`, `haproxy -c` válida y `curl` al balanceador responde `200` | Conforme |
| P16 | Servidor `192.168.1.18:8080` presente | `DELETE /api/haproxy/backend/servers/192.168.1.18:8080` | Línea eliminada, `haproxy -c` válida, `curl` responde `503` y repetir la baja da `404` | Conforme |
| P17 | Operaciones anteriores en la base | `GET /api/historial?q=paquetes&limite=5` | Operaciones con descripción legible y filtro aplicado | Conforme |
| P18 | VM1 vinculada (`192.168.1.16`) | `PUT /api/session` con VM2 | La sesión informa VM2 y `192.168.1.18` | Conforme |

## Observaciones

- **P01 y P02** se ejecutaron sobre VM2 porque era la máquina que aún no tenía
  registro en la copia de la base. El diagnóstico conservó el aviso de Guest
  Additions 6.0.0 frente a VirtualBox 7.2.18; la consola no oculta ese problema.
- **P06 y P07** se ejecutaron sobre VM2. Antes de generar la llave se movió la
  llave del anfitrión y se ajustó la credencial de VM2 en la copia de la base;
  al terminar se restauraron la llave y la credencial. El archivo descargado se
  guardó con permisos 0600 para que SSH lo aceptara.
- **P12 a P14** usaron una unidad temporal `vmcons-demo.service` con un script
  que escribe la fecha cada dos segundos. No se usó la aplicación del catálogo
  porque el enunciado no entrega su binario; el mecanismo de despliegue,
  permisos, `daemon-reload`, arranque, habilitación, reinicio y parada es el
  mismo que usaría esa aplicación.
- **P15 y P16** usaron un `frontend vmcons_front` y un `backend vmcons_app`
  temporales añadidos al `haproxy.cfg` del balanceador, para no alterar su
  configuración original. Al terminar se eliminó el servidor agregado y la
  configuración quedó válida.
- La comprobación de P15 con `curl` al balanceador devolvió `200` porque VM2
  atendía el puerto 8080 con un servidor HTTP temporal; después de P16 el mismo
  `curl` devolvió `503` al quedar el back-end sin servidores.

## Reproducción

El arnés quedó en `scripts/pruebas-campo.py` y los resultados crudos en
`docs/resultados-pruebas.json`. Las credenciales y la base usadas fueron las
de la copia temporal, no las del repositorio. La ejecución completa tardó unos
cuatro minutos y se protegió con `systemd-inhibit` para evitar que el anfitrión
entrara en suspensión durante las pruebas.
