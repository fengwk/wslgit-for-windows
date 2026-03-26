package bridge

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

type PathTranslation struct {
	WSLPath string
	Distro  string
}

type PathChecker interface {
	Exists(path string) bool
}

type DriveRemoteResolver interface {
	Resolve(drive string) (string, bool, error)
}

type OSPathChecker struct{}

func (OSPathChecker) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func WindowsPathToWSL(path string) (string, error) {
	translation, err := ResolvePathTranslation(path)
	if err != nil {
		return "", err
	}
	return translation.WSLPath, nil
}

func ResolvePathTranslation(path string) (PathTranslation, error) {
	return resolvePathTranslation(path, defaultDriveRemoteResolver)
}

func resolvePathTranslation(path string, resolver DriveRemoteResolver) (PathTranslation, error) {
	if translation, ok := parseWSLUNCPath(path); ok {
		return translation, nil
	}

	if isWindowsDrivePath(path) {
		if translation, ok, err := resolveMappedDrivePath(path, resolver); err != nil {
			return PathTranslation{}, err
		} else if ok {
			return translation, nil
		}

		drive := unicode.ToLower(rune(path[0]))
		rest := strings.ReplaceAll(path[2:], "\\", "/")
		rest = strings.TrimPrefix(rest, "/")
		if rest == "" {
			return PathTranslation{WSLPath: fmt.Sprintf("/mnt/%c", drive)}, nil
		}
		return PathTranslation{WSLPath: fmt.Sprintf("/mnt/%c/%s", drive, rest)}, nil
	}

	if isUNCPath(path) {
		return PathTranslation{}, fmt.Errorf("UNC 路径暂不支持: %s", path)
	}

	return PathTranslation{}, fmt.Errorf("不是 Windows 本机盘符路径: %s", path)
}

func RewriteArgs(args []string, cwd string, checker PathChecker) []string {
	rewritten := make([]string, 0, len(args))
	waitingForPathValue := false
	pathspecMode := false

	for _, arg := range args {
		switch {
		case pathspecMode:
			rewritten = append(rewritten, rewriteKnownPath(arg))
		case waitingForPathValue:
			rewritten = append(rewritten, rewriteExplicitPathValue(arg))
			waitingForPathValue = false
		case arg == "--":
			rewritten = append(rewritten, arg)
			pathspecMode = true
		case isExplicitPathOption(arg):
			rewritten = append(rewritten, arg)
			waitingForPathValue = true
		case strings.HasPrefix(arg, "--git-dir="):
			rewritten = append(rewritten, "--git-dir="+rewriteExplicitPathValue(strings.TrimPrefix(arg, "--git-dir=")))
		case strings.HasPrefix(arg, "--work-tree="):
			rewritten = append(rewritten, "--work-tree="+rewriteExplicitPathValue(strings.TrimPrefix(arg, "--work-tree=")))
		default:
			rewritten = append(rewritten, rewriteGeneralArg(arg, cwd, checker))
		}
	}

	return rewritten
}

func rewriteExplicitPathValue(arg string) string {
	if wslPath, err := WindowsPathToWSL(arg); err == nil {
		return wslPath
	}
	return normalizeRelativePath(arg)
}

func rewriteKnownPath(arg string) string {
	if wslPath, err := WindowsPathToWSL(arg); err == nil {
		return wslPath
	}
	return normalizeRelativePath(arg)
}

func rewriteGeneralArg(arg string, cwd string, checker PathChecker) string {
	if wslPath, err := WindowsPathToWSL(arg); err == nil {
		return wslPath
	}

	if shouldRewriteRelativeArg(arg, cwd, checker) {
		return normalizeRelativePath(arg)
	}

	return arg
}

func shouldRewriteRelativeArg(arg string, cwd string, checker PathChecker) bool {
	if checker == nil {
		return false
	}
	if strings.HasPrefix(arg, "-") || looksLikeURL(arg) || !looksLikeRelativeWindowsPath(arg) {
		return false
	}
	if looksLikeWildcardPath(arg) || strings.HasPrefix(arg, ".\\") || strings.HasPrefix(arg, "..\\") {
		return true
	}

	candidate := joinWindowsPath(cwd, arg)
	return checker.Exists(candidate)
}

func looksLikeRelativeWindowsPath(arg string) bool {
	if arg == "" {
		return false
	}
	return strings.Contains(arg, "\\") || strings.HasPrefix(arg, ".\\") || strings.HasPrefix(arg, "..\\")
}

func normalizeRelativePath(arg string) string {
	return strings.ReplaceAll(arg, "\\", "/")
}

func isExplicitPathOption(arg string) bool {
	switch arg {
	case "-C", "--git-dir", "--work-tree":
		return true
	default:
		return false
	}
}

func isWindowsDrivePath(path string) bool {
	if len(path) < 3 {
		return false
	}
	return ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' && (path[2] == '\\' || path[2] == '/')
}

func isUNCPath(path string) bool {
	return strings.HasPrefix(path, "\\\\") || strings.HasPrefix(path, "//")
}

func looksLikeURL(path string) bool {
	return strings.Contains(path, "://")
}

func looksLikeWildcardPath(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func joinWindowsPath(base string, child string) string {
	base = strings.TrimRight(strings.ReplaceAll(base, "/", "\\"), "\\")
	child = strings.TrimLeft(strings.ReplaceAll(child, "/", "\\"), "\\")
	if base == "" {
		return child
	}
	if child == "" {
		return base
	}
	return base + "\\" + child
}

func resolveMappedDrivePath(path string, resolver DriveRemoteResolver) (PathTranslation, bool, error) {
	if resolver == nil {
		return PathTranslation{}, false, nil
	}

	drive := strings.ToUpper(path[:2])
	remoteRoot, ok, err := resolver.Resolve(drive)
	if err != nil {
		return PathTranslation{}, false, err
	}
	if !ok {
		return PathTranslation{}, false, nil
	}

	relativePath := strings.TrimLeft(path[2:], `\\/`)
	remotePath := remoteRoot
	if relativePath != "" {
		remotePath = joinWindowsPath(remoteRoot, relativePath)
	}

	translation, matched := parseWSLUNCPath(remotePath)
	if matched {
		return translation, true, nil
	}

	return PathTranslation{}, false, fmt.Errorf("不支持的映射盘 %s，当前仅支持映射到 \\\\wsl$ 或 \\\\wsl.localhost 的盘符", drive)
}

func parseWSLUNCPath(path string) (PathTranslation, bool) {
	if !isUNCPath(path) {
		return PathTranslation{}, false
	}

	normalized := strings.ReplaceAll(path, "/", "\\")
	normalized = strings.TrimLeft(normalized, "\\")
	parts := strings.Split(normalized, "\\")
	if len(parts) < 2 {
		return PathTranslation{}, false
	}

	host := strings.ToLower(parts[0])
	if host != "wsl$" && host != "wsl.localhost" {
		return PathTranslation{}, false
	}

	distro := parts[1]
	rest := "/"
	if len(parts) > 2 {
		rest = "/" + strings.Join(parts[2:], "/")
	}

	return PathTranslation{WSLPath: rest, Distro: distro}, true
}

type noopDriveRemoteResolver struct{}

func (noopDriveRemoteResolver) Resolve(string) (string, bool, error) {
	return "", false, nil
}
