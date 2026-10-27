# Resultados de las pruebas de campo (P01–P22)

Ejecución del **26 de octubre de 2026** sobre el escenario del parcial:

| Máquina | IP | Papel |
| --- | --- | --- |
| S1 | `192.168.1.29` | Servidor de imágenes en el puerto 8000; pruebas de sistema |
| S2 | `192.168.1.30` | Segundo servidor de imágenes; registro, llave de `root` y cambio de máquina |
| B | `192.168.1.32` | Balanceador HAProxy (`vmcons_front` → `vmcons_app`) |

La consola se ejecutó con una copia de `data/` en `/tmp/opencode/e2e/data`, de
modo que la base real no se modificó. Cada prueba siguió el mismo ciclo:
comprobar el estado anterior por SSH, ejecutar la operación por la API REST y
comprobar el estado posterior por SSH. Las comprobaciones se hicieron con la
llave de `root` instalada por la propia consola y, cuando la prueba lo exigía,
con `sshpass` para verificar que el acceso por contraseña estaba bloqueado.

Las pruebas **P01–P18** se ejecutaron con el arnés
`scripts/pruebas-campo.py`, que dejó los resultados crudos en
`docs/resultados-pruebas.json`. Las pruebas **P19–P22** de HAProxy se
verificaron en la misma sesión desde la interfaz y el socket de estadísticas,
con `oha 1.16.0` como generador de carga. El resultado fue **22 de 22 pruebas
conformes**.

Las pruebas restauran lo que modifican: la llave privada del anfitrión, el
nombre de host, `/etc/issue`, la unidad temporal de systemd, el usuario
temporal, la configuración de HAProxy y los servidores del balanceador
quedaron como estaban antes de la ejecución. La clave pública que P06 agregó a
`authorized_keys` en S2 queda inerte al retirar la llave privada nueva; el
acceso con la llave original se comprobó al final.

| Prueba | Estado anterior | Operación | Estado posterior | Resultado |
| --- | --- | --- | --- | --- |
| P01 | S2 sin registrar (registro eliminado) | `POST /api/session/register` con `jorge` | `registrada=true`; Guest Additions 6.0.0 r127566 | Conforme |
| P02 | S2 registrada | `GET /api/session/diagnostico` | Red puente, IP `192.168.1.30`, SSH abierto y aviso de Guest Additions desactualizado | Conforme |
| P03 | `tree` no instalado | `POST /api/packages` con `tree` | `install ok installed` | Conforme |
| P04 | `tree` instalado | `DELETE /api/packages/tree` | `dpkg-query`: no se encontró el paquete | Conforme |
| P05 | `tree` eliminado | `POST /api/packages` con `paquete-inexistente` | `500`, código 1 y `stderr` de APT con `E:` | Conforme |
| P06 | Sin llave privada en el anfitrión | `POST /api/keys/root` y `GET /api/keys/root` | SSH como `root` con la llave nueva; `passwordauthentication no` | Conforme |
| P07 | Contraseña deshabilitada por P06 | `ssh -o PubkeyAuthentication=no root@192.168.1.30` | `Permission denied (publickey)` | Conforme |
| P08 | `/etc/issue` original de la distribución | `PUT /api/issue` con mensaje, nombre, fecha, hora e IP | Archivo con `\n`, `\d`, `\t` y `\4`; restaurado al terminar | Conforme |
| P09 | Nombre de host `base` | `PUT /api/hostname` con `vmcons-lab` | `hostname` devuelve `vmcons-lab`; `/etc/hosts` incluye `192.168.1.29 vmcons-lab`; restaurado | Conforme |
| P10 | Usuario `alumno` inexistente | `POST /api/users` con `sudo` y llaves | UID 1001, grupo `sudo`, contraseña bloqueada, `authorized_keys` en 0600 y descarga de la llave privada | Conforme |
| P11 | `alumno` con directorio personal | `DELETE /api/users/alumno?files=true` | Usuario y `/home/alumno` eliminados | Conforme |
| P12 | Unidad `vmcons-demo.service` inexistente | `PUT /api/units/vmcons-demo.service` con script y unidad | Unidad creada y `systemctl cat` la muestra; `daemon-reload` | Conforme |
| P13 | Unidad inactiva y deshabilitada | `PATCH activo`, `PATCH habilitado`, `POST restarts` y `PATCH detener` | `active`, `enabled`, `202` e `inactive` | Conforme |
| P14 | Script sin ejecuciones | Iniciar y reiniciar la unidad | Cinco líneas con fecha y hora ISO-8601 | Conforme |
| P15 | Back-end `vmcons_app` sin servidores | `POST /api/haproxy/backend/servers` con `192.168.1.29:8000` y `192.168.1.30:8000` | Líneas `srv_192_168_1_29_8000` y `srv_192_168_1_30_8000`, `haproxy -c` válida y `curl` al balanceador responde `200` | Conforme |
| P16 | S1 y S2 en el back-end | `DELETE /api/haproxy/backend/servers` de S1 y después de S2 | Con S1 fuera el tráfico solo llega a S2 (`200`); sin servidores la entrada responde `503` y repetir la baja da `404` | Conforme |
| P17 | Operaciones anteriores en la base | `GET /api/historial?q=paquetes&limite=5` | Operaciones con descripción legible y filtro aplicado | Conforme |
| P18 | S1 vinculada (`192.168.1.29`) | `PUT /api/session` con S2 | La sesión informa S2 y `192.168.1.30` | Conforme |
| P19 | Configuración sin bloques `backend` | `GET /api/haproxy/status` | Problema «no tiene bloques backend»; binario, configuración válida, servicio activo y puerto informados | Conforme |
| P20 | Sin `frontend` ni `backend` propios | `POST /api/haproxy/configurar` | `vmcons_front` en el puerto 80, `vmcons_app`, copia `.bak-<fecha>` y servicio recargado | Conforme |
| P21 | S1 y S2 agregados al back-end | `GET /api/haproxy/backend/servers` con el detalle abierto | `UP`, peso 1, `L4OK`, sesiones, cola, bytes y más de 13 000 peticiones acumuladas por servidor | Conforme |
| P22 | Totales similares en S1 y S2 (13 822 cada uno) | `oha --no-tui -n 600 -c 30 http://192.168.1.32/` | `Success rate: 100.00%`, 600 respuestas `2xx` y 300 registros más en cada servidor (14 222) | Conforme |

## Observaciones

- **P01 y P02** se ejecutaron sobre S2 porque era la máquina que aún no tenía
  registro en la copia de la base. El diagnóstico conservó el aviso de Guest
  Additions 6.0.0 frente a VirtualBox 7.2.18; la consola no oculta ese problema.
- **P06 y P07** se ejecutaron sobre S2. Antes de generar la llave se movió la
  llave del anfitrión y se ajustó la credencial de S2 en la copia de la base;
  al terminar se restauraron la llave y la credencial. El archivo descargado se
  guardó con permisos 0600 para que SSH lo aceptara.
- **P12 a P14** usaron una unidad temporal `vmcons-demo.service` con un script
  que escribe la fecha cada dos segundos. No se usó la aplicación del catálogo
  porque el enunciado no entrega su binario; el mecanismo de despliegue,
  permisos, `daemon-reload`, arranque, habilitación, reinicio y parada es el
  mismo que usaría esa aplicación.
- **P15 y P16** trabajaron sobre el `frontend vmcons_front` y el `backend
  vmcons_app` que crea la configuración rápida de P20. Los servidores S1 y S2
  atendían el puerto 8000 con `srvimglinux.service`; al quitar S1, HAProxy
  siguió atendiendo por S2 y, al quitar S2, respondió `503` porque el back-end
  quedó sin servidores.
- **P19 y P20** partieron de la configuración por omisión de HAProxy, que no
  tiene bloques `backend`. La consola lo detectó sin modificar nada y la
  configuración rápida escribió `vmcons_front` en el puerto 80 (libre),
  respaldó el archivo anterior como `haproxy.cfg.bak-<fecha>` y recargó el
  servicio. Al final se restauró la configuración de la grabación.
- **P21** leyó las métricas del socket `/run/haproxy/admin.sock` con `socat`.
  Ambos servidores aparecieron `UP`, con peso 1, chequeo `L4OK` —que solo
  valida TCP— y más de 13 000 peticiones acumuladas; el detalle de la interfaz
  muestra además duración del chequeo, cola, bytes enviados y recibidos.
- **P22** copió la URL de entrada (`http://192.168.1.32/`) desde la tarjeta de
  la consola y generó 600 peticiones con `oha` (30 conexiones concurrentes).
  Todas las respuestas fueron `2xx`; los dos servidores pasaron de 13 822 a
  14 222 peticiones, es decir, 300 cada uno, y el máximo de sesiones
  simultáneas subió en ambos, lo que evidencia el reparto round-robin con 30
  conexiones concurrentes.

## Reproducción

El arnés de P01–P18 quedó en `scripts/pruebas-campo.py` y los resultados crudos
en `docs/resultados-pruebas.json`. Ese archivo conserva el nombre y la IP que
tenían las máquinas durante la ejecución (VM1/VM2/balanceador,
`.16`/`.18`/`.21`); después se renombraron a S1, S2 y B, y el DHCP les asignó
`.29`, `.30` y `.32`, que es el escenario con el que se graba el video. Las
credenciales y la base usadas fueron las de la copia temporal, no las del
repositorio. Las pruebas de HAProxy (P19–P22) se reproducen desde la interfaz
con las órdenes descritas en `docs/guion-video/guion.pdf`; `oha` se instala
aparte en el anfitrión.
