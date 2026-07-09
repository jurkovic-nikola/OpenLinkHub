//go:build linux && cgo

package nvidiagpu

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

typedef int32_t  NV_S32;
typedef uint32_t NV_U32;
typedef uint8_t  NV_U8;
typedef NV_S32   NV_STATUS;
typedef NV_S32*  NV_HANDLE;
typedef NV_HANDLE NV_PHYSICAL_GPU_HANDLE;
typedef char NV_SHORT_STRING[64];

typedef void* (*nvapi_QueryInterface_t)(NV_U32);
typedef NV_STATUS (*NvAPI_Initialize_t)(void);
typedef NV_STATUS (*NvAPI_EnumPhysicalGPUs_t)(NV_PHYSICAL_GPU_HANDLE*, NV_S32*);
typedef NV_STATUS (*NvAPI_GPU_GetFullName_t)(NV_PHYSICAL_GPU_HANDLE, NV_SHORT_STRING);
typedef NV_STATUS (*NvAPI_GPU_GetPCIIdentifiers_t)(NV_PHYSICAL_GPU_HANDLE, NV_U32*, NV_U32*, NV_U32*, NV_U32*);

#define OLH_NVAPI_OK 0
#define OLH_NVAPI_MAX_PHYSICAL_GPUS 64
#define OLH_NVIDIA_VENDOR 0x10de
#define OLH_NVIDIA_TITANRTX_DEVICE 0x1e02
#define OLH_NVIDIA_TITANRTX_SUBSYSTEM 0x12a3
#define OLH_NVIDIA_ILLUM_ZONE_COUNT_MAX 32
#define OLH_NVIDIA_ILLUM_PARAMS_VERSION 72012

#define OLH_NVIDIA_ZONE_TYPE_RGB 1
#define OLH_NVIDIA_ZONE_TYPE_COLOR_FIXED 2
#define OLH_NVIDIA_ZONE_TYPE_RGBW 3
#define OLH_NVIDIA_ZONE_TYPE_SINGLE_COLOR 4
#define OLH_NVIDIA_CTRL_MODE_MANUAL_RGB 0

typedef struct {
    NV_S32 type;
    NV_S32 ctrlMode;
    NV_U8  data[128];
    NV_U8  rsvd[64];
} olh_nvapi_zone_control;

typedef struct {
    NV_U32 version;
    NV_U32 bDefault;
    NV_U32 numIllumZonesControl;
    NV_U8  rsvd[64];
    olh_nvapi_zone_control zones[OLH_NVIDIA_ILLUM_ZONE_COUNT_MAX];
} olh_nvapi_zone_control_params;

typedef NV_STATUS (*NvAPI_GPU_ClientIllumZonesGetControl_t)(NV_PHYSICAL_GPU_HANDLE, olh_nvapi_zone_control_params*);
typedef NV_STATUS (*NvAPI_GPU_ClientIllumZonesSetControl_t)(NV_PHYSICAL_GPU_HANDLE, olh_nvapi_zone_control_params*);

typedef struct {
    NV_PHYSICAL_GPU_HANDLE handle;
    NV_U32 device_id;
    NV_U32 sub_system_id;
    NV_U32 revision_id;
    NV_U32 ext_device_id;
    int zone_count;
    int zone_types[OLH_NVIDIA_ILLUM_ZONE_COUNT_MAX];
    int treats_rgbw_as_rgb;
    char name[64];
} olh_nvapi_gpu;

static void* olh_nvapi_lib = 0;
static NvAPI_Initialize_t olh_nvapi_initialize = 0;
static NvAPI_EnumPhysicalGPUs_t olh_nvapi_enum_physical_gpus = 0;
static NvAPI_GPU_GetFullName_t olh_nvapi_gpu_get_full_name = 0;
static NvAPI_GPU_GetPCIIdentifiers_t olh_nvapi_gpu_get_pci_identifiers = 0;
static NvAPI_GPU_ClientIllumZonesGetControl_t olh_nvapi_gpu_client_illum_zones_get_control = 0;
static NvAPI_GPU_ClientIllumZonesSetControl_t olh_nvapi_gpu_client_illum_zones_set_control = 0;

static void olh_set_error(char* err, int err_len, const char* msg) {
    if(err && err_len > 0) {
        snprintf(err, (size_t)err_len, "%s", msg);
    }
}

static void* olh_query(nvapi_QueryInterface_t query, NV_U32 id) {
    if(!query) {
        return 0;
    }
    return query(id);
}

static int olh_nvapi_load(char* err, int err_len) {
    nvapi_QueryInterface_t query = 0;

    if(!olh_nvapi_lib) {
        olh_nvapi_lib = dlopen("libnvidia-api.so.1", RTLD_LAZY);
        if(!olh_nvapi_lib) {
            olh_nvapi_lib = dlopen("libnvidia-api.so", RTLD_LAZY);
        }
        if(!olh_nvapi_lib) {
            olh_set_error(err, err_len, "unable to load libnvidia-api.so");
            return -1;
        }
    }

    query = (nvapi_QueryInterface_t)dlsym(olh_nvapi_lib, "nvapi_QueryInterface");
    if(!query) {
        olh_set_error(err, err_len, "unable to load nvapi_QueryInterface");
        return -1;
    }

    if(!olh_nvapi_initialize) {
        olh_nvapi_initialize = (NvAPI_Initialize_t)olh_query(query, 0x0150E828);
        olh_nvapi_enum_physical_gpus = (NvAPI_EnumPhysicalGPUs_t)olh_query(query, 0xE5AC921F);
        olh_nvapi_gpu_get_full_name = (NvAPI_GPU_GetFullName_t)olh_query(query, 0x0CEEE8E9F);
        olh_nvapi_gpu_get_pci_identifiers = (NvAPI_GPU_GetPCIIdentifiers_t)olh_query(query, 0x2DDFB66E);
        olh_nvapi_gpu_client_illum_zones_get_control = (NvAPI_GPU_ClientIllumZonesGetControl_t)olh_query(query, 0x3DBF5764);
        olh_nvapi_gpu_client_illum_zones_set_control = (NvAPI_GPU_ClientIllumZonesSetControl_t)olh_query(query, 0x197D065E);
    }

    if(!olh_nvapi_initialize ||
       !olh_nvapi_enum_physical_gpus ||
       !olh_nvapi_gpu_get_pci_identifiers ||
       !olh_nvapi_gpu_client_illum_zones_get_control ||
       !olh_nvapi_gpu_client_illum_zones_set_control) {
        olh_set_error(err, err_len, "required NVAPI illumination functions are unavailable");
        return -1;
    }

    NV_STATUS status = olh_nvapi_initialize();
    if(status != OLH_NVAPI_OK) {
        snprintf(err, (size_t)err_len, "NvAPI_Initialize returned %d", status);
        return status;
    }
    return OLH_NVAPI_OK;
}

static int olh_nvapi_is_titan_rtx(NV_U32 device_id, NV_U32 sub_system_id) {
    NV_U32 pci_vendor = device_id & 0xffff;
    NV_U32 pci_device = device_id >> 16;
    NV_U32 pci_subsystem_vendor = sub_system_id & 0xffff;
    NV_U32 pci_subsystem_device = sub_system_id >> 16;

    return pci_vendor == OLH_NVIDIA_VENDOR &&
           pci_device == OLH_NVIDIA_TITANRTX_DEVICE &&
           pci_subsystem_vendor == OLH_NVIDIA_VENDOR &&
           pci_subsystem_device == OLH_NVIDIA_TITANRTX_SUBSYSTEM;
}

static int olh_nvapi_detect_titan(olh_nvapi_gpu* out, int max, char* err, int err_len) {
    NV_PHYSICAL_GPU_HANDLE gpu_handles[OLH_NVAPI_MAX_PHYSICAL_GPUS];
    NV_S32 gpu_count = 0;
    int found = 0;
    int load_status = olh_nvapi_load(err, err_len);
    if(load_status != OLH_NVAPI_OK) {
        return -1;
    }

    memset(gpu_handles, 0, sizeof(gpu_handles));
    NV_STATUS status = olh_nvapi_enum_physical_gpus(gpu_handles, &gpu_count);
    if(status != OLH_NVAPI_OK) {
        snprintf(err, (size_t)err_len, "NvAPI_EnumPhysicalGPUs returned %d", status);
        return -1;
    }

    for(NV_S32 i = 0; i < gpu_count && found < max; i++) {
        NV_U32 device_id = 0;
        NV_U32 sub_system_id = 0;
        NV_U32 revision_id = 0;
        NV_U32 ext_device_id = 0;
        olh_nvapi_zone_control_params params;

        status = olh_nvapi_gpu_get_pci_identifiers(
            gpu_handles[i],
            &device_id,
            &sub_system_id,
            &revision_id,
            &ext_device_id
        );
        if(status != OLH_NVAPI_OK || !olh_nvapi_is_titan_rtx(device_id, sub_system_id)) {
            continue;
        }

        memset(&params, 0, sizeof(params));
        params.version = OLH_NVIDIA_ILLUM_PARAMS_VERSION;
        params.bDefault = 0;
        status = olh_nvapi_gpu_client_illum_zones_get_control(gpu_handles[i], &params);
        usleep(30000);
        if(status != OLH_NVAPI_OK || params.numIllumZonesControl == 0) {
            continue;
        }

        memset(&out[found], 0, sizeof(out[found]));
        out[found].handle = gpu_handles[i];
        out[found].device_id = device_id;
        out[found].sub_system_id = sub_system_id;
        out[found].revision_id = revision_id;
        out[found].ext_device_id = ext_device_id;
        out[found].zone_count = params.numIllumZonesControl > OLH_NVIDIA_ILLUM_ZONE_COUNT_MAX ? OLH_NVIDIA_ILLUM_ZONE_COUNT_MAX : (int)params.numIllumZonesControl;
        out[found].treats_rgbw_as_rgb = 0;
        snprintf(out[found].name, sizeof(out[found].name), "NVIDIA TITAN RTX");

        if(olh_nvapi_gpu_get_full_name) {
            NV_SHORT_STRING name;
            memset(name, 0, sizeof(name));
            if(olh_nvapi_gpu_get_full_name(gpu_handles[i], name) == OLH_NVAPI_OK && name[0] != 0) {
                snprintf(out[found].name, sizeof(out[found].name), "%s", name);
            }
        }

        for(int zone = 0; zone < out[found].zone_count; zone++) {
            out[found].zone_types[zone] = params.zones[zone].type;
        }
        found++;
    }

    return found;
}

static int olh_nvapi_set_zone(NV_PHYSICAL_GPU_HANDLE handle, int zone, NV_U8 red, NV_U8 green, NV_U8 blue, NV_U8 brightness, int treats_rgbw_as_rgb, char* err, int err_len) {
    olh_nvapi_zone_control_params params;
    NV_STATUS status;

    if(!handle) {
        olh_set_error(err, err_len, "empty NVAPI GPU handle");
        return -1;
    }

    if(olh_nvapi_load(err, err_len) != OLH_NVAPI_OK) {
        return -1;
    }

    memset(&params, 0, sizeof(params));
    params.version = OLH_NVIDIA_ILLUM_PARAMS_VERSION;
    params.bDefault = 0;
    status = olh_nvapi_gpu_client_illum_zones_get_control(handle, &params);
    usleep(30000);
    if(status != OLH_NVAPI_OK) {
        snprintf(err, (size_t)err_len, "NvAPI_GPU_ClientIllumZonesGetControl returned %d", status);
        return status;
    }

    if(zone < 0 || (NV_U32)zone >= params.numIllumZonesControl || zone >= OLH_NVIDIA_ILLUM_ZONE_COUNT_MAX) {
        olh_set_error(err, err_len, "invalid NVAPI illumination zone");
        return -2;
    }

    olh_nvapi_zone_control* z = &params.zones[zone];
    z->ctrlMode = OLH_NVIDIA_CTRL_MODE_MANUAL_RGB;

    switch(z->type) {
    case OLH_NVIDIA_ZONE_TYPE_RGB:
        z->data[0] = red;
        z->data[1] = green;
        z->data[2] = blue;
        z->data[3] = brightness;
        break;
    case OLH_NVIDIA_ZONE_TYPE_RGBW: {
        NV_U8 white = 0;
        if(!treats_rgbw_as_rgb) {
            NV_U8 min_rgb = red;
            NV_U8 max_rgb = red;
            if(green < min_rgb) min_rgb = green;
            if(blue < min_rgb) min_rgb = blue;
            if(green > max_rgb) max_rgb = green;
            if(blue > max_rgb) max_rgb = blue;
            if((NV_U8)(max_rgb - min_rgb) <= 10) {
                red = 0;
                green = 0;
                blue = 0;
                white = (NV_U8)(((NV_U32)max_rgb + (NV_U32)min_rgb) / 2);
            }
        }
        z->data[0] = red;
        z->data[1] = green;
        z->data[2] = blue;
        z->data[3] = white;
        z->data[4] = brightness;
        break;
    }
    case OLH_NVIDIA_ZONE_TYPE_SINGLE_COLOR:
    case OLH_NVIDIA_ZONE_TYPE_COLOR_FIXED:
        z->data[0] = (red == 0 && green == 0 && blue == 0) ? 0 : brightness;
        break;
    default:
        snprintf(err, (size_t)err_len, "unsupported NVAPI illumination zone type %d", z->type);
        return -3;
    }

    status = olh_nvapi_gpu_client_illum_zones_set_control(handle, &params);
    usleep(30000);
    if(status != OLH_NVAPI_OK) {
        snprintf(err, (size_t)err_len, "NvAPI_GPU_ClientIllumZonesSetControl returned %d", status);
        return status;
    }

    return OLH_NVAPI_OK;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

func detectNativeGPUs() ([]nativeGPU, error) {
	const maxGPUs = 8
	var detected [maxGPUs]C.olh_nvapi_gpu
	errBuf := (*C.char)(C.malloc(256))
	defer C.free(unsafe.Pointer(errBuf))
	C.memset(unsafe.Pointer(errBuf), 0, 256)

	count := C.olh_nvapi_detect_titan((*C.olh_nvapi_gpu)(unsafe.Pointer(&detected[0])), C.int(maxGPUs), errBuf, 256)
	if count < 0 {
		return nil, errors.New(C.GoString(errBuf))
	}
	if count == 0 {
		return nil, nil
	}

	gpus := make([]nativeGPU, 0, int(count))
	for i := 0; i < int(count); i++ {
		gpu := detected[i]
		zones := make([]int, 0, int(gpu.zone_count))
		for zone := 0; zone < int(gpu.zone_count); zone++ {
			zones = append(zones, int(gpu.zone_types[zone]))
		}
		gpus = append(gpus, nativeGPU{
			Handle:          uintptr(unsafe.Pointer(gpu.handle)),
			DeviceID:        uint32(gpu.device_id),
			SubSystemID:     uint32(gpu.sub_system_id),
			RevisionID:      uint32(gpu.revision_id),
			ExtDeviceID:     uint32(gpu.ext_device_id),
			Name:            C.GoString(&gpu.name[0]),
			Zones:           zones,
			TreatsRGBWAsRGB: int(gpu.treats_rgbw_as_rgb) != 0,
		})
	}

	return gpus, nil
}

func setNativeZone(handle uintptr, zone int, red, green, blue, brightness uint8, treatsRGBWAsRGB bool) error {
	errBuf := (*C.char)(C.malloc(256))
	defer C.free(unsafe.Pointer(errBuf))
	C.memset(unsafe.Pointer(errBuf), 0, 256)

	treats := C.int(0)
	if treatsRGBWAsRGB {
		treats = 1
	}
	status := C.olh_nvapi_set_zone(
		(C.NV_PHYSICAL_GPU_HANDLE)(unsafe.Pointer(handle)),
		C.int(zone),
		C.NV_U8(red),
		C.NV_U8(green),
		C.NV_U8(blue),
		C.NV_U8(brightness),
		treats,
		errBuf,
		256,
	)
	if status != 0 {
		return errors.New(C.GoString(errBuf))
	}
	return nil
}
