class DownsamplePcm16Processor extends AudioWorkletProcessor {
  constructor(options) {
    super();

    const processorOptions = options?.processorOptions ?? {};

    this.targetSampleRate = Number(processorOptions.targetSampleRate);
    this.frameSamples = Number(processorOptions.frameSamples);
    if (!Number.isFinite(this.targetSampleRate) || this.targetSampleRate <= 0 ||
        !Number.isInteger(this.frameSamples) || this.frameSamples <= 0) {
      throw new Error("A valid target sample rate and frame size are required.");
    }


    this.ratio = sampleRate / this.targetSampleRate;

    this.sourceSamples = [];
    this.readPosition = 0;
    this.pendingPcm16 = [];

    this.port.onmessage = (event) => {
      if (event.data?.type === "flush") {
        this.flush();
      }
    };
  }

  flush() {

    this.emitAllPending();
    this.port.postMessage({ type: "flush_complete" });
  }

  clamp(sample) {
    if (sample > 1) return 1;
    if (sample < -1) return -1;
    return sample;
  }

  toPcm16(sample) {
    const normalized = this.clamp(sample);
    return normalized < 0
      ? Math.round(normalized * 0x8000)
      : Math.round(normalized * 0x7fff);
  }

  consumeSourceToPendingPcm16() {
    while (this.readPosition + 1 < this.sourceSamples.length) {
      const index = Math.floor(this.readPosition);
      const frac = this.readPosition - index;
      const left = this.sourceSamples[index] ?? 0;
      const right = this.sourceSamples[index + 1] ?? left;
      const interpolated = left + (right - left) * frac;

      this.pendingPcm16.push(this.toPcm16(interpolated));
      this.readPosition += this.ratio;
    }

    const dropCount = Math.max(0, Math.floor(this.readPosition) - 1);
    if (dropCount > 0) {
      this.sourceSamples.splice(0, dropCount);
      this.readPosition -= dropCount;
    }
  }

  emitFrame(frameSize) {
    if (this.pendingPcm16.length < frameSize) {
      return false;
    }

    const frame = new Int16Array(frameSize);
    for (let i = 0; i < frameSize; i += 1) {
      frame[i] = this.pendingPcm16[i];
    }

    this.pendingPcm16.splice(0, frameSize);
    this.port.postMessage({ type: "pcm16", samples: frame.buffer }, [
      frame.buffer,
    ]);
    return true;
  }

  emitAllPending() {
    while (this.emitFrame(this.frameSamples)) {
    }

    if (this.pendingPcm16.length > 0) {
      const tail = new Int16Array(this.pendingPcm16.length);
      for (let i = 0; i < this.pendingPcm16.length; i += 1) {
        tail[i] = this.pendingPcm16[i];
      }
      this.pendingPcm16.length = 0;
      this.port.postMessage({ type: "pcm16", samples: tail.buffer }, [
        tail.buffer,
      ]);
    }
  }

  process(inputs, outputs) {
    const inputChannels = inputs[0];

    if (inputChannels && inputChannels.length > 0) {
      const frameLength = inputChannels[0].length;
      const channelCount = inputChannels.length;

      for (let i = 0; i < frameLength; i += 1) {
        let sum = 0;
        for (let c = 0; c < channelCount; c += 1) {
          sum += inputChannels[c][i];
        }
        this.sourceSamples.push(sum / channelCount);
      }

      this.consumeSourceToPendingPcm16();

      while (this.emitFrame(this.frameSamples)) {
      }
    }

    const outputChannels = outputs[0];
    if (outputChannels) {
      for (let c = 0; c < outputChannels.length; c += 1) {
        outputChannels[c].fill(0);
      }
    }

    return true;
  }
}

registerProcessor("downsample-pcm16-processor", DownsamplePcm16Processor);
