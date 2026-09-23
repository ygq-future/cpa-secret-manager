// Command abi_probe loads a built plugin dynamic library and exercises the C ABI
// contract: symbol presence, init handshake, register, management route
// dispatch, buffer freeing and shutdown.
//
// Usage: abi_probe <plugin-dynamic-library> <state-path>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

// cliproxyBuffer mirrors the C cliproxy_buffer struct.
type cliproxyBuffer struct {
	ptr unsafe.Pointer
	len uintptr
}

// hostAPI mirrors cliproxy_host_api. The plugin under test implements no host
// capability, so the callback table stays zeroed.
type hostAPI struct {
	abiVersion uint32
	_          uint32
	hostCtx    uintptr
	call       uintptr
	freeBuffer uintptr
}

// pluginAPI mirrors cliproxy_plugin_api without the padding ambiguity.
type pluginAPI struct {
	abiVersion uint32
	_          uint32
	call       uintptr
	freeBuffer uintptr
	shutdown   uintptr
}

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type managementResponse struct {
	StatusCode int    `json:"StatusCode"`
	Body       []byte `json:"Body"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ABI_PROBE_FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("VERIFY_CGO_ABI_SUCCESS")
}

func run() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: abi_probe <plugin-dynamic-library> <state-path>")
	}
	libraryPath := os.Args[1]
	statePath := os.Args[2]

	library, err := syscall.LoadDLL(libraryPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", libraryPath, err)
	}
	defer func() {
		_ = library.Release()
	}()

	initProc, err := library.FindProc("cliproxy_plugin_init")
	if err != nil {
		return fmt.Errorf("find cliproxy_plugin_init: %w", err)
	}
	if _, err := library.FindProc("cpaSecretManagerPluginCall"); err != nil {
		return fmt.Errorf("find cpaSecretManagerPluginCall: %w", err)
	}
	if _, err := library.FindProc("cpaSecretManagerPluginFree"); err != nil {
		return fmt.Errorf("find cpaSecretManagerPluginFree: %w", err)
	}
	if _, err := library.FindProc("cpaSecretManagerPluginShutdown"); err != nil {
		return fmt.Errorf("find cpaSecretManagerPluginShutdown: %w", err)
	}

	if rc, _, _ := initProc.Call(0, 0); int32(uint32(rc)) != -1 {
		return fmt.Errorf("cliproxy_plugin_init(nil, nil) = %d, want -1", int32(uint32(rc)))
	}

	var host hostAPI
	var plugin pluginAPI
	rc, _, _ := initProc.Call(uintptr(unsafe.Pointer(&host)), uintptr(unsafe.Pointer(&plugin)))
	if int32(uint32(rc)) != 0 {
		return fmt.Errorf("cliproxy_plugin_init(host, plugin) = %d, want 0", int32(uint32(rc)))
	}
	if plugin.abiVersion != 1 {
		return fmt.Errorf("plugin abi_version = %d, want 1", plugin.abiVersion)
	}
	if plugin.call == 0 || plugin.freeBuffer == 0 || plugin.shutdown == 0 {
		return fmt.Errorf("plugin function table is incomplete: %+v", plugin)
	}

	call := func(method string, payload []byte) (envelope, error) {
		methodPtr, err := syscall.BytePtrFromString(method)
		if err != nil {
			return envelope{}, err
		}
		var payloadPtr uintptr
		if len(payload) > 0 {
			payloadPtr = uintptr(unsafe.Pointer(&payload[0]))
		}

		var response cliproxyBuffer
		callRC, _, _ := syscall.SyscallN(plugin.call,
			uintptr(unsafe.Pointer(methodPtr)), payloadPtr, uintptr(len(payload)), uintptr(unsafe.Pointer(&response)))
		runtime.KeepAlive(payload)
		runtime.KeepAlive(methodPtr)

		if int32(uint32(callRC)) != 0 {
			return envelope{}, fmt.Errorf("%s returned status %d", method, int32(uint32(callRC)))
		}
		if response.ptr == nil || response.len == 0 {
			return envelope{}, fmt.Errorf("%s returned an empty buffer", method)
		}

		raw := unsafe.Slice((*byte)(response.ptr), response.len)
		body := make([]byte, len(raw))
		copy(body, raw)
		syscall.SyscallN(plugin.freeBuffer, uintptr(response.ptr), response.len)

		var env envelope
		if err := json.Unmarshal(body, &env); err != nil {
			return envelope{}, fmt.Errorf("decode %s response %q: %w", method, body, err)
		}
		if !env.OK {
			return envelope{}, fmt.Errorf("%s failed: %+v", method, env.Error)
		}
		return env, nil
	}

	registerPayload, err := json.Marshal(map[string]string{
		"config_yaml": "state_path: " + filepathSlash(statePath) + "\n",
	})
	if err != nil {
		return err
	}
	registerEnv, err := call("plugin.register", registerPayload)
	if err != nil {
		return err
	}
	var metadata struct {
		Name    string `json:"Name"`
		Version string `json:"Version"`
	}
	var registration struct {
		SchemaVersion int             `json:"schema_version"`
		Metadata      json.RawMessage `json:"metadata"`
		Capabilities  map[string]bool `json:"capabilities"`
	}
	if err := json.Unmarshal(registerEnv.Result, &registration); err != nil {
		return fmt.Errorf("decode registration: %w", err)
	}
	if err := json.Unmarshal(registration.Metadata, &metadata); err != nil {
		return fmt.Errorf("decode metadata: %w", err)
	}
	if registration.SchemaVersion != 1 {
		return fmt.Errorf("schema_version = %d, want 1", registration.SchemaVersion)
	}
	if metadata.Name == "" || metadata.Version == "" {
		return fmt.Errorf("metadata is incomplete: %+v", metadata)
	}
	if !registration.Capabilities["management_api"] {
		return fmt.Errorf("management_api capability missing: %+v", registration.Capabilities)
	}

	if _, err := call("management.register", nil); err != nil {
		return err
	}

	settingsRequest, err := json.Marshal(map[string]string{
		"Method": "GET",
		"Path":   "/v0/management/plugins/cpa-secret-manager/settings",
	})
	if err != nil {
		return err
	}
	settingsEnv, err := call("management.handle", settingsRequest)
	if err != nil {
		return err
	}
	var response managementResponse
	if err := json.Unmarshal(settingsEnv.Result, &response); err != nil {
		return fmt.Errorf("decode management response: %w", err)
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("settings status = %d, want 200", response.StatusCode)
	}
	var settings struct {
		UsageEnabled bool   `json:"usage_enabled"`
		StatePath    string `json:"state_path"`
	}
	if err := json.Unmarshal(response.Body, &settings); err != nil {
		return fmt.Errorf("decode settings %q: %w", response.Body, err)
	}
	if !settings.UsageEnabled {
		return fmt.Errorf("usage_enabled = false, want the default true")
	}
	if filepathSlash(settings.StatePath) != filepathSlash(statePath) {
		return fmt.Errorf("state_path = %q, want %q", settings.StatePath, statePath)
	}

	for index := range 5 {
		if _, err := call("management.handle", settingsRequest); err != nil {
			return fmt.Errorf("repeat call %d: %w", index, err)
		}
	}

	syscall.SyscallN(plugin.shutdown)
	return nil
}

func filepathSlash(path string) string {
	out := make([]byte, 0, len(path))
	for index := range len(path) {
		if path[index] == '\\' {
			out = append(out, '/')
			continue
		}
		out = append(out, path[index])
	}
	return string(out)
}
