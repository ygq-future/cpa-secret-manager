package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef int (*cliproxy_host_call_fn)(void*, const char*, const uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_host_free_fn)(void*, size_t);

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	cliproxy_host_call_fn call;
	cliproxy_host_free_fn free_buffer;
} cliproxy_host_api;

typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);

typedef struct {
	uint32_t abi_version;
	cliproxy_plugin_call_fn call;
	cliproxy_plugin_free_fn free_buffer;
	cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

extern int cpaSecretManagerPluginCall(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cpaSecretManagerPluginFree(void*, size_t);
extern void cpaSecretManagerPluginShutdown(void);
*/
import "C"

import (
	"context"
	"unsafe"

	pluginruntime "cpa-secret-manager/internal/runtime"
)

const abiVersion uint32 = 1

// cpaRuntime is the single application-layer instance owned by this plugin.
var cpaRuntime = pluginruntime.New(pluginruntime.Options{})

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(host *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if plugin == nil {
		return -1
	}
	// The host callback table is intentionally unused: this plugin implements no
	// host capability (no credential access, no upstream HTTP, no model calls).
	_ = host
	plugin.abi_version = C.uint32_t(abiVersion)
	plugin.call = C.cliproxy_plugin_call_fn(C.cpaSecretManagerPluginCall)
	plugin.free_buffer = C.cliproxy_plugin_free_fn(C.cpaSecretManagerPluginFree)
	plugin.shutdown = C.cliproxy_plugin_shutdown_fn(C.cpaSecretManagerPluginShutdown)
	return 0
}

//export cpaSecretManagerPluginCall
func cpaSecretManagerPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response != nil {
		response.ptr = nil
		response.len = 0
	}
	if method == nil {
		writeResponse(response, []byte(`{"ok":false,"error":{"code":"invalid_method","message":"method is required"}}`))
		return 1
	}

	payload := copyRequestBytes(request, requestLen)
	writeResponse(response, cpaRuntime.Handle(context.Background(), C.GoString(method), payload))
	return 0
}

//export cpaSecretManagerPluginFree
func cpaSecretManagerPluginFree(ptr unsafe.Pointer, length C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
	_ = length
}

//export cpaSecretManagerPluginShutdown
func cpaSecretManagerPluginShutdown() {
	_ = cpaRuntime.Shutdown(context.Background())
}

func copyRequestBytes(request *C.uint8_t, requestLen C.size_t) []byte {
	if request == nil || requestLen == 0 {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(request), C.int(requestLen))
}

func writeResponse(response *C.cliproxy_buffer, raw []byte) {
	if response == nil || len(raw) == 0 {
		return
	}
	ptr := C.CBytes(raw)
	if ptr == nil {
		return
	}
	response.ptr = ptr
	response.len = C.size_t(len(raw))
}
