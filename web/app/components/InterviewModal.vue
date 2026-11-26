<script setup lang="ts">
import { z } from 'zod';
import type { Application, Interview, Status } from '~/types/items';

// A move to a status that needs an interview asks only for it: the interviews (pro plan) or the one date (base plan).
const props = defineProps<{ application?: Application; status?: Status }>();
const open = defineModel<boolean>('open', { required: true });
const emit = defineEmits<{ save: [set: Partial<Application>] }>();
const { t } = useI18n();
const { pro } = usePreferences();
const interviewColumns = useInterviewColumns();
const blank = (): Interview => ({ at: '', kind: 'other', note: '' });

const state = reactive({ interviews: [] as Interview[], interview_at: '' });
watch(open, (isOpen) => {
  if (isOpen) Object.assign(state, { interviews: [blank()], interview_at: '' });
});
const schema = computed(() =>
  pro.value
    ? z.object({
        interviews: z
          .array(z.object({ at: z.string().min(1, t('form.required')), kind: z.string(), note: z.string().optional() }))
          .min(1, t('form.required')),
      })
    : z.object({ interview_at: z.string().min(1, t('form.required')) }),
);

function submit() {
  emit('save', {
    status: props.status,
    ...(pro.value ? { interviews: state.interviews } : { interview_at: state.interview_at }),
  });
  open.value = false;
}
</script>

<template>
  <UModal
    v-model:open="open"
    :title="status ? t(`status.${status}`) : ''"
    :description="t('form.interviewsHint')"
    :ui="{ content: pro ? 'sm:max-w-xl' : undefined }"
  >
    <template #body>
      <UForm id="interview-form" :schema="schema" :state="state" @submit="submit">
        <ListTable
          v-if="pro"
          v-model="state.interviews"
          name="interviews"
          :title="`${application?.company ?? ''} · ${t('form.interviews')}`"
          :columns="interviewColumns"
          :blank="blank"
        />
        <UFormField v-else name="interview_at" :label="t('form.interviewAt')" required>
          <DateTimeInput v-model="state.interview_at" class="w-full" />
        </UFormField>
      </UForm>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="outline" :label="t('common.cancel')" @click="open = false" />
        <UButton type="submit" form="interview-form" :label="t('common.save')" />
      </div>
    </template>
  </UModal>
</template>
