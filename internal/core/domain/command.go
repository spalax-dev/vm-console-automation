// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package domain

import "strings"

// Command describe una orden que debe ejecutarse dentro de la máquina
// administrada. Es independiente del mecanismo que la ejecute.
type Command struct {
	// Path es la ruta absoluta del programa a ejecutar.
	Path string
	// Args son los argumentos del programa, sin interpretación del shell.
	Args []string
	// Dir es el directorio de trabajo desde el que se ejecuta la orden.
	Dir string
	// Env son variables de entorno adicionales en formato CLAVE=valor.
	Env []string
}

// Quote encierra token entre comillas simples para que un shell POSIX
// lo trate como un solo argumento literal, escapando las comillas simples
// que contenga.
func Quote(token string) string {
	if token == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(token, "'", `'\''`) + "'"
}

// ShellLine convierte la orden en una línea de shell POSIX en la que
// cada componente está citado. Es la representación que la consola
// entrega a los adaptadores que ejecutan órdenes a través de un shell.
func (c *Command) ShellLine() string {
	words := make([]string, 0, len(c.Args)+1)
	words = append(words, Quote(c.Path))

	for _, arg := range c.Args {
		words = append(words, Quote(arg))
	}

	line := strings.Join(words, " ")

	if len(c.Env) > 0 {
		envs := make([]string, 0, len(c.Env))
		for _, kv := range c.Env {
			envs = append(envs, Quote(kv))
		}

		line = "env " + strings.Join(envs, " ") + " " + line
	}

	if c.Dir != "" {
		line = "cd " + Quote(c.Dir) + " && " + line
	}

	return line
}
