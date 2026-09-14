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

type WorkerMatch = {
  songName: string;
  artist: string;
  match: boolean;
  verdict: string;
};

type WorkerMessage =
  | { type: "result"; result: WorkerMatch }
  | { type: "error"; message: string };

type AudioRuntime = {
  audioContext: AudioContext;
  sourceNode: MediaStreamAudioSourceNode;
  lowpassA: BiquadFilterNode;
  lowpassB: BiquadFilterNode;
  workletNode: AudioWorkletNode;
  muteGain: GainNode;
  worker: Worker;
  sourceSampleRate: number;
  flushResolver: (() => void) | null;
};

const TARGET_SAMPLE_RATE = 11025;
const LOWPASS_CUTOFF_HZ = 5000;
const FRAME_DURATION_MS = 200;
const FRAME_SAMPLES = Math.round(
  (TARGET_SAMPLE_RATE * FRAME_DURATION_MS) / 1000,
);

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

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

type Props = {
  setMatch: React.Dispatch<React.SetStateAction<MatchingSong>>;
};

export default function RecordButton(props: Props) {
  const [state, setState] = useState<RecordingState>("idle");
  const [errorMessage, setErrorMessage] = useState("");
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
    // start a fresh recording even after the worker found a match.
    runtimeRef.current = null;
    runtime.flushResolver = null;

    runtime.workletNode.port.onmessage = null;
    runtime.worker.onmessage = null;
    runtime.worker.onerror = null;
    runtime.worker.onmessageerror = null;
    runtime.worker.terminate();

    runtime.sourceNode.disconnect();
    runtime.lowpassA.disconnect();
    runtime.lowpassB.disconnect();
    runtime.workletNode.disconnect();
    runtime.muteGain.disconnect();

    if (runtime.audioContext.state !== "closed") {
      await runtime.audioContext.close();
    }
  }

  async function stopRecording(finalState: RecordingState = "idle") {
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

      setState(finalState);
      stopRequestedRef.current = false;
    } catch (error) {
      console.error(error);
      setErrorMessage(getErrorMessage(error));
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
    setErrorMessage("");
    setState("recording");

    let localAudioContext: AudioContext | null = null;
    let localWorker: Worker | null = null;

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

      const worker = new Worker(
        new URL("./fingerprint-worker.ts", import.meta.url),
      );
      localWorker = worker;

      const runtime: AudioRuntime = {
        audioContext,
        sourceNode,
        lowpassA,
        lowpassB,
        workletNode,
        muteGain,
        worker,
        sourceSampleRate: audioContext.sampleRate,
        flushResolver: null,
      };

      runtimeRef.current = runtime;
      processedSampleCountRef.current = 0;

      const handleWorkerError = (message: string) => {
        console.error(message);
        setErrorMessage(message);
        void stopRecording("error");
      };

      worker.onmessage = (event: MessageEvent<WorkerMessage>) => {
        const message = event.data;
        if (message.type === "error") {
          handleWorkerError(message.message);
          return;
        }

        const result = message.result;
        setMatch({
          name: result.songName,
          artist: result.artist,
          confidence: result.verdict,
        });
        console.log("winner-detected", { song: result.songName });
        void stopRecording();
      };

      worker.onerror = (event) => {
        event.preventDefault();
        handleWorkerError(
          event.message || "The fingerprint worker stopped unexpectedly.",
        );
      };

      worker.onmessageerror = () => {
        handleWorkerError("The fingerprint worker returned an invalid message.");
      };

      workletNode.port.onmessage = (event: MessageEvent<WorkletMessage>) => {
        const data = event.data;

        if (data.type === "pcm16") {
          processedSampleCountRef.current += data.samples.byteLength / 2;

          try {
            worker.postMessage(data.samples, [data.samples]);
          } catch (error) {
            handleWorkerError(
              `Could not send audio to the fingerprint worker: ${getErrorMessage(error)}`,
            );
          }
          return;
        }

        if (data.type === "flush_complete") {
          runtime.flushResolver?.();
        }
      };
      sourceNode.connect(lowpassA);
      lowpassA.connect(lowpassB);
      lowpassB.connect(workletNode);
      workletNode.connect(muteGain);
      muteGain.connect(audioContext.destination);

      console.log("dsp-started", {
        sourceSampleRate: audioContext.sampleRate,
        targetSampleRate: TARGET_SAMPLE_RATE,
        lowpassHz: LOWPASS_CUTOFF_HZ,
        frameSamples: FRAME_SAMPLES,
      });

      setState("recording");
    } catch (error) {
      console.error(error);
      localWorker?.terminate();
      if (localAudioContext && localAudioContext.state !== "closed") {
        await localAudioContext.close();
      }
      await stopAudioRuntime();
      stopTracks();
      setErrorMessage(getErrorMessage(error));
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
      ? errorMessage || "Error"
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
