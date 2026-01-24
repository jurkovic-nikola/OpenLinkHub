/*
 * audio.c
 *
 * PipeWire-based virtual audio engine with real-time DSP.
 *
 * Author:
 *   Nikola Jurkovic
 *
 * License:
 *   GPL-3.0 or later
 */
#include "audio.h"

#include <pipewire/pipewire.h>
#include <pipewire/stream.h>
#include <pipewire/keys.h>
#include <pipewire/loop.h>
#include <spa/param/audio/format-utils.h>
#include <spa/pod/builder.h>

#include <math.h>
#include <stdint.h>
#include <stdatomic.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>

#define BANDS 10
#define CH 2
#define MAX_SINKS 64
#define DSP_BLOCK_FRAMES 512

typedef struct {
	uint32_t id;
	uint32_t serial;
	char name[128];
	char desc[256];
} sink_info_t;

static sink_info_t sinks[MAX_SINKS];
static atomic_int sink_count = 0;
static pthread_mutex_t sinks_mu = PTHREAD_MUTEX_INITIALIZER;
static char last_error[256];

typedef struct {
	float b0,b1,b2,a1,a2;
} biquad_coeff_t;

typedef struct {
	float z1,z2;
} biquad_state_t;

typedef struct {
	float *data;
	uint32_t size;
	atomic_uint wpos;
	atomic_uint rpos;
	uint32_t channels;
} ring_t;

typedef struct {
	struct pw_main_loop *loop;
	struct pw_context *context;
	struct pw_core *core;

	struct pw_stream *cap;
	struct spa_hook cap_listener;

	_Atomic(struct pw_stream *) pb;
	struct spa_hook pb_listener;

	uint32_t rate;
	uint32_t channels;

	float freqs[BANDS];

	atomic_int band_mdb[BANDS];
	atomic_uint master_gain_bits;

	biquad_state_t state[CH][BANDS];

	biquad_coeff_t coeff_buf[2][CH][BANDS];
	_Atomic(biquad_coeff_t (*)[CH][BANDS]) coeff_ptr;
	int coeff_write_idx;

	ring_t ring;

	uint32_t target_serial;
	atomic_int reconnect_pb;

	atomic_int pb_streaming;
	atomic_int quit;
} app_t;

static app_t app;

typedef struct {
	uint32_t rate;
	uint32_t channels;
	uint32_t polling_rate;
	uint32_t debug;
	uint32_t ring_frames;
	char latency[32];
	char max_latency[32];
	char preferred_sink_name[128];
	char preferred_sink_desc[256];
} pw_config_t;

static pw_config_t pw_cfg = {
	.rate			= 48000,
	.channels		= 2,
	.polling_rate	= 10,
	.debug 			= 0,
	.ring_frames    = 512,
	.latency		= "128/48000",
	.max_latency	= "256/48000"
};

typedef struct {
	const char *name;
	const char *desc;
	const char *media_class;
	const char *role;
	const char *node_group;
	const char *link_group;
	const char *type;
	const char *category;
} pw_device_desc_t;

static const pw_device_desc_t DEVICE_SINK_DESC = {
	.name           = "openlinkhub-virtual-device",
	.desc           = "OpenLinkHub Virtual Audio Device",
	.media_class	= "Audio/Sink",
	.role           = "DSP",
	.node_group     = "openlinkhub-audio",
	.link_group     = "openlinkhub-audio"
};

static const pw_device_desc_t DEVICE_PLAYBACK_DESC = {
	.name       = "openlinkhub-virtual-device-playback",
	.type       = "Audio",
	.category   = "Playback",
	.role       = "Music",
	.node_group = "openlinkhub-audio",
	.link_group = "openlinkhub-audio"
};

static atomic_int pw_started = 0;
static atomic_int pw_running = 0;

static inline void log_debug(const char *fmt, ...)
{
	if (!pw_cfg.debug)
	{
		return;
	}

	struct timespec ts;
	clock_gettime(CLOCK_MONOTONIC, &ts);

	fprintf(stderr,
		"[DEBUG %ld.%03ld] ",
		ts.tv_sec,
		ts.tv_nsec / 1000000
	);

	va_list ap;
	va_start(ap, fmt);
	vfprintf(stderr, fmt, ap);
	va_end(ap);
}

static void set_error(const char *msg)
{
	snprintf(
		last_error,
		sizeof(last_error),
		"%s",
		msg
	);
}

const char* audio_engine_last_error(void)
{
	return last_error;
}

static inline void biquad_state_reset(biquad_state_t *s)
{
	s->z1 = 0.f;
	s->z2 = 0.f;
}

static inline float biquad_process(
	const biquad_coeff_t *c,
	biquad_state_t *s,
	float x)
{
	float y = c->b0 * x + s->z1;
	s->z1 = c->b1 * x - c->a1 * y + s->z2;
	s->z2 = c->b2 * x - c->a2 * y;
	return y;
}

static inline void biquad_set_peaking_coeff(
	biquad_coeff_t *c,
	float fs,
	float f0,
	float Q,
	float gainDB)
{
	if (fs <= 0.f)
	{
		fs = 48000.f;
	}

	if (f0 <= 0.f)
	{
		f0 = 10.f;
	}

	if (f0 > fs * 0.49f)
	{
		f0 = fs * 0.49f;
	}

	if (Q <= 0.f)
	{
		Q = 0.707f;
	}

	float A = powf(10.f, gainDB / 40.f);
	float w0 = 2.f * (float)M_PI * f0 / fs;
	float cw = cosf(w0), sw = sinf(w0);
	float alpha = sw / (2.f * Q);

	float b0 = 1 + alpha * A;
	float b1 = -2 * cw;
	float b2 = 1 - alpha * A;
	float a0 = 1 + alpha / A;
	float a1 = -2 * cw;
	float a2 = 1 - alpha / A;

	c->b0 = b0 / a0;
	c->b1 = b1 / a0;
	c->b2 = b2 / a0;
	c->a1 = a1 / a0;
	c->a2 = a2 / a0;
}

static inline uint32_t f32_to_bits(float f)
{
	union
	{
		float f;
		uint32_t u;
	} v;

	v.f=f;
	return v.u;
}

static inline float bits_to_f32(uint32_t u)
{
	union
	{
		float f;
		uint32_t u;
	} v;

	v.u=u;
	return v.f;
}

static inline uint32_t ring_used_frames(ring_t *r)
{
	uint32_t rp = atomic_load_explicit(&r->rpos, memory_order_relaxed);
	uint32_t wp = atomic_load_explicit(&r->wpos, memory_order_acquire);

	if (wp >= rp)
	{
		return wp - rp;
	}

	return (r->size - rp) + wp;
}

static inline uint32_t ring_free_frames(ring_t *r)
{
    return (r->size - 1) - ring_used_frames(r);
}

static inline void ring_write(ring_t *r, const float *src, uint32_t frames)
{
	uint32_t ch = r->channels;
	uint32_t freef = ring_free_frames(r);

	if (freef == 0)
	{
		return;
	}

	if (frames > freef)
	{
		frames = freef;
	}

	uint32_t w = atomic_load_explicit(&r->wpos, memory_order_relaxed);
	uint32_t first = r->size - w;

	if (frames <= first)
	{
		memcpy(&r->data[w*ch], src, (size_t)frames * ch * sizeof(float));
	}
	else
	{
		memcpy(&r->data[w*ch], src, (size_t)first * ch * sizeof(float));
		memcpy(&r->data[0], &src[first*ch], (size_t)(frames-first) * ch * sizeof(float));
	}

	w = (w + frames) % r->size;
	atomic_store_explicit(&r->wpos, w, memory_order_release);
}

static inline uint32_t ring_read(ring_t *r, float *dst, uint32_t frames)
{
	uint32_t ch = r->channels;
	uint32_t used = ring_used_frames(r);
	uint32_t take = frames <= used ? frames : used;

	uint32_t rd = atomic_load_explicit(&r->rpos, memory_order_relaxed);
	uint32_t first = r->size - rd;

	if (take <= first)
	{
		memcpy(dst, &r->data[rd*ch], (size_t)take * ch * sizeof(float));
	}
	else
	{
		memcpy(dst, &r->data[rd*ch], (size_t)first * ch * sizeof(float));
		memcpy(&dst[first*ch], &r->data[0], (size_t)(take-first) * ch * sizeof(float));
	}

	rd = (rd + take) % r->size;
	atomic_store_explicit(&r->rpos, rd, memory_order_release);
	return take;
}

static int pipewire_available(void)
{
	pw_init(NULL, NULL);

	struct pw_main_loop *loop = pw_main_loop_new(NULL);
	if (!loop)
	{
		return 0;
	}

	struct pw_context *ctx = pw_context_new(pw_main_loop_get_loop(loop), NULL, 0);
	if (!ctx)
	{
		pw_main_loop_destroy(loop);
		return 0;
	}

	struct pw_core *core = pw_context_connect(ctx, NULL, 0);
	if (!core)
	{
		pw_context_destroy(ctx);
		pw_main_loop_destroy(loop);
		return 0;
	}

	pw_core_disconnect(core);
	pw_context_destroy(ctx);
	pw_main_loop_destroy(loop);
	return 1;
}

static void rebuild_coeffs_control(app_t *a)
{
	int wi = a->coeff_write_idx ^ 1;
	biquad_coeff_t (*dst)[BANDS] = a->coeff_buf[wi];

	for (uint32_t ch = 0; ch < a->channels && ch < CH; ch++)
	{
		for (int b = 0; b < BANDS; b++)
		{
			float db = atomic_load(&a->band_mdb[b]) / 1000.f;
			biquad_set_peaking_coeff(
				&dst[ch][b],
				(float)a->rate,
				a->freqs[b],
				1.0f,
				db
			);
		}
	}

	atomic_store_explicit(
		&a->coeff_ptr,
		&a->coeff_buf[wi],
		memory_order_release
	);

	a->coeff_write_idx = wi;
}

static void on_state_changed(
	const char *tag,
	enum pw_stream_state old,
	enum pw_stream_state st,
	const char *err)
{
	log_debug(
		"%s: %s -> %s%s%s\n",
		tag,
		pw_stream_state_as_string(old),
		pw_stream_state_as_string(st),
		err ? " error=" : "",
		err ? err : ""
	);
}

static void on_cap_state_changed(
	void *ud,
	enum pw_stream_state old,
	enum pw_stream_state st,
	const char *err)
{
	(void)ud;
	on_state_changed("capture ", old, st, err);
}

static void on_pb_state_changed(
	void *ud,
	enum pw_stream_state old,
	enum pw_stream_state st,
	const char *err)
{
	app_t *a = ud;
	if (st == PW_STREAM_STATE_STREAMING)
	{
		atomic_store_explicit(&a->pb_streaming, 1, memory_order_release);
	}
	else
	{
		atomic_store_explicit(&a->pb_streaming, 0, memory_order_release);
	}
	on_state_changed("playback", old, st, err);
}

static void on_cap_process(void *userdata)
{
	app_t *a = (app_t*)userdata;

	struct pw_buffer *inb = pw_stream_dequeue_buffer(a->cap);
	if (!inb)
	{
		return;
	}

	struct spa_buffer *in = inb->buffer;
	if (!in || !in->datas[0].data || !in->datas[0].chunk)
	{
		pw_stream_queue_buffer(a->cap, inb);
		return;
	}

	uint32_t stride = in->datas[0].chunk->stride;
	if (stride == 0)
	{
		stride = a->channels * sizeof(float);
	}

	uint32_t bytes = in->datas[0].chunk->size;
	if (bytes == 0)
	{
		pw_stream_queue_buffer(a->cap, inb);
		return;
	}

	uint32_t frames = bytes / stride;
	if (frames == 0)
	{
		pw_stream_queue_buffer(a->cap, inb);
		return;
	}

	uint8_t *base = (uint8_t*)in->datas[0].data;
	float *src = (float*)(base + in->datas[0].chunk->offset);

	float tmp[DSP_BLOCK_FRAMES * CH];
	float mgain = bits_to_f32(atomic_load(&a->master_gain_bits));

	uint32_t done = 0;
	while (done < frames)
	{
		uint32_t n = frames - done;
		if (n > DSP_BLOCK_FRAMES)
		{
			n = DSP_BLOCK_FRAMES;
		}

		biquad_coeff_t (*coeff)[CH][BANDS] = atomic_load_explicit(&a->coeff_ptr, memory_order_acquire);
		for (uint32_t f = 0; f < n; f++)
		{
			for (uint32_t ch = 0; ch < a->channels; ch++)
			{
				uint32_t i = (done+f)*a->channels + ch;
				float x = src[i] * mgain;

				for (int b = 0; b < BANDS; b++)
				{
					x = biquad_process(&(*coeff)[ch][b], &a->state[ch][b], x);
				}

				if (x > 0.95f)
				{
					x = 0.95f;
				}

				if (x < -0.95f)
				{
					x = -0.95f;
				}
				tmp[f*a->channels + ch] = x;
			}
		}

		ring_write(&a->ring, tmp, n);
		done += n;
	}

	struct pw_stream *pb = atomic_load_explicit(&a->pb, memory_order_acquire);
	if (pb && atomic_load_explicit(&a->pb_streaming, memory_order_acquire))
	{
		pw_stream_trigger_process(pb);
	}

	pw_stream_queue_buffer(a->cap, inb);
}

static void on_pb_process(void *userdata)
{
	app_t *a = (app_t*)userdata;
	if (!a->pb)
	{
		return;
	}

	struct pw_buffer *outb = pw_stream_dequeue_buffer(a->pb);
	if (!outb)
	{
		return;
	}

	struct spa_buffer *out = outb->buffer;
	if (!out || !out->datas[0].data || !out->datas[0].chunk)
	{
		pw_stream_queue_buffer(a->pb, outb);
		return;
	}

	uint32_t stride = out->datas[0].chunk->stride;
	if (stride == 0)
	{
		stride = a->channels * sizeof(float);
	}

	uint32_t frames;

	if (out->datas[0].chunk->size > 0)
	{
		frames = out->datas[0].chunk->size / stride;
	}
	else
	{
		frames = 128;
	}

	uint32_t max_frames = out->datas[0].maxsize / stride;
	if (frames > max_frames)
	{
		frames = max_frames;
	}

	if (frames == 0)
	{
		pw_stream_queue_buffer(a->pb, outb);
		return;
	}

	uint8_t *base = (uint8_t*)out->datas[0].data;
	float *dst = (float*)(base + out->datas[0].chunk->offset);

	uint32_t got = ring_read(&a->ring, dst, frames);
	if (got < frames)
	{
		memset(
			&dst[got * a->channels],
			0,
			(size_t)(frames - got) * stride
		);
	}

	out->datas[0].chunk->size = frames * stride;
	pw_stream_queue_buffer(a->pb, outb);
}

static const struct pw_stream_events cap_events = {
	PW_VERSION_STREAM_EVENTS,
	.process = on_cap_process,
	.state_changed = on_cap_state_changed,
};

static const struct pw_stream_events pb_events = {
	PW_VERSION_STREAM_EVENTS,
	.process = on_pb_process,
	.state_changed = on_pb_state_changed,
};

static atomic_int  reg_sync_done = 0;
static atomic_uint reg_sync_seq  = 0;

static void core_done(void *data, uint32_t id, int seq)
{
	(void)data;
	(void)id;

	if ((uint32_t)seq == atomic_load(&reg_sync_seq))
	{
		atomic_store(&reg_sync_done, 1);
	}
}

static const struct pw_core_events core_events = {
	PW_VERSION_CORE_EVENTS,
	.done = core_done,
};

static void registry_global(
	void *data,
	uint32_t id,
	uint32_t permissions,
	const char *type,
	uint32_t version,
	const struct spa_dict *props)
{
	(void)data;
	(void)permissions;
	(void)version;

	if (!props)
	{
		return;
	}

	if (strcmp(type, PW_TYPE_INTERFACE_Node) != 0)
	{
		return;
	}

	const char *media_class = spa_dict_lookup(props, PW_KEY_MEDIA_CLASS);
	if (!media_class || strcmp(media_class, "Audio/Sink") != 0)
	{
		return;
	}

	const char *serial = spa_dict_lookup(props, PW_KEY_OBJECT_SERIAL);
	if (!serial)
	{
		return;
	}

	pthread_mutex_lock(&sinks_mu);

	for (int i = 0; i < atomic_load(&sink_count); i++)
	{
		if (sinks[i].id == id)
		{
			pthread_mutex_unlock(&sinks_mu);
			return;
		}
	}

	int idx = atomic_load(&sink_count);
	if (idx < MAX_SINKS)
	{
		sinks[idx].id = id;
		sinks[idx].serial = (uint32_t)atoi(serial);
		const char *name = spa_dict_lookup(props, PW_KEY_NODE_NAME);
		const char *desc = spa_dict_lookup(props, PW_KEY_NODE_DESCRIPTION);

		snprintf(
			sinks[idx].name,
			sizeof(sinks[idx].name),
			"%s",
			name ? name : ""
		);

		snprintf(
			sinks[idx].desc,
			sizeof(sinks[idx].desc),
			"%s",
			desc ? desc : ""
		);

		atomic_store(&sink_count, idx + 1);
	}

	pthread_mutex_unlock(&sinks_mu);
}

static void registry_global_remove(void *data, uint32_t id)
{
	(void)data;

	pthread_mutex_lock(&sinks_mu);

	int n = atomic_load(&sink_count);
	for (int i = 0; i < n; i++)
	{
		if (sinks[i].id == id)
		{
			if (sinks[i].serial == app.target_serial)
			{
				app.target_serial = 0;
				atomic_store(&app.reconnect_pb, 1);
			}

			sinks[i] = sinks[n - 1];
			atomic_store(&sink_count, n - 1);
			break;
		}
	}

	pthread_mutex_unlock(&sinks_mu);
}

static const struct pw_registry_events registry_events = {
	PW_VERSION_REGISTRY_EVENTS,
	.global = registry_global,
	.global_remove = registry_global_remove,
};

static void enumerate_sinks(app_t *a)
{
	atomic_store(&sink_count, 0);

	struct pw_registry *reg = pw_core_get_registry(a->core, PW_VERSION_REGISTRY, 0);

	static struct spa_hook reg_listener;
	pw_registry_add_listener(reg, &reg_listener, &registry_events, NULL);

	static struct spa_hook core_listener;
	pw_core_add_listener(a->core, &core_listener, &core_events, NULL);

	atomic_store(&reg_sync_done, 0);
	int seq = pw_core_sync(a->core, PW_ID_CORE, 0);
	atomic_store(&reg_sync_seq, (uint32_t)seq);

	for (int i = 0; i < 100 && !atomic_load(&reg_sync_done); i++)
	{
		pw_loop_iterate(pw_main_loop_get_loop(a->loop), 10);
	}

	pw_proxy_destroy((struct pw_proxy*)reg);
}

static int sink_index_by_serial(uint32_t serial)
{
	pthread_mutex_lock(&sinks_mu);

	int n = atomic_load(&sink_count);
	for (int i = 0; i < n; i++)
	{
		if (sinks[i].serial == serial)
		{
			pthread_mutex_unlock(&sinks_mu);
			return i;
		}
	}

	pthread_mutex_unlock(&sinks_mu);
	return -1;
}

static int sink_index_by_identity(const char *name, const char *desc)
{
	pthread_mutex_lock(&sinks_mu);

	int n = atomic_load(&sink_count);
	for (int i = 0; i < n; i++)
	{
		if (strcmp(sinks[i].name, name) == 0 && strcmp(sinks[i].desc, desc) == 0)
		{
			pthread_mutex_unlock(&sinks_mu);
			return i;
		}
	}

	pthread_mutex_unlock(&sinks_mu);
	return -1;
}

void audio_engine_band(int i, float db)
{
	if (i < 0 || i >= BANDS)
	{
		return;
	}

	atomic_store(&app.band_mdb[i], (int)lrintf(db * 1000.f));
	rebuild_coeffs_control(&app);
}

void audio_engine_master(float db)
{
	float g = powf(10.f, db/20.f);
	atomic_store(&app.master_gain_bits, f32_to_bits(g));
}

int audio_engine_set_target_sink(uint32_t serial)
{
	if (sink_index_by_serial(serial) < 0)
	{
		return -1;
	}

	app.target_serial = serial;
	atomic_store(&app.reconnect_pb, 1);
	return 0;
}

const char* audio_engine_self_sink_name(void)
{
	return DEVICE_SINK_DESC.name;
}

int audio_engine_sink_count(void)
{
	pthread_mutex_lock(&sinks_mu);
	int n = atomic_load(&sink_count);
	pthread_mutex_unlock(&sinks_mu);

	return n;
}

const char* audio_engine_sink_desc(int i)
{
	const char *ret = NULL;

	pthread_mutex_lock(&sinks_mu);
	if (i >= 0 && i < atomic_load(&sink_count))
	{
		ret = sinks[i].desc;
	}
	pthread_mutex_unlock(&sinks_mu);

	return ret;
}

const char* audio_engine_sink_name(int i)
{
	const char *ret = NULL;

	pthread_mutex_lock(&sinks_mu);
	if (i >= 0 && i < atomic_load(&sink_count))
	{
		ret = sinks[i].name;
	}
	pthread_mutex_unlock(&sinks_mu);

	return ret;
}

uint32_t audio_engine_sink_serial(int i)
{
	uint32_t ret = 0;

	pthread_mutex_lock(&sinks_mu);
	if (i >= 0 && i < atomic_load(&sink_count))
	{
		ret = sinks[i].serial;
	}
	pthread_mutex_unlock(&sinks_mu);

	return ret;
}

uint32_t audio_engine_current_sink_serial(void)
{
	return app.target_serial;
}

void audio_engine_stop(void)
{
	atomic_store(&app.quit, 1);
}

const char* audio_engine_current_sink_name(void)
{
	const char *ret = NULL;

	pthread_mutex_lock(&sinks_mu);
	for (int i = 0; i < atomic_load(&sink_count); i++)
	{
		if (sinks[i].serial == app.target_serial)
		{
			ret = sinks[i].name;
			break;
		}
	}
	pthread_mutex_unlock(&sinks_mu);

	return ret;
}

const char* audio_engine_current_sink_desc(void)
{
	const char *ret = NULL;

	pthread_mutex_lock(&sinks_mu);
	for (int i = 0; i < atomic_load(&sink_count); i++)
	{
		if (sinks[i].serial == app.target_serial)
		{
			ret = sinks[i].desc;
			break;
		}
	}
	pthread_mutex_unlock(&sinks_mu);

	return ret;
}

int audio_engine_config(
	uint32_t rate,
	uint32_t channels,
	uint32_t polling_rate,
	uint32_t debug,
    uint32_t ring_frames,
	const char *lat,
	const char *max_lat,
	const char *preferred_sink_name,
	const char *preferred_sink_desc)
{
	if (atomic_load(&pw_started))
	{
		return -1;
	}

	if (rate < 8000 || rate > 192000)
	{
		return -2;
	}

	if (channels < 1 || channels > 8)
	{
		return -3;
	}

	if (polling_rate < 1 || polling_rate > 50)
	{
		return -4;
	}

	if (!lat || !max_lat)
	{
		return -5;
	}

	pw_cfg.rate = rate;
	pw_cfg.channels = channels;
	pw_cfg.polling_rate = polling_rate;
	pw_cfg.debug = debug;
	pw_cfg.ring_frames = ring_frames;

	if (pw_cfg.ring_frames < 128)
	{
		pw_cfg.ring_frames = 128;
	}

	if (pw_cfg.ring_frames > 8192)
	{
		pw_cfg.ring_frames = 8192;
	}

	snprintf(
		pw_cfg.latency,
		sizeof(pw_cfg.latency),
		"%s",
		lat
    );

	snprintf(
		pw_cfg.max_latency,
		sizeof(pw_cfg.max_latency),
		"%s",
		max_lat
    );

	if (preferred_sink_name && preferred_sink_name[0] != '\0')
	{
		snprintf(
			pw_cfg.preferred_sink_name,
			sizeof(pw_cfg.preferred_sink_name),
			"%s",
			preferred_sink_name
		);
	}
    else
    {
		pw_cfg.preferred_sink_name[0] = '\0';
	}

	if (preferred_sink_desc && preferred_sink_desc[0] != '\0')
	{
		snprintf(
			pw_cfg.preferred_sink_desc,
			sizeof(pw_cfg.preferred_sink_desc),
			"%s",
			preferred_sink_desc
		);
	}
    else
    {
		pw_cfg.preferred_sink_desc[0] = '\0';
	}

	return 0;
}

static int connect_capture(app_t *a)
{
	struct spa_audio_info_raw info = {0};
	info.format = SPA_AUDIO_FORMAT_F32;
	info.rate = a->rate;
	info.channels = a->channels;
	info.position[0] = SPA_AUDIO_CHANNEL_FL;
	info.position[1] = SPA_AUDIO_CHANNEL_FR;

	uint8_t podbuf[256];
	struct spa_pod_builder b = SPA_POD_BUILDER_INIT(podbuf, sizeof(podbuf));
	const struct spa_pod *param = spa_format_audio_raw_build(&b, SPA_PARAM_EnumFormat, &info);
	const struct spa_pod *params[1] = { param };

	const char *lat = pw_cfg.latency;
	const char *max_lat = pw_cfg.max_latency;

	log_debug("connect_capture_latency: %s\n", pw_cfg.latency);
	log_debug("connect_capture_latency_max: %s\n", pw_cfg.max_latency);

    char rate_str[16];
    snprintf(
		rate_str,
		sizeof(rate_str),
		"%u",
		a->rate
    );

	a->cap = pw_stream_new(
		a->core,
		DEVICE_SINK_DESC.name,
		pw_properties_new(
			PW_KEY_MEDIA_CLASS, DEVICE_SINK_DESC.media_class,
			PW_KEY_NODE_NAME, DEVICE_SINK_DESC.name,
			PW_KEY_NODE_DESCRIPTION, DEVICE_SINK_DESC.desc,
			PW_KEY_MEDIA_ROLE, DEVICE_SINK_DESC.role,
			PW_KEY_NODE_LATENCY, lat,
			PW_KEY_NODE_MAX_LATENCY, max_lat,
			"node.lock-quantum", "true",
			"node.rate", rate_str,
			"node.group", DEVICE_SINK_DESC.node_group,
			"link-group", DEVICE_SINK_DESC.link_group,
			NULL
		)
	);

	if (!a->cap)
	{
		return -1;
	}

	pw_stream_add_listener(a->cap, &a->cap_listener, &cap_events, a);

	int rc = pw_stream_connect(
		a->cap,
		PW_DIRECTION_INPUT,
		PW_ID_ANY,
		PW_STREAM_FLAG_MAP_BUFFERS |
		PW_STREAM_FLAG_RT_PROCESS,
		params,
		1
	);

	if (rc < 0)
	{
		log_debug("pw_stream_connect(capture) failed: %d\n", rc);
		return rc;
	}
	return 0;
}

static int connect_playback(app_t *a)
{
	if (a->target_serial == 0)
	{
		log_debug("No target sink selected (serial=0)\n");
		return -1;
	}

	struct spa_audio_info_raw info = {0};
	info.format = SPA_AUDIO_FORMAT_F32;
	info.rate = a->rate;
	info.channels = a->channels;
	info.position[0] = SPA_AUDIO_CHANNEL_FL;
	info.position[1] = SPA_AUDIO_CHANNEL_FR;

	uint8_t podbuf[256];
	struct spa_pod_builder b = SPA_POD_BUILDER_INIT(podbuf, sizeof(podbuf));
	const struct spa_pod *param = spa_format_audio_raw_build(&b, SPA_PARAM_EnumFormat, &info);
	const struct spa_pod *params[1] = { param };

	const char *lat     = pw_cfg.latency;
	const char *max_lat = pw_cfg.max_latency;

	log_debug("connect_playback_latency: %s\n", pw_cfg.latency);
	log_debug("connect_playback_latency_max: %s\n", pw_cfg.max_latency);

	char target_str[32];
	snprintf(
		target_str,
		sizeof(target_str),
		"%u",
		a->target_serial
    );

    char rate_str[16];
    snprintf(
		rate_str,
		sizeof(rate_str),
		"%u",
		a->rate
    );

	a->pb = pw_stream_new(
		a->core,
		DEVICE_PLAYBACK_DESC.name,
		pw_properties_new(
			PW_KEY_MEDIA_TYPE,          DEVICE_PLAYBACK_DESC.type,
			PW_KEY_MEDIA_CATEGORY,      DEVICE_PLAYBACK_DESC.category,
			PW_KEY_MEDIA_ROLE,          DEVICE_PLAYBACK_DESC.role,
			PW_KEY_NODE_NAME,           DEVICE_PLAYBACK_DESC.name,
			PW_KEY_TARGET_OBJECT,       target_str,
			PW_KEY_NODE_LATENCY,        lat,
			PW_KEY_NODE_MAX_LATENCY,    max_lat,
			"node.lock-quantum",        "true",
			"node.rate",                rate_str,
			"node.group",               DEVICE_PLAYBACK_DESC.node_group,
			"link-group",               DEVICE_PLAYBACK_DESC.link_group,
			NULL
		)
	);

	if (!a->pb)
	{
		return -1;
	}

	pw_stream_add_listener(a->pb, &a->pb_listener, &pb_events, a);

	int rc = pw_stream_connect(
		a->pb,
		PW_DIRECTION_OUTPUT,
		PW_ID_ANY,
		PW_STREAM_FLAG_AUTOCONNECT |
		PW_STREAM_FLAG_DONT_RECONNECT |
		PW_STREAM_FLAG_TRIGGER |
		PW_STREAM_FLAG_MAP_BUFFERS |
		PW_STREAM_FLAG_RT_PROCESS,
		params,
		1
	);

	if (rc < 0)
	{
		log_debug("pw_stream_connect(playback) failed: %d\n", rc);
		return rc;
	}

	log_debug("playback connected to target serial=%u\n", a->target_serial);
	return 0;
}

static void disconnect_playback(app_t *a)
{
	struct pw_stream *pb = atomic_exchange_explicit(&a->pb, NULL, memory_order_acq_rel);
	if (pb)
	{
		pw_stream_disconnect(pb);
		pw_stream_destroy(pb);
	}
}

int audio_engine_running(void)
{
	return atomic_load(&pw_started);
}

int audio_engine_start()
{
	if (!pipewire_available())
	{
		set_error("PipeWire is not available");
		return 1;
	}

	memset(&app, 0, sizeof(app));
	app.rate = pw_cfg.rate;
	app.channels = pw_cfg.channels;
    app.ring.size = pw_cfg.ring_frames;
    app.ring.channels = app.channels;
	app.coeff_write_idx = 0;
	atomic_store(&pw_started, 1);
	atomic_store(&app.coeff_ptr, &app.coeff_buf[0]);
	atomic_store(&app.pb_streaming, 0);

    if (app.channels != CH)
    {
        set_error("channels must match CH");
        return 1;
    }

    app.ring.data = calloc(
        app.ring.size * app.ring.channels,
        sizeof(float)
    );

    if (!app.ring.data)
    {
        set_error("failed to allocate ring buffer");
        return 1;
    }

	float f[BANDS] = {32,64,125,250,500,1000,2000,4000,8000,16000};
	memcpy(app.freqs, f, sizeof(f));

	for (int i = 0; i < BANDS; i++)
	{
		atomic_store(&app.band_mdb[i], 0);
	}

	atomic_store(&app.master_gain_bits, f32_to_bits(1.0f));
	atomic_store(&app.ring.wpos, 0);
	atomic_store(&app.ring.rpos, 0);
	atomic_store(&app.quit, 0);
	rebuild_coeffs_control(&app);

	for (int ch = 0; ch < CH; ch++)
	{
		for (int b = 0; b < BANDS; b++)
		{
			biquad_state_reset(&app.state[ch][b]);
		}
	}

	pw_init(NULL, NULL);

	app.loop = pw_main_loop_new(NULL);
	app.context = pw_context_new(pw_main_loop_get_loop(app.loop), NULL, 0);
	app.core = pw_context_connect(app.context, NULL, 0);

	if (!app.core)
	{
		set_error("Failed to connect to PipeWire");
		return 1;
	}

	struct pw_registry *reg = pw_core_get_registry(app.core, PW_VERSION_REGISTRY, 0);
	static struct spa_hook reg_listener;
	pw_registry_add_listener(reg, &reg_listener, &registry_events, NULL);

	enumerate_sinks(&app);
	if (pw_cfg.preferred_sink_name[0])
	{
		int si = sink_index_by_identity(
			pw_cfg.preferred_sink_name,
			pw_cfg.preferred_sink_desc
		);

		if (si >= 0)
		{
			log_debug(
				"Physical sink: %s (%s)\n",
				pw_cfg.preferred_sink_desc,
				pw_cfg.preferred_sink_name
			);
			app.target_serial = sinks[si].serial;
		}
	}

	if (app.target_serial == 0 && atomic_load(&sink_count) > 0)
	{
		app.target_serial = sinks[0].serial;
	}

	int rc = connect_capture(&app);
	if (rc < 0)
	{
		return 1;
	}

	if (app.target_serial != 0)
	{
		connect_playback(&app);
	}

	log_debug("Running.\n");

	log_debug(
		"Virtual sink: %s (%s)\n",
		DEVICE_SINK_DESC.name,
		DEVICE_SINK_DESC.desc
	);

    log_debug(
		"Playback node: %s (role=%s, category=%s)\n",
		DEVICE_PLAYBACK_DESC.name,
		DEVICE_PLAYBACK_DESC.role,
		DEVICE_PLAYBACK_DESC.category
	);

	log_debug(
		"Ring buffer: %d frames (%.2fs)\n",
		app.ring.size,
		(double)app.ring.size / (double)app.rate
	);

	atomic_store(&pw_running, 1);
	for (;;)
	{
		pw_loop_iterate(pw_main_loop_get_loop(app.loop), pw_cfg.polling_rate);

		if (atomic_exchange(&app.reconnect_pb, 0))
		{
			log_debug("switching sink to serial=%u\n", app.target_serial);
			disconnect_playback(&app);
			connect_playback(&app);
		}

		if (atomic_load(&app.quit))
		{
			log_debug("stopping\n");
			break;
		}
	}

	if (app.cap)
	{
		pw_stream_disconnect(app.cap);
		pw_stream_destroy(app.cap);
		app.cap = NULL;
	}

	if (app.pb)
	{
		pw_stream_disconnect(app.pb);
		pw_stream_destroy(app.pb);
		app.pb = NULL;
	}

	if (app.core)
	{
		pw_core_disconnect(app.core);
	}

	if (app.context)
	{
		pw_context_destroy(app.context);
	}

	if (app.loop)
	{
		pw_main_loop_destroy(app.loop);
	}

	if (app.ring.data)
	{
		free(app.ring.data);
		app.ring.data = NULL;
	}

	pw_deinit();
	atomic_store(&pw_started, 0);
	atomic_store(&pw_running, 0);

	return 0;
}