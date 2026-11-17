class MessageSoundManager {
  private audioContext: AudioContext | null = null;
  private readonly cooldownMs = 1000;
  private lastPlayedAt = 0;
  private hasUnlocked = false;

  public installUnlockOnInteraction() {
    const unlock = () => {
      void this.unlock();
    };

    window.addEventListener('pointerdown', unlock, { once: true, capture: true });
    window.addEventListener('keydown', unlock, { once: true, capture: true });
    window.addEventListener('touchstart', unlock, { once: true, capture: true });
  }

  public async unlock() {
    const context = this.getAudioContext();
    if (!context) {
      return;
    }

    if (context.state === 'suspended') {
      try {
        await context.resume();
      } catch {
        // Ignore unlock failures and allow later retries.
      }
    }

    if (!this.hasUnlocked && context.state === 'running') {
      this.hasUnlocked = true;
      this.prime(context);
    }
  }

  public async play() {
    const now = Date.now();
    if (now - this.lastPlayedAt < this.cooldownMs) {
      return;
    }

    const context = this.getAudioContext();
    if (!context) {
      return;
    }

    if (context.state === 'suspended') {
      try {
        await context.resume();
      } catch {
        return;
      }
    }

    this.lastPlayedAt = now;

    const startAt = context.currentTime + 0.01;
    this.scheduleTone(context, startAt, 740, 0.1, 0.065);
    this.scheduleTone(context, startAt + 0.12, 988, 0.14, 0.055);
  }

  private getAudioContext(): AudioContext | null {
    if (typeof window === 'undefined') {
      return null;
    }

    if (!this.audioContext) {
      const AudioContextCtor = window.AudioContext
        || (window as Window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
      if (!AudioContextCtor) {
        return null;
      }
      this.audioContext = new AudioContextCtor();
    }

    return this.audioContext;
  }

  private scheduleTone(context: AudioContext, startAt: number, frequency: number, duration: number, volume: number) {
    const oscillator = context.createOscillator();
    const gainNode = context.createGain();

    oscillator.type = 'sine';
    oscillator.frequency.setValueAtTime(frequency, startAt);

    gainNode.gain.setValueAtTime(0.0001, startAt);
    gainNode.gain.exponentialRampToValueAtTime(volume, startAt + 0.012);
    gainNode.gain.exponentialRampToValueAtTime(0.0001, startAt + duration);

    oscillator.connect(gainNode);
    gainNode.connect(context.destination);

    oscillator.start(startAt);
    oscillator.stop(startAt + duration + 0.02);
  }

  private prime(context: AudioContext) {
    const oscillator = context.createOscillator();
    const gainNode = context.createGain();

    oscillator.type = 'sine';
    gainNode.gain.setValueAtTime(0.00001, context.currentTime);

    oscillator.connect(gainNode);
    gainNode.connect(context.destination);

    oscillator.start(context.currentTime);
    oscillator.stop(context.currentTime + 0.01);
  }
}

const messageSound = new MessageSoundManager();

export default messageSound;
