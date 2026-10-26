// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package domain

import (
	"strings"
	"testing"
)

const (
	saludo       = "hola"
	binEcho      = "/bin/echo"
	msgShellLine = "ShellLine() = %q, want %q"
)

func TestQuoteWrapsInSingleQuotes(t *testing.T) {
	t.Parallel()

	if got := Quote(saludo); got != "'hola'" {
		t.Errorf("Quote(hola) = %q, want %q", got, "'hola'")
	}
}

func TestQuoteEscapesSingleQuotes(t *testing.T) {
	t.Parallel()

	if got := Quote("it's"); got != `'it'\''s'` {
		t.Errorf("Quote(it's) = %q, want %q", got, `'it'\''s'`)
	}
}

func TestQuoteEmptyToken(t *testing.T) {
	t.Parallel()

	if got := Quote(""); got != "''" {
		t.Errorf("Quote(\"\") = %q, want %q", got, "''")
	}
}

func TestShellLineQuotesProgramAndArguments(t *testing.T) {
	t.Parallel()

	cmd := &Command{Path: binEcho, Args: []string{saludo, "mundo cruel"}}
	want := `'/bin/echo' 'hola' 'mundo cruel'`

	if got := cmd.ShellLine(); got != want {
		t.Errorf(msgShellLine, got, want)
	}
}

func TestShellLineWithDir(t *testing.T) {
	t.Parallel()

	cmd := &Command{Path: binEcho, Args: []string{saludo}, Dir: "/opt/datos"}
	want := `cd '/opt/datos' && '/bin/echo' 'hola'`

	if got := cmd.ShellLine(); got != want {
		t.Errorf(msgShellLine, got, want)
	}
}

func TestShellLineWithEnv(t *testing.T) {
	t.Parallel()

	cmd := &Command{Path: binEcho, Args: []string{saludo}, Env: []string{"FRASE=buenos días"}}
	want := `env 'FRASE=buenos días' '/bin/echo' 'hola'`

	if got := cmd.ShellLine(); got != want {
		t.Errorf(msgShellLine, got, want)
	}
}

func TestShellLineWithDirAndEnv(t *testing.T) {
	t.Parallel()

	cmd := &Command{Path: binEcho, Dir: "/tmp", Env: []string{"A=b"}}
	want := `cd '/tmp' && env 'A=b' '/bin/echo'`

	if got := cmd.ShellLine(); got != want {
		t.Errorf(msgShellLine, got, want)
	}
}

func TestExitErrorMessageContainsCode(t *testing.T) {
	t.Parallel()

	err := &ExitError{Code: 3, Command: "apt-get update"}

	if !strings.Contains(err.Error(), "3") {
		t.Errorf("ExitError.Error() = %q, quiere que mencione el código 3", err.Error())
	}
}

func TestMachineStateValues(t *testing.T) {
	t.Parallel()

	testCases := map[MachineState]string{
		StateRunning:    "running",
		StatePoweredOff: "poweroff",
		StatePaused:     "paused",
		StateSaved:      "saved",
		StateUnknown:    "unknown",
	}

	for state, want := range testCases {
		if string(state) != want {
			t.Errorf("MachineState = %q, want %q", string(state), want)
		}
	}
}
