import { API_BASE_URL } from "./api-config"
import "./wasm/wasm_exec.js"
import { BATCH_DURATION_SECONDS as totalBatchTime, MAX_RECORDING_SECONDS as totalTime, HASH_HISTORY_BATCHES as batchWindow } from "./recording-config"
import type { GeneratedHash, WorkerMatch as Match, WorkerMessage } from "./types"

declare const Go: new () => {
  importObject: WebAssembly.Imports
  run(instance: WebAssembly.Instance): Promise<void>
}

declare function samplesToHashes(samples: Uint8Array, timeOffset: number): string
declare function getFingerprintConfig(): { sampleRate: number }
declare function secondsToWindows(seconds: number): number

let ready = false;
let windowsPerBatch = 0;
let samplesPerBatch = 0;

async function initializeWasm() {
  const go = new Go();
  const result = await WebAssembly.instantiateStreaming(fetch("../wasm/main.wasm"), go.importObject)
  go.run(result.instance).catch((err: unknown) => {
    sendError(`Fingerprint engine stopped: ${getErrorMessage(err)}`)
  });
  const config = getFingerprintConfig()
  samplesPerBatch = totalBatchTime * config.sampleRate
  windowsPerBatch = secondsToWindows(totalBatchTime)
  ready = true;
  const message: WorkerMessage = { type: "ready", sampleRate: config.sampleRate }
  postMessage(message)
}

initializeWasm().catch((err: unknown) => {
  sendError(`Could not load the fingerprint engine: ${getErrorMessage(err)}`)
});

const masterHashList: GeneratedHash[][] = [];
let buffer: number[] = []
let chunkIndex = 0;
let foundSong = false;
let processing = false;

function getErrorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

function sendError(message: string) {
  if (foundSong) return
  foundSong = true
  const workerMessage: WorkerMessage = { type: "error", message }
  postMessage(workerMessage)
}

onmessage = (e) => {
  if (foundSong) return
  if (chunkIndex * totalBatchTime >= totalTime) {
    return
  }

  buffer.push(...new Uint8Array(e.data));
  if (!ready) return;
  processBufferedBatches()
}

async function processBufferedBatches() {
  if (processing) return
  processing = true;

  try {
    //multiply by 2 since 2 pcm values will give one float sample later
    const bytesPerBatch = samplesPerBatch * 2;
    const totalBatches = totalTime / totalBatchTime;
    while (!foundSong && chunkIndex < totalBatches && buffer.length >= bytesPerBatch) {
      const chunkBytes: number[] = buffer.slice(0, bytesPerBatch);
      buffer = buffer.slice(bytesPerBatch);

      const timeOffset: number = chunkIndex * windowsPerBatch;
      chunkIndex++;

      const res = JSON.parse(samplesToHashes(new Uint8Array(chunkBytes), timeOffset))
      //console.log(res)
      if (res.error != null) {
        sendError(`Fingerprint generation failed: ${res.error}`)
        return
      }

      masterHashList.push(res.hashes)
      const hashBatch: GeneratedHash[] = [];

      let start = Math.max(0, masterHashList.length - batchWindow)
      for (; start < masterHashList.length; start++) {
        hashBatch.push(...(masterHashList[start]))
      }

      const response = await fetch(`${API_BASE_URL}/song`, {
        method: "POST",
        body: JSON.stringify(hashBatch)
      })
      if (!response.ok) {
        const details = (await response.text()).trim()
        throw new Error(details || `Matching request failed (${response.status})`)
      }
      const song: Match = await response.json()

      if (song.match || chunkIndex === totalBatches) {
        foundSong = true;
        const workerMessage: WorkerMessage = { type: "result", result: song }
        postMessage(workerMessage);
        break
      }
    }
  } catch (err) {
    sendError(getErrorMessage(err))
  } finally {
    processing = false;
  }
}
