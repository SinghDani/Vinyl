import { useEffect, useRef, useState } from "react";
import type { MatchingSong, WorkerMessage } from "./types";
import { FRAME_DURATION_MS, LOWPASS_CUTOFF_HZ } from "./recording-config";

type RecordingState = "idle" | "recording" | "error";

type WorkletPcmMessage = {
  type: "pcm16";
  samples: ArrayBuffer;
};

type WorkletFlushCompleteMessage = {
  type: "flush_complete";
};

type WorkletMessage = WorkletPcmMessage | WorkletFlushCompleteMessage;

type AudioRuntime = {
  audioContext: AudioContext;
  sourceNode: MediaStreamAudioSourceNode;
  lowpassA: BiquadFilterNode;
  lowpassB: BiquadFilterNode;
  workletNode: AudioWorkletNode;
  muteGain: GainNode;
  worker: Worker;
  sourceSampleRate: number;
  targetSampleRate: number;
  frameSamples: number;
  flushResolver: (() => void) | null;
};

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
  const sessionRef = useRef<object | null>(null);
  const cancelWorkerSetupRef = useRef<(() => void) | null>(null);

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
    sessionRef.current = null;
    cancelWorkerSetupRef.current?.();

    try {
      const runtime = runtimeRef.current;
      const sourceSampleRate = runtime?.sourceSampleRate;

      if (runtime) {
        await flushWorklet(runtime);
      }

      await stopAudioRuntime();
      stopTracks();

      const sampleCount = processedSampleCountRef.current;
      const durationSeconds = runtime ? sampleCount / runtime.targetSampleRate : 0;

      console.log("processed-audio", {
        sourceSampleRate,
        targetSampleRate: runtime?.targetSampleRate,
        frameSamples: runtime?.frameSamples,
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
    if (state === "recording" || runtimeRef.current || sessionRef.current || stopRequestedRef.current) {
      return;
    }
    stopRequestedRef.current = false;
    const session = {};
    sessionRef.current = session;
    setMatch({ name: "", artist: "", confidence: "" });
    setErrorMessage("");
    setState("recording");

    let localAudioContext: AudioContext | null = null;
    let localWorker: Worker | null = null;
    let localStream: MediaStream | null = null;

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
      localStream = stream;
      if (sessionRef.current !== session) return;
      streamRef.current = stream;

      const AudioContextCtor = getAudioContextConstructor();
      const audioContext = new AudioContextCtor();
      localAudioContext = audioContext;
      await audioContext.resume();
      if (sessionRef.current !== session) return;

      await audioContext.audioWorklet.addModule("/audio-resample-worklet.js");
      if (sessionRef.current !== session) return;

      const worker = new Worker(
        new URL("./fingerprint-worker.ts", import.meta.url),
        { type: "module" },
      );
      localWorker = worker;
      const sampleRate = await new Promise<number | null>((resolve, reject) => {
        const finish = () => {
          cancelWorkerSetupRef.current = null;
          worker.onmessage = null;
          worker.onerror = null;
          worker.onmessageerror = null;
        };
        cancelWorkerSetupRef.current = () => {
          finish();
          worker.terminate();
          resolve(null);
        };
        worker.onmessage = (event: MessageEvent<WorkerMessage>) => {
          const message = event.data;
          if (message.type === "ready") {
            finish();
            resolve(message.sampleRate);
          } else if (message.type === "error") {
            finish();
            reject(new Error(message.message));
          }
        };
        worker.onerror = (event) => {
          event.preventDefault();
          finish();
          reject(new Error(event.message || "Could not start the fingerprint worker."));
        };
        worker.onmessageerror = () => {
          finish();
          reject(new Error("Could not read the fingerprint worker configuration."));
        };
      });
      if (sessionRef.current !== session || sampleRate === null) return;
      const frameSamples = Math.round(sampleRate * FRAME_DURATION_MS / 1000);

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
            targetSampleRate: sampleRate,
            frameSamples,
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
        worker,
        sourceSampleRate: audioContext.sampleRate,
        targetSampleRate: sampleRate,
        frameSamples,
        flushResolver: null,
      };

      runtimeRef.current = runtime;
      processedSampleCountRef.current = 0;

      const handleWorkerError = (message: string) => {
        if (sessionRef.current !== session) return;
        console.error(message);
        setErrorMessage(message);
        void stopRecording("error");
      };

      worker.onmessage = (event: MessageEvent<WorkerMessage>) => {
        if (sessionRef.current !== session) return;
        const message = event.data;
        if (message.type === "ready") return;
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
        console.log("recognition-result", { song: result.songName, match: result.match });
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
        targetSampleRate: sampleRate,
        lowpassHz: LOWPASS_CUTOFF_HZ,
        frameSamples,
      });

      setState("recording");
    } catch (error) {
      if (sessionRef.current !== session) return;
      console.error(error);
      setErrorMessage(getErrorMessage(error));
      await stopRecording("error");
    } finally {
      // A cancelled setup only releases its own resources, never a newer session's.
      if (sessionRef.current !== session) {
        localWorker?.terminate();
        localStream?.getTracks().forEach((track) => track.stop());
        if (streamRef.current === localStream) streamRef.current = null;
        if (localAudioContext && localAudioContext.state !== "closed") {
          await localAudioContext.close().catch(console.error);
        }
      }
    }
  }

  useEffect(() => {
    return () => {
      sessionRef.current = null;
      cancelWorkerSetupRef.current?.();
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
