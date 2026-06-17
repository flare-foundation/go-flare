'use client';

let audioContext: AudioContext | null = null;

function getAudioContext(): AudioContext {
  if (!audioContext) {
    audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();
  }
  return audioContext;
}

function createOscillator(
  ctx: AudioContext,
  frequency: number,
  type: OscillatorType = 'sine',
  duration: number = 0.15,
  startTime: number = 0,
  volume: number = 0.3
): void {
  const oscillator = ctx.createOscillator();
  const gainNode = ctx.createGain();

  oscillator.connect(gainNode);
  gainNode.connect(ctx.destination);

  oscillator.type = type;
  oscillator.frequency.setValueAtTime(frequency, ctx.currentTime + startTime);

  gainNode.gain.setValueAtTime(0, ctx.currentTime + startTime);
  gainNode.gain.linearRampToValueAtTime(volume, ctx.currentTime + startTime + 0.01);
  gainNode.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + startTime + duration);

  oscillator.start(ctx.currentTime + startTime);
  oscillator.stop(ctx.currentTime + startTime + duration);
}

// Soft click for piece select
export function playSelect(): void {
  try {
    const ctx = getAudioContext();
    createOscillator(ctx, 880, 'sine', 0.08, 0, 0.15);
    createOscillator(ctx, 1100, 'sine', 0.06, 0.02, 0.08);
  } catch (e) {
    console.warn('Audio playback failed:', e);
  }
}

// Whoosh for piece movement
export function playMove(): void {
  try {
    const ctx = getAudioContext();
    const bufferSize = ctx.sampleRate * 0.1;
    const buffer = ctx.createBuffer(1, bufferSize, ctx.sampleRate);
    const data = buffer.getChannelData(0);
    for (let i = 0; i < bufferSize; i++) {
      data[i] = (Math.random() * 2 - 1) * (1 - i / bufferSize);
    }
    const source = ctx.createBufferSource();
    source.buffer = buffer;
    
    const filter = ctx.createBiquadFilter();
    filter.type = 'bandpass';
    filter.frequency.setValueAtTime(800, ctx.currentTime);
    filter.frequency.exponentialRampToValueAtTime(200, ctx.currentTime + 0.1);
    filter.Q.value = 0.5;
    
    const gain = ctx.createGain();
    gain.gain.setValueAtTime(0.25, ctx.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.1);
    
    source.connect(filter);
    filter.connect(gain);
    gain.connect(ctx.destination);
    source.start();
  } catch (e) {
    console.warn('Audio playback failed:', e);
  }
}

// Heavier thud for captures
export function playCapture(): void {
  try {
    const ctx = getAudioContext();
    createOscillator(ctx, 160, 'square', 0.12, 0, 0.25);
    createOscillator(ctx, 80, 'sawtooth', 0.15, 0, 0.2);
  } catch (e) {
    console.warn('Audio playback failed:', e);
  }
}

// Subtle warning for check
export function playCheck(): void {
  try {
    const ctx = getAudioContext();
    createOscillator(ctx, 440, 'sine', 0.1, 0, 0.2);
    createOscillator(ctx, 550, 'sine', 0.12, 0.05, 0.2);
    createOscillator(ctx, 660, 'sine', 0.08, 0.1, 0.15);
  } catch (e) {
    console.warn('Audio playback failed:', e);
  }
}

// Triumphant chime for checkmate
export function playCheckmate(): void {
  try {
    const ctx = getAudioContext();
    // Rising arpeggio
    const notes = [261.63, 329.63, 392.00, 523.25, 659.25]; // C4 E4 G4 C5 E5
    notes.forEach((freq, i) => {
      createOscillator(ctx, freq, 'sine', 0.5, i * 0.12, 0.3);
      // Add harmonic
      createOscillator(ctx, freq * 2, 'sine', 0.3, i * 0.12, 0.4);
    });
  } catch (e) {
    console.warn('Audio playback failed:', e);
  }
}

export function resumeAudioContext(): void {
  try {
    const ctx = getAudioContext();
    if (ctx.state === 'suspended') {
      ctx.resume();
    }
  } catch (e) {
    console.warn('Could not resume audio context:', e);
  }
}
