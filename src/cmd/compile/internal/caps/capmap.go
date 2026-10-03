// Copyright 2026 The Gallivant Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package caps implements Gallivant's import capability analysis.
//
// Each package is charged a set of capabilities for the standard-library
// symbols it references directly (see Required and RequiredOnImport) and
// inherits the effective capabilities of every non-standard-library
// package it imports. The effective set is recorded in export data (see
// Fact) and, at every import site that crosses a module boundary, is
// compared against the capabilities granted by a trailing //caps: comment.
//
// See doc/gallivant/caps.md for the design.
package caps

import (
	"sort"
	"strings"
)

// Cap is a single capability token as it appears in a //caps: directive.
type Cap string

// String implements fmt.Stringer.
func (c Cap) String() string { return string(c) }

const (
	CapFileRead  Cap = "file:read"
	CapFileWrite Cap = "file:write"
	CapNet       Cap = "net"
	CapExec      Cap = "exec"
	CapEnv       Cap = "env"
	CapUnsafe    Cap = "unsafe"
)

// FormatList renders a cap slice as "[a, b, c]" with stable presentation
// suitable for diagnostics and fact output.
func FormatList(caps []Cap) string {
	parts := make([]string, len(caps))
	for i, c := range caps {
		parts[i] = c.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// required is the lookup table mapping stdlib pkgPath -> symbol name ->
// required caps. Symbol names are either "FuncName" or "TypeName.MethodName".
var required = map[string]map[string][]Cap{
	"os": {
		// File operations — open / create
		"Open":       {CapFileRead},
		"OpenFile":   {CapFileRead, CapFileWrite},
		"ReadFile":   {CapFileRead},
		"Create":     {CapFileWrite},
		"CreateTemp": {CapFileWrite},
		"WriteFile":  {CapFileWrite},
		// File operations — mutate / delete
		"Remove":    {CapFileWrite},
		"RemoveAll": {CapFileWrite},
		"Rename":    {CapFileWrite},
		"Symlink":   {CapFileWrite},
		"Link":      {CapFileWrite},
		"Chtimes":   {CapFileWrite},
		// Directory operations
		"Mkdir":     {CapFileWrite},
		"MkdirAll":  {CapFileWrite},
		"MkdirTemp": {CapFileWrite},
		"Truncate":  {CapFileWrite},
		"Chmod":     {CapFileWrite},
		"Chown":     {CapFileWrite},
		"Lchown":    {CapFileWrite},
		// Read-only filesystem queries
		"Stat":     {CapFileRead},
		"Lstat":    {CapFileRead},
		"Readlink": {CapFileRead},
		"ReadDir":  {CapFileRead},
		"Getwd":    {CapFileRead},
		"SameFile": {CapFileRead},
		// Pipe
		"Pipe": {CapFileRead, CapFileWrite},
		// Environment
		"Getenv":        {CapEnv},
		"LookupEnv":     {CapEnv},
		"Environ":       {CapEnv},
		"Setenv":        {CapEnv},
		"Unsetenv":      {CapEnv},
		"Clearenv":      {CapEnv},
		"Expand":        {CapEnv},
		"ExpandEnv":     {CapEnv},
		"UserHomeDir":   {CapEnv},
		"UserConfigDir": {CapEnv},
		"UserCacheDir":  {CapEnv},
		// Process execution
		"StartProcess": {CapExec},
		// Method entries for *os.File.
		"File.Read":     {CapFileRead},
		"File.Write":    {CapFileWrite},
		"File.Stat":     {CapFileRead},
		"File.Chmod":    {CapFileWrite},
		"File.Chown":    {CapFileWrite},
		"File.Truncate": {CapFileWrite},
		"File.Sync":     {CapFileWrite},
	},
	"io/ioutil": {
		"ReadFile":  {CapFileRead},
		"WriteFile": {CapFileWrite},
		"ReadDir":   {CapFileRead},
		"TempFile":  {CapFileWrite},
		"TempDir":   {CapFileWrite},
	},
	"path/filepath": {
		"Glob":         {CapFileRead},
		"Walk":         {CapFileRead},
		"WalkDir":      {CapFileRead},
		"EvalSymlinks": {CapFileRead},
	},
	"fmt": {
		// Functions that implicitly write to os.Stdout.
		"Print":   {CapFileWrite},
		"Printf":  {CapFileWrite},
		"Println": {CapFileWrite},
		// Functions that implicitly read from os.Stdin.
		"Scan":   {CapFileRead},
		"Scanf":  {CapFileRead},
		"Scanln": {CapFileRead},
	},
	"log": {
		"Fatal":   {CapFileWrite},
		"Fatalf":  {CapFileWrite},
		"Fatalln": {CapFileWrite},
		"Panic":   {CapFileWrite},
		"Panicf":  {CapFileWrite},
		"Panicln": {CapFileWrite},
		"Print":   {CapFileWrite},
		"Printf":  {CapFileWrite},
		"Println": {CapFileWrite},
		"Output":  {CapFileWrite},
	},
	"net": {
		// Dial
		"Dial":               {CapNet},
		"DialTimeout":        {CapNet},
		"DialIP":             {CapNet},
		"DialTCP":            {CapNet},
		"DialUDP":            {CapNet},
		"DialUnix":           {CapNet},
		"Dialer.Dial":        {CapNet},
		"Dialer.DialContext": {CapNet},
		// Listen
		"Listen":       {CapNet},
		"ListenTCP":    {CapNet},
		"ListenUDP":    {CapNet},
		"ListenUnix":   {CapNet},
		"ListenPacket": {CapNet},
		// DNS lookup — package-level
		"LookupAddr":  {CapNet},
		"LookupCNAME": {CapNet},
		"LookupHost":  {CapNet},
		"LookupIP":    {CapNet},
		"LookupMX":    {CapNet},
		"LookupNS":    {CapNet},
		"LookupPort":  {CapNet},
		"LookupSRV":   {CapNet},
		"LookupTXT":   {CapNet},
		// DNS lookup — via Resolver
		"Resolver.LookupAddr":   {CapNet},
		"Resolver.LookupCNAME":  {CapNet},
		"Resolver.LookupHost":   {CapNet},
		"Resolver.LookupIP":     {CapNet},
		"Resolver.LookupIPAddr": {CapNet},
		"Resolver.LookupMX":     {CapNet},
		"Resolver.LookupNS":     {CapNet},
		"Resolver.LookupPort":   {CapNet},
		"Resolver.LookupSRV":    {CapNet},
		"Resolver.LookupTXT":    {CapNet},
		// Method entries for concrete net.Conn implementations.
		"TCPConn.Read":     {CapNet},
		"TCPConn.Write":    {CapNet},
		"TCPConn.ReadFrom": {CapNet},
		"UDPConn.Read":     {CapNet},
		"UDPConn.Write":    {CapNet},
		"UDPConn.ReadFrom": {CapNet},
		"UDPConn.WriteTo":  {CapNet},
		"UnixConn.Read":    {CapNet},
		"UnixConn.Write":   {CapNet},
		"conn.Read":        {CapNet},
		"conn.Write":       {CapNet},
	},
	"net/http": {
		"Get":               {CapNet},
		"Post":              {CapNet},
		"PostForm":          {CapNet},
		"Head":              {CapNet},
		"Do":                {CapNet},
		"Client.Do":         {CapNet},
		"Client.Get":        {CapNet},
		"Client.Post":       {CapNet},
		"Client.Head":       {CapNet},
		"Client.PostForm":   {CapNet},
		"ListenAndServe":    {CapNet},
		"ListenAndServeTLS": {CapNet},
		"Serve":             {CapNet},
		"ServeTLS":          {CapNet},
		"ServeFile":         {CapNet, CapFileRead},
	},
	"crypto/tls": {
		"Dial":           {CapNet},
		"DialWithDialer": {CapNet},
		"Listen":         {CapNet},
		"Server":         {CapNet},
		"Client":         {CapNet},
	},
	"net/smtp": {
		"Dial":                    {CapNet},
		"NewClient":               {CapNet},
		"SendMail":                {CapNet},
		"SMTP.Auth":               {CapNet},
		"SMTP.Data":               {CapNet},
		"SMTP.Extension":          {CapNet},
		"SMTP.Hello":              {CapNet},
		"SMTP.Mail":               {CapNet},
		"SMTP.Noop":               {CapNet},
		"SMTP.Quit":               {CapNet},
		"SMTP.Rcpt":               {CapNet},
		"SMTP.Reset":              {CapNet},
		"SMTP.StartTLS":           {CapNet},
		"SMTP.TLSConnectionState": {CapNet},
		"SMTP.Verify":             {CapNet},
	},
	"os/exec": {
		"Command":        {CapExec},
		"CommandContext": {CapExec},
		"LookPath":       {CapExec},
		// Method entries for *exec.Cmd.
		"Cmd.Run":            {CapExec},
		"Cmd.Start":          {CapExec},
		"Cmd.Output":         {CapExec},
		"Cmd.CombinedOutput": {CapExec},
		"Cmd.StdinPipe":      {CapExec},
		"Cmd.StdoutPipe":     {CapExec},
		"Cmd.StderrPipe":     {CapExec},
	},
	"syscall": {
		// Process execution
		"Exec":     {CapExec},
		"ForkExec": {CapExec},
		"Kill":     {CapExec},
		// File I/O
		"Open":   {CapFileRead, CapFileWrite},
		"Read":   {CapFileRead},
		"Write":  {CapFileWrite},
		"Creat":  {CapFileWrite},
		"Mkdir":  {CapFileWrite},
		"Rmdir":  {CapFileWrite},
		"Unlink": {CapFileWrite},
		"Rename": {CapFileWrite},
		"Chmod":  {CapFileWrite},
		"Chown":  {CapFileWrite},
		"Lstat":  {CapFileRead},
		"Stat":   {CapFileRead},
		// Network
		"Socket":  {CapNet},
		"Bind":    {CapNet},
		"Listen":  {CapNet},
		"Accept":  {CapNet},
		"Accept4": {CapNet},
		"Connect": {CapNet},
		// Environment
		"Getenv":   {CapEnv},
		"Setenv":   {CapEnv},
		"Unsetenv": {CapEnv},
		"Clearenv": {CapEnv},
	},
}

// requiredOnImport maps stdlib package paths whose use signals a
// trust-escape capability. Importing any of these gives the package the
// means to bypass the static, selector-based view of stdlib usage:
//
//   - "C"       — cgo can call any C function, invisible to the table.
//   - "unsafe"  — pointer arithmetic / hand-built function pointers.
//   - "plugin"  — loads arbitrary .so code at runtime.
//   - "reflect" — dynamic dispatch (e.g. MethodByName.Call) reaches
//     methods that may never appear as a static selector.
//
// CapUnsafe is charged wholesale on import rather than per-symbol because
// each of these surfaces has many ways to escape and tracking them
// individually is brittle.
var requiredOnImport = map[string][]Cap{
	"C":       {CapUnsafe},
	"unsafe":  {CapUnsafe},
	"plugin":  {CapUnsafe},
	"reflect": {CapUnsafe},
}

// All returns the canonical ordering of caps used for stable diagnostics.
func All() []Cap {
	return []Cap{
		CapFileRead,
		CapFileWrite,
		CapNet,
		CapExec,
		CapEnv,
		CapUnsafe,
	}
}

// Parse parses a single capability token (e.g. "file:write"). Surrounding
// whitespace is trimmed. Returns ok=false for unknown tokens.
func Parse(s string) (Cap, bool) {
	switch Cap(strings.TrimSpace(s)) {
	case CapFileRead:
		return CapFileRead, true
	case CapFileWrite:
		return CapFileWrite, true
	case CapNet:
		return CapNet, true
	case CapExec:
		return CapExec, true
	case CapEnv:
		return CapEnv, true
	case CapUnsafe:
		return CapUnsafe, true
	}
	return "", false
}

// ParseList parses a comma-separated list of capability tokens (the body
// after "//caps:"). It returns the successfully parsed caps and the slice
// of raw (trimmed) tokens that failed to parse so the caller can emit one
// diagnostic per bad token. Empty tokens (e.g. trailing commas) are
// ignored.
func ParseList(s string) ([]Cap, []string) {
	var caps []Cap
	var bad []string
	for _, raw := range strings.Split(s, ",") {
		tok := strings.TrimSpace(raw)
		if tok == "" {
			continue
		}
		if c, ok := Parse(tok); ok {
			caps = append(caps, c)
		} else {
			bad = append(bad, tok)
		}
	}
	return caps, bad
}

// RequiredOnImport returns the caps charged for importing pkgPath
// wholesale, independent of which symbols are used. Returns nil for
// imports that do not, on their own, require a capability.
func RequiredOnImport(pkgPath string) []Cap {
	caps, ok := requiredOnImport[pkgPath]
	if !ok {
		return nil
	}
	out := make([]Cap, len(caps))
	copy(out, caps)
	return out
}

// Required returns the caps required by referencing pkgPath.name. Returns
// nil if the symbol does not require a capability or is unknown. The name
// may be "FuncName" or "TypeName.MethodName".
func Required(pkgPath, name string) []Cap {
	syms, ok := required[pkgPath]
	if !ok {
		return nil
	}
	caps, ok := syms[name]
	if !ok {
		return nil
	}
	// Return a copy so callers can't mutate the table.
	out := make([]Cap, len(caps))
	copy(out, caps)
	return out
}

// IsStdlib reports whether pkgPath refers to a standard library package.
// The heuristic: a package path is stdlib iff the first '/'-separated
// segment contains no '.'. The empty string counts as stdlib. Vendored
// standard-library dependencies ("vendor/golang.org/x/...") are
// correctly classified as stdlib.
func IsStdlib(pkgPath string) bool {
	if pkgPath == "" {
		return true
	}
	first := pkgPath
	if i := strings.IndexByte(pkgPath, '/'); i >= 0 {
		first = pkgPath[:i]
	}
	return !strings.ContainsRune(first, '.')
}

// Set is an unordered collection of caps.
type Set map[Cap]struct{}

// NewSet returns a Set containing the given caps.
func NewSet(caps ...Cap) Set {
	s := make(Set, len(caps))
	for _, c := range caps {
		s[c] = struct{}{}
	}
	return s
}

// Add inserts c into the set.
func (s Set) Add(c Cap) {
	s[c] = struct{}{}
}

// Has reports whether c is in the set.
func (s Set) Has(c Cap) bool {
	_, ok := s[c]
	return ok
}

// Diff returns a new Set containing caps in s that are not in other (s \ other).
func (s Set) Diff(other Set) Set {
	out := make(Set, len(s))
	for c := range s {
		if !other.Has(c) {
			out[c] = struct{}{}
		}
	}
	return out
}

// Union returns a new Set containing every cap present in s or other.
func (s Set) Union(other Set) Set {
	out := make(Set, len(s)+len(other))
	for c := range s {
		out[c] = struct{}{}
	}
	for c := range other {
		out[c] = struct{}{}
	}
	return out
}

// Sorted returns the caps in canonical order (matching All). Any caps
// outside the canonical list are appended in lexical order so callers
// never silently lose data.
func (s Set) Sorted() []Cap {
	out := make([]Cap, 0, len(s))
	seen := make(map[Cap]bool, len(s))
	for _, c := range All() {
		if _, ok := s[c]; ok {
			out = append(out, c)
			seen[c] = true
		}
	}
	var extra []Cap
	for c := range s {
		if !seen[c] {
			extra = append(extra, c)
		}
	}
	sort.Slice(extra, func(i, j int) bool { return extra[i] < extra[j] })
	return append(out, extra...)
}
