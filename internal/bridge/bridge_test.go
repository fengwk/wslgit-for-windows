package bridge

import (
	"strings"
	"testing"
)

type fakeChecker map[string]bool

type fakeDriveResolver map[string]string

func (f fakeChecker) Exists(path string) bool {
	return f[path]
}

func (f fakeDriveResolver) Resolve(drive string) (string, bool, error) {
	remote, ok := f[drive]
	return remote, ok, nil
}

func TestWindowsPathToWSL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "drive root", input: `C:\`, want: "/mnt/c"},
		{name: "nested path", input: `D:\work\repo`, want: "/mnt/d/work/repo"},
		{name: "slash path", input: `E:/src/app`, want: "/mnt/e/src/app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WindowsPathToWSL(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestResolvePathTranslationMappedWSLDrive(t *testing.T) {
	translation, err := resolvePathTranslation(`X:\home\fengwk\proj\bro-cli`, fakeDriveResolver{
		`X:`: `\\wsl.localhost\Ubuntu-24.04`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if translation.WSLPath != "/home/fengwk/proj/bro-cli" {
		t.Fatalf("got path %q", translation.WSLPath)
	}
	if translation.Distro != "Ubuntu-24.04" {
		t.Fatalf("got distro %q", translation.Distro)
	}
}

func TestResolvePathTranslationUNCWSLPath(t *testing.T) {
	translation, err := resolvePathTranslation(`\\wsl$\Ubuntu-24.04\home\fengwk\repo`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if translation.WSLPath != "/home/fengwk/repo" {
		t.Fatalf("got path %q", translation.WSLPath)
	}
	if translation.Distro != "Ubuntu-24.04" {
		t.Fatalf("got distro %q", translation.Distro)
	}
}

func TestResolvePathTranslationRejectsNonWSLMappedDrive(t *testing.T) {
	_, err := resolvePathTranslation(`Z:\repo`, fakeDriveResolver{
		`Z:`: `\\server\share`,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `仅支持映射到 \\wsl$ 或 \\wsl.localhost`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRewriteArgs(t *testing.T) {
	checker := fakeChecker{
		`C:\repo\src\main.go`: true,
	}
	args := []string{"add", `src\main.go`, "--git-dir", `C:\repo\.git`, "--", `dir\nested.txt`}
	got := RewriteArgs(args, `C:\repo`, checker)
	want := []string{"add", "src/main.go", "--git-dir", "/mnt/c/repo/.git", "--", "dir/nested.txt"}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d got %q want %q", i, got[i], want[i])
		}
	}
}

func TestRewriteArgsKeepsNonPathValues(t *testing.T) {
	args := []string{"commit", "-m", `fix\message`}
	got := RewriteArgs(args, `C:\repo`, fakeChecker{})
	if got[2] != `fix\message` {
		t.Fatalf("got %q want %q", got[2], `fix\message`)
	}
}

func TestRewriteArgsSupportsWildcardPathspec(t *testing.T) {
	args := []string{"add", `src\*.go`}
	got := RewriteArgs(args, `C:\repo`, fakeChecker{})
	if got[1] != `src/*.go` {
		t.Fatalf("got %q want %q", got[1], `src/*.go`)
	}
}

func TestBuildInvocationFallback(t *testing.T) {
	config := Config{WSLPath: "wsl.exe", GitBinary: "git", Distro: "Ubuntu"}
	invocation := BuildInvocation(config, "/mnt/c/repo", []string{"status"}, false)
	if invocation.Program != "wsl.exe" {
		t.Fatalf("got program %q", invocation.Program)
	}
	if len(invocation.Args) < 7 {
		t.Fatalf("unexpected args: %v", invocation.Args)
	}
	if invocation.Args[0] != "-d" || invocation.Args[1] != "Ubuntu" {
		t.Fatalf("unexpected distro args: %v", invocation.Args)
	}
	if invocation.Args[2] != "sh" || invocation.Args[3] != "-lc" {
		t.Fatalf("unexpected fallback args: %v", invocation.Args)
	}
}
