import { computed, onBeforeUnmount, ref } from 'vue';
import { useSingleFlight } from '@/composables/useSingleFlight';

type SendSmsHandler = (phone: string) => Promise<void>;

export function useSmsCode(phoneGetter: () => string, sendSms: SendSmsHandler) {
  const countdown = ref(0);
  const requestFlight = useSingleFlight();
  const sending = requestFlight.running;
  let timer: number | null = null;

  const canSend = computed(() => !sending.value && countdown.value === 0);

  function stopCountdown() {
    if (timer !== null) {
      window.clearInterval(timer);
      timer = null;
    }
  }

  function startCountdown(seconds = 60) {
    stopCountdown();
    countdown.value = seconds;
    timer = window.setInterval(() => {
      if (countdown.value <= 1) {
        countdown.value = 0;
        stopCountdown();
        return;
      }
      countdown.value -= 1;
    }, 1000);
  }

  async function requestCode() {
    if (!canSend.value) {
      return;
    }

    const phone = phoneGetter().trim();
    if (!phone) {
      throw new Error('请先输入手机号');
    }

    await requestFlight.run(async () => {
      await sendSms(phone);
      startCountdown();
    });
  }

  onBeforeUnmount(() => {
    stopCountdown();
  });

  return {
    countdown,
    sending,
    canSend,
    requestCode,
  };
}
