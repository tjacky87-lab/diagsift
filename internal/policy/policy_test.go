package policy

import "testing"

func TestShellInterpreterDenials(t *testing.T) {
	tests := []struct {
		exe  string
		args []string
	}{
		{"cmd", nil},
		{"CMD.EXE", []string{"/q"}},
		{`C:\Windows\System32\CmD.eXe`, []string{"--help"}},
		{"powershell.com", nil},
		{`C:/Tools/PwSh.ExE`, []string{"-File", "script.ps1"}},
		{"/bin/sh", nil},
		{"/usr/local/bin/BASH", []string{"--version"}},
		{"zsh", []string{"-f"}},
		{"/bin/DASH", nil},
		{"ksh.bat", []string{"script"}},
	}
	for _, test := range tests {
		if err := ValidateCommand(test.exe, test.args); err == nil {
			t.Errorf("ValidateCommand(%q, %q) unexpectedly succeeded", test.exe, test.args)
		}
	}
}

func TestWindowsBatchFileDenials(t *testing.T) {
	for _, executable := range []string{
		`C:\tools\collect.bat`,
		"diag.CMD",
		`C:\Program Files\DiagSift\COLLECT.BAT`,
		"C:/tools/nested/diag.cmd",
	} {
		if err := ValidateCommand(executable, []string{"ignored"}); err == nil {
			t.Errorf("ValidateCommand(%q) unexpectedly allowed a Windows batch file", executable)
		}
	}
}

func TestDirectCommandAllowed(t *testing.T) {
	for _, executable := range []string{"example-app", `C:\Program Files\Example\example.exe`, "/usr/bin/uname", "mybash"} {
		if err := ValidateCommand(executable, []string{"version"}); err != nil {
			t.Errorf("ValidateCommand(%q) failed: %v", executable, err)
		}
	}
}
