declare function importScripts(...urls: string[]): void

declare const Go: new () => {
  importObject: WebAssembly.Imports
  run(instance: WebAssembly.Instance): Promise<void>
}

declare function samplesToHashes(samples: Uint8Array, timeOffset: number): string

importScripts("../wasm/wasm_exec.js")

const go = new Go();
let ready = false;
WebAssembly.instantiateStreaming(fetch("../wasm/main.wasm"), go.importObject).then((result) => {
  go.run(result.instance).catch((err: unknown) => {
    sendError(`Fingerprint engine stopped: ${getErrorMessage(err)}`)
  });
  ready = true;
}).catch((err: unknown) => {
  sendError(`Could not load the fingerprint engine: ${getErrorMessage(err)}`)
});

const samplingRate = 11025
const windowSize = 1024
const hopFactor = 2
const hopSize = windowSize / hopFactor //how much to slide each window by

const totalTime = 40     // record for max of 40 sec
const totalBatchTime = 5 // process in batches of 5 sec
const windowsPerBatch = SecondsToWindows(totalBatchTime)
const samplesPerBatch = totalBatchTime * samplingRate


function SecondsToWindows(seconds : number) : number {
  if (seconds == 0) return 0;
  const secondsPerHop = hopSize / samplingRate
  return Math.ceil(seconds / secondsPerHop)
}

type GeneratedHash = {
  hash: number
  anchorTime: number
}

type Match = {
	songName: string
	artist: string
	match: boolean
	verdict: string
}

type WorkerMessage =
  | { type: "result", result: Match }
  | { type: "error", message: string }

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
    //send could not find song message
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
      const batchWindow = 3; //only look at the last 15 seconds
      const hashBatch: GeneratedHash[] = [];

      let start = Math.max(0, masterHashList.length - batchWindow)
      for (; start < masterHashList.length; start++) {
        hashBatch.push(...(masterHashList[start]))
      }

      const response = await fetch("http://localhost:3000/song", {
        method: "POST",
        body: JSON.stringify(hashBatch)
      })
      if (!response.ok) {
        const details = (await response.text()).trim()
        throw new Error(details || `Matching request failed (${response.status})`)
      }
      const song: Match = await response.json()

      if (song.match) {
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
