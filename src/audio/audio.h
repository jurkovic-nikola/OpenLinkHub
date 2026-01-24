#pragma once
#include <stdint.h>

/*
 * Public API for the PipeWire audio engine.
 *
 * This header defines the control-plane interface used by the Go backend
 * via CGO. All functions declared here are safe to call from non–real-time
 * threads and do not perform audio processing directly.
 *
 * The implementation resides in audio.c.
 */

/* Returns the last error message set by the audio engine, or NULL if none. */
const char* audio_engine_last_error(void);

/*
 * Configures the audio engine.
 *
 * Must be called before audio_engine_start().
 * Returns 0 on success, negative value on error.
 */
int audio_engine_config(
    uint32_t rate,
    uint32_t channels,
    uint32_t polling_rate,
    uint32_t debug,
    uint32_t ring_frames,
    const char *lat,
    const char *max_lat,
    const char *preferred_sink_name,
    const char *preferred_sink_desc
);

/*
 * Starts the audio engine.
 *
 * Initializes PipeWire, creates the virtual sink, and enters the
 * processing loop. This call blocks until audio_engine_stop() is invoked.
 *
 * Returns 0 on normal shutdown, non-zero on error.
 */
int audio_engine_start(void);

/*
 * Requests the audio engine to stop.
 *
 * This function is asynchronous and signals the processing loop to exit.
 */
void audio_engine_stop(void);

/* Returns the status of audio engine. */
int audio_engine_running(void);

/* Sets gain (in dB) for a specific EQ band (0..9). */
void audio_engine_band(int i, float db);

/* Sets master gain (in dB). */
void audio_engine_master(float db);

/* Returns the PipeWire node name of the engine's virtual sink. */
const char* audio_engine_self_sink_name(void);

/* Returns the number of available PipeWire audio sinks. */
int audio_engine_sink_count(void);

/* Returns the serial number of the sink at index i. */
uint32_t audio_engine_sink_serial(int i);

/* Returns the internal name of the sink at index i. */
const char* audio_engine_sink_name(int i);

/* Returns the human-readable description of the sink at index i. */
const char* audio_engine_sink_desc(int i);

/*
 * Selects the target output sink by serial number.
 *
 * Returns 0 on success, -1 if the sink does not exist.
 */
int audio_engine_set_target_sink(uint32_t serial);

/* Returns the serial number of the currently selected sink, or 0 if none. */
uint32_t audio_engine_current_sink_serial(void);

/* Returns the name of the currently selected sink, or NULL if none. */
const char* audio_engine_current_sink_name(void);

/* Returns the description of the currently selected sink, or NULL if none. */
const char* audio_engine_current_sink_desc(void);