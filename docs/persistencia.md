# Persistencia de la consola

Las credenciales de las máquinas registradas se guardan en **SQLite**, no en
archivos de texto. La contraseña se cifra con **AES-256-GCM** antes de
insertarse.

## Dependencia: `modernc.org/sqlite`

Se eligió `modernc.org/sqlite` porque es una implementación **pura de Go** del
controlador SQLite:

- no requiere CGO ni una biblioteca del sistema, así que `task build` sigue
  produciendo un binario único y multiplataforma (Windows y GNU/Linux);
- no obliga a instalar SQLite en el equipo anfitrión;
- la API es la de `database/sql`, la misma que usaría cualquier otro
  controlador.

Las alternativas con CGO (`mattn/go-sqlite3`) o SQLCipher rompen la compilación
cruzada y el requisito de binario único; por eso se descartaron.

## Archivos

| Archivo | Contenido | Permisos |
| --- | --- | --- |
| `data/vmcons.db` | Tabla `credenciales` | `0600` |
| `data/credenciales.key` | Clave de 32 bytes para AES-GCM | `0600` |

La ruta de la base se puede cambiar con `VMCONS_DB`; la clave se guarda en el
mismo directorio. Ambos archivos están excluidos de Git (`.gitignore`).

## Esquema

```sql
CREATE TABLE IF NOT EXISTS credenciales (
    vm             TEXT PRIMARY KEY,
    usuario        TEXT NOT NULL,
    password       TEXT NOT NULL,   -- base64(nonce + AES-256-GCM)
    puerto         INTEGER NOT NULL DEFAULT 22,
    direccion      TEXT NOT NULL DEFAULT '',
    registrada     INTEGER NOT NULL DEFAULT 0,
    actualizado_en TEXT NOT NULL
);
```

La tabla `operaciones` del historial comparte la misma base:

```sql
CREATE TABLE operaciones (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    fecha_hora   TEXT NOT NULL,
    vm           TEXT NOT NULL,
    modulo       TEXT NOT NULL,
    operacion    TEXT NOT NULL,
    descripcion  TEXT NOT NULL DEFAULT '',  -- resumen legible
    resultado    TEXT NOT NULL,
    detalle      TEXT NOT NULL DEFAULT ''
);
```

La tabla `configuracion` guarda ajustes simples; por ahora, la última
máquina vinculada y el secreto que firma las cookies de la consola:

```sql
CREATE TABLE configuracion (
    clave TEXT PRIMARY KEY,
    valor TEXT NOT NULL
);
```

La tabla `usuarios_consola` guarda las cuentas que pueden entrar a la
consola. Solo se conserva el hash `bcrypt` de la contraseña:

```sql
CREATE TABLE usuarios_consola (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    usuario   TEXT NOT NULL UNIQUE,
    hash      TEXT NOT NULL,              -- bcrypt, nunca la contraseña
    creado_en TEXT NOT NULL
);
```

La primera vez que se abre la consola la tabla está vacía y la interfaz
ofrece crear la cuenta; a partir de ahí, `POST /api/auth/setup` responde
`409`. El secreto de las cookies se genera con `crypto/rand` en el primer
arranque y se guarda en `configuracion` con la clave `consola_secreto`.

El hash se calcula con `bcrypt` de `golang.org/x/crypto`, que ya formaba
parte del grafo de módulos como dependencia indirecta y pasa a usarse
directamente. Se elige bcrypt —y no SHA-256— porque incorpora sal por
cuenta y un costo configurable, de modo que una copia de la base no
permite probar contraseñas a gran velocidad. La contraseña no se guarda
en ningún otro sitio ni se escribe en los logs.

## Cifrado

1. Al primer arranque se generan 32 bytes aleatorios en
   `credenciales.key` (`0600`).
2. Cada contraseña se cifra con AES-256-GCM y un *nonce* aleatorio; el
   resultado se codifica en base64 y se guarda en `password`.
3. Al leer, se descifra con la misma clave. Si la clave no coincide, la
   contraseña no se puede recuperar y la máquina debe registrarse de nuevo.

La clave vive junto a la base: protege contra lectura casual del archivo y
contra copias de la base sin la clave, pero no contra un usuario con acceso
total al anfitrión. Para el alcance del parcial es suficiente y se documenta
como limitación.

## Migración desde la versión JSON

La versión anterior guardaba `credenciales.json`. Ese archivo ya no se lee:
las máquinas deben registrarse de nuevo desde el modal. El archivo puede
borrarse.

## Copia de seguridad

Para conservar las credenciales hay que copiar **los dos archivos**:
`vmcons.db` y `credenciales.key`.
