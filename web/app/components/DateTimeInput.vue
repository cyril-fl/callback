<script setup lang="ts">
import { parseDateTime, type DateValue } from '@internationalized/date';

// A date and time picked with Nuxt UI, held as "2026-01-31T14:30" (the datetime-local format the lists store).
const model = defineModel<string>({ default: '' });
const { t } = useI18n();
const { icons } = useIcons();

const value = computed<DateValue | undefined>({
  get: () => (model.value ? parseDateTime(model.value) : undefined),
  set: (v) => {
    const s = v ? v.toString().slice(0, 16) : '';
    // The calendar of an empty field gives a bare date: keep the time already typed, midnight otherwise.
    model.value = s.length === 10 ? `${s}T${model.value.slice(11, 16) || '00:00'}` : s;
  },
});
</script>

<template>
  <UInputDate v-model="value" granularity="minute">
    <template #trailing>
      <UPopover>
        <UButton
          color="neutral"
          variant="link"
          size="sm"
          :icon="icons.calendar"
          :aria-label="t('form.pickDate')"
          class="px-0"
        />
        <template #content>
          <UCalendar v-model="value" class="p-2" />
        </template>
      </UPopover>
    </template>
  </UInputDate>
</template>
