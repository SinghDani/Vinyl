import { useEffect, useRef, useState } from "react";
import type { MatchingSong } from "./types";

type RecordingState = "idle" | "recording" | "error";

type WorkletPcmMessage = {
  type: "pcm16";
  samples: ArrayBuffer;
};

type WorkletFlushCompleteMessage = {
  type: "flush_complete";
};

type WorkletMessage = WorkletPcmMessage | WorkletFlushCompleteMessage;

type BackendMatchMessage = {
  songName: string;
  artist: string;
  verdict: string;
};

type AudioRuntime = {
  audioContext: AudioContext;
  sourceNode: MediaStreamAudioSourceNode;
  lowpassA: BiquadFilterNode;
  lowpassB: BiquadFilterNode;
  workletNode: AudioWorkletNode;
  muteGain: GainNode;
  socket: WebSocket;
  sourceSampleRate: number;
  flushResolver: (() => void) | null;
};

const TARGET_SAMPLE_RATE = 11_025;
const LOWPASS_CUTOFF_HZ = 5_000;
const FRAME_DURATION_MS = 200;
const FRAME_SAMPLES = Math.round(
  (TARGET_SAMPLE_RATE * FRAME_DURATION_MS) / 1_000,
);

function getWebSocketUrl(): string {
  const fromEnv = import.meta.env.VITE_WS_URL;
  if (typeof fromEnv !== "string" || fromEnv.trim().length === 0) {
    throw new Error("Missing VITE_WS_URL in frontend .env");
  }
  return fromEnv;
}

function getAudioContextConstructor(): typeof AudioContext {
  const maybeWindow = window as Window & {
    webkitAudioContext?: typeof AudioContext;
  };

  if (typeof window.AudioContext !== "undefined") {
    return window.AudioContext;
  }
  if (typeof maybeWindow.webkitAudioContext !== "undefined") {
    return maybeWindow.webkitAudioContext;
  }
  throw new Error("AudioContext is not supported in this browser.");
}

function connectWebSocket(url: string): Promise<WebSocket> {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(url);
    ws.binaryType = "arraybuffer";

    const cleanup = () => {
      ws.removeEventListener("open", handleOpen);
      ws.removeEventListener("error", handleError);
    };

    const handleOpen = () => {
      cleanup();
      resolve(ws);
    };

    const handleError = () => {
      cleanup();
      ws.close();
      reject(new Error(`WebSocket connection failed: ${url}`));
    };

    ws.addEventListener("open", handleOpen);
    ws.addEventListener("error", handleError);
  });
}

function parseMatchMessage(message: string): MatchingSong {
  const payload = JSON.parse(message) as BackendMatchMessage;
  return {
    name: payload.songName,
    artist: payload.artist,
    confidence: payload.verdict,
  };
}

type Props = {
  setMatch: React.Dispatch<React.SetStateAction<MatchingSong>>;
};

export default function RecordButton(props: Props) {
  const [state, setState] = useState<RecordingState>("idle");
  const { setMatch } = props;

  const streamRef = useRef<MediaStream | null>(null);
  const runtimeRef = useRef<AudioRuntime | null>(null);
  const processedSampleCountRef = useRef(0);
  const stopRequestedRef = useRef(false);

  function stopTracks() {
    const stream = streamRef.current;
    if (!stream) return;
    stream.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
  }

  async function flushWorklet(runtime: AudioRuntime): Promise<void> {
    return await new Promise<void>((resolve) => {
      let finished = false;

      const finish = () => {
        if (finished) return;
        finished = true;
        runtime.flushResolver = null;
        resolve();
      };

      runtime.flushResolver = finish;

      runtime.workletNode.port.postMessage({ type: "flush" });

      window.setTimeout(finish, 200);
    });
  }

  async function stopAudioRuntime(): Promise<void> {
    const runtime = runtimeRef.current;
    if (!runtime) return;

    // Mark the session as gone before closing resources, so the next click can
    // start a fresh recording even after the backend found a match.
    runtimeRef.current = null;
    runtime.flushResolver = null;

    runtime.workletNode.port.onmessage = null;
    runtime.socket.onmessage = null;
    runtime.socket.onerror = null;
    runtime.socket.onclose = null;

    runtime.sourceNode.disconnect();
    runtime.lowpassA.disconnect();
    runtime.lowpassB.disconnect();
    runtime.workletNode.disconnect();
    runtime.muteGain.disconnect();

    if (runtime.socket.readyState === WebSocket.OPEN) {
      runtime.socket.close();
    } else if (runtime.socket.readyState === WebSocket.CONNECTING) {
      runtime.socket.close();
    }

    if (runtime.audioContext.state !== "closed") {
      await runtime.audioContext.close();
    }
  }

  async function stopRecording() {
    if (stopRequestedRef.current) {
      return;
    }
    stopRequestedRef.current = true;

    try {
      const runtime = runtimeRef.current;
      const sourceSampleRate = runtime?.sourceSampleRate;

      if (runtime) {
        await flushWorklet(runtime);
      }

      await stopAudioRuntime();
      stopTracks();

      const sampleCount = processedSampleCountRef.current;
      const durationSeconds = sampleCount / TARGET_SAMPLE_RATE;

      console.log("processed-audio", {
        sourceSampleRate,
        targetSampleRate: TARGET_SAMPLE_RATE,
        frameSamples: FRAME_SAMPLES,
        sampleCount,
        byteLength: sampleCount * 2,
        durationSeconds: Number(durationSeconds.toFixed(3)),
      });

      setState("idle");
      stopRequestedRef.current = false;
    } catch (error) {
      console.error(error);
      setState("error");
      stopRequestedRef.current = false;
    }
  }

  async function startRecording() {
    if (state === "recording" || runtimeRef.current) {
      return;
    }
    stopRequestedRef.current = false;
    setMatch({ name: "", artist: "", confidence: "" });
    setState("recording");

    let localAudioContext: AudioContext | null = null;
    let localSocket: WebSocket | null = null;

    try {
      if (!navigator.mediaDevices?.getUserMedia) {
        throw new Error("getUserMedia is not supported in this browser.");
      }

      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          channelCount: 1,
          echoCancellation: false,
          noiseSuppression: false,
          autoGainControl: false,
        },
      });
      streamRef.current = stream;

      localSocket = await connectWebSocket(getWebSocketUrl());

      const AudioContextCtor = getAudioContextConstructor();
      const audioContext = new AudioContextCtor();
      localAudioContext = audioContext;
      await audioContext.resume();

      await audioContext.audioWorklet.addModule("/audio-resample-worklet.js");

      const sourceNode = audioContext.createMediaStreamSource(stream);

      const lowpassA = audioContext.createBiquadFilter();
      lowpassA.type = "lowpass";
      lowpassA.frequency.value = LOWPASS_CUTOFF_HZ;
      lowpassA.Q.value = 0.707;

      const lowpassB = audioContext.createBiquadFilter();
      lowpassB.type = "lowpass";
      lowpassB.frequency.value = LOWPASS_CUTOFF_HZ;
      lowpassB.Q.value = 0.707;

      const workletNode = new AudioWorkletNode(
        audioContext,
        "downsample-pcm16-processor",
        {
          numberOfInputs: 1,
          numberOfOutputs: 1,
          outputChannelCount: [1],
          processorOptions: {
            targetSampleRate: TARGET_SAMPLE_RATE,
            frameSamples: FRAME_SAMPLES,
          },
        },
      );

      const muteGain = audioContext.createGain();
      muteGain.gain.value = 0;

      const runtime: AudioRuntime = {
        audioContext,
        sourceNode,
        lowpassA,
        lowpassB,
        workletNode,
        muteGain,
        socket: localSocket,
        sourceSampleRate: audioContext.sampleRate,
        flushResolver: null,
      };

      runtimeRef.current = runtime;
      processedSampleCountRef.current = 0;

      workletNode.port.onmessage = (event: MessageEvent<WorkletMessage>) => {
        const data = event.data;

        if (data.type === "pcm16") {
          processedSampleCountRef.current += data.samples.byteLength / 2;

          if (runtime.socket.readyState === WebSocket.OPEN) {
            runtime.socket.send(data.samples);
          }
          return;
        }

        if (data.type === "flush_complete") {
          runtime.flushResolver?.();
        }
      };


      runtime.socket.onmessage = (event: MessageEvent<string>) => {
        try {
          const matchResult = parseMatchMessage(event.data);
          setMatch(matchResult);
          console.log("winner-detected", { song: matchResult.name });
          void stopRecording();
        } catch (error) {
          console.error("Invalid websocket match message", error);
          setState("error");
          void stopRecording();
        }
      };

      runtime.socket.onerror = (event) => {
        console.error("WebSocket stream error", event);
        if (!stopRequestedRef.current) {
          void stopRecording();
        }
      };

      runtime.socket.onclose = () => {
        if (!stopRequestedRef.current && runtimeRef.current === runtime) {
          console.error("WebSocket closed during recording.");
          void stopRecording();
        }
      };

      sourceNode.connect(lowpassA);
      lowpassA.connect(lowpassB);
      lowpassB.connect(workletNode);
      workletNode.connect(muteGain);
      muteGain.connect(audioContext.destination);

      console.log("dsp-started", {
        wsUrl: getWebSocketUrl(),
        sourceSampleRate: audioContext.sampleRate,
        targetSampleRate: TARGET_SAMPLE_RATE,
        lowpassHz: LOWPASS_CUTOFF_HZ,
        frameSamples: FRAME_SAMPLES,
      });

      setState("recording");
    } catch (error) {
      console.error(error);
      if (localSocket && localSocket.readyState < WebSocket.CLOSING) {
        localSocket.close();
      }
      if (localAudioContext && localAudioContext.state !== "closed") {
        await localAudioContext.close();
      }
      await stopAudioRuntime();
      stopTracks();
      setState("error");
      stopRequestedRef.current = false;
    }
  }

  useEffect(() => {
    return () => {
      void stopAudioRuntime();
      stopTracks();
    };
  }, []);

  function handleClick() {
    if (state === "recording") {
      void stopRecording();
      return;
    }
    void startRecording();
  }

  const isRecordingUi = state === "recording";
  const buttonLabel = isRecordingUi ? "STOP" : "RECORD";
  const statusText = isRecordingUi
    ? "Listening..."
    : state === "error"
      ? "Error"
      : "";

  return (
    <>
      <button
        className={`record-btn ${isRecordingUi ? "recording" : ""}`}
        onClick={handleClick}
      >
        <span className="bracket">[</span>
        <span className="dot" />
        <span>{buttonLabel}</span>
        <span className="bracket">]</span>
      </button>
      <p className={`record-status ${isRecordingUi ? "active" : ""}`}>
        {statusText}
      </p>
    </>
  );
}
