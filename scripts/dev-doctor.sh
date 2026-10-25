#!/usr/bin/env bash
# Comprueba las dependencias de desarrollo de vmcons.
# Uso: task doctor

set -u

rojo=""; verde=""; amarillo=""; fin=""
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
  rojo="\033[31m"; verde="\033[32m"; amarillo="\033[33m"; fin="\033[0m"
fi

fallos=0
faltantes=()

version() {
  case "$1" in
    go) go version ;;
    ssh) "$1" -V 2>&1 | head -n 1 ;;
    scp|sshpass) echo "instalado" ;;
    *) "$1" --version 2>&1 | head -n 1 ;;
  esac
}

comprobar_go_tool() {
  nombre="$1"
  herramienta="$2"

  if texto=$(go tool "$herramienta" --version 2>&1 | head -n 1); then
    printf "  ${verde}OK${fin}    %-15s %s\n" "$nombre" "$texto"
    return
  fi

  printf "  ${rojo}FALTA${fin} %-15s\n" "$nombre"
  fallos=$((fallos + 1))
}

comprobar() {
  nombre="$1"
  comando="$2"
  obligatorio="$3"

  if command -v "$comando" >/dev/null 2>&1; then
    texto=$(version "$comando")
    printf "  ${verde}OK${fin}    %-15s %s\n" "$nombre" "$texto"
    return
  fi

  if [ "$obligatorio" = "si" ]; then
    printf "  ${rojo}FALTA${fin} %-15s\n" "$nombre"
    fallos=$((fallos + 1))
    faltantes+=("$comando")
  else
    printf "  ${amarillo}OPC.${fin}  %-15s no instalado\n" "$nombre"
  fi
}

echo
echo "Dependencias de desarrollo de vmcons"
echo "------------------------------------"
echo "Requeridas:"
comprobar "Go"             go             si
comprobar "Task"           task           si
comprobar_go_tool "gofumpt"       gofumpt
comprobar_go_tool "golangci-lint" golangci-lint
comprobar_go_tool "gomarkdoc"     gomarkdoc
comprobar "Node.js"        node           si
comprobar "pnpm"           pnpm           si
comprobar "VirtualBox"     VBoxManage     si
comprobar "OpenSSH"        ssh            si
comprobar "scp"            scp            si
comprobar "sshpass"        sshpass        si

echo
echo "Opcionales:"
comprobar "PlantUML"       plantuml       no
comprobar "latexmk"        latexmk        no
comprobar "npx (Redocly)"  npx            no

if [ "$fallos" -ne 0 ]; then
  echo
  echo -e "${rojo}Faltan $fallos dependencias requeridas.${fin}"
  echo "Sugerencias:"
  echo "  - Herramientas de Go:      task setup-go (go mod download)"
  echo "  - Frontend:                task setup-web"
  echo "  - pnpm:                    corepack enable && corepack prepare pnpm@latest --activate"
  echo "  - Task:                    go install github.com/go-task/task/v3/cmd/task@latest"
  echo "  - Debian/Ubuntu:           sudo apt install virtualbox openssh-client sshpass"
  echo "  - Fedora:                  sudo dnf install VirtualBox openssh-clients sshpass"
  echo "  - Arch:                    sudo pacman -S virtualbox openssh sshpass"
  echo "  - Las herramientas de Go se ejecutan con «go tool»; no hace falta el PATH."
  exit 1
fi

echo
echo -e "${verde}Todas las dependencias requeridas están instaladas.${fin}"
