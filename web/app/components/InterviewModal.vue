<script setup lang="ts">
import { z } from 'zod';
import type { Application, Interview, Status } from '~/types/items';

// A move to a status that needs an interview asks only for it: the interviews (pro plan) or the one date (base plan).
const props = defineProps<{ application?: Application; status?: Status }>();
const open = defineModel<boolean>('open', { required: true });
const emit = defineEmits<{ save: [set: Partial<Application>] }>();
const { t } = useI18n();
const { pro } = usePreferences();
const { icons } = useIcons();
const kinds = [...useInterviewColumns()[1].items];
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
    :title="application ? `${application.company} · ${t(`status.${status}`)}` : ''"
    :description="t('form.interviewsHint')"
  >
    <template #body>
      <UForm id="interview-form" :schema="schema" :state="state" @submit="submit">
        <div v-if="pro" class="space-y-3">
          <div v-for="(interview, i) in state.interviews" :key="i" class="space-y-3 rounded-lg p-3 ring-1 ring-default">
            <div class="flex items-end gap-2">
              <UFormField :name="`interviews.${i}.at`" :label="t('form.at')" required class="flex-1">
                <DateTimeInput v-model="interview.at" class="w-full" />
              </UFormField>
              <UFormField :name="`interviews.${i}.kind`" :label="t('form.kind')" class="w-36">
                <USelect v-model="interview.kind" :items="kinds" class="w-full" />
              </UFormField>
              <UButton
                v-if="state.interviews.length > 1"
                :icon="icons.delete"
                color="neutral"
                variant="ghost"
                :aria-label="t('common.delete')"
                @click="state.interviews.splice(i, 1)"
              />
            </div>
            <UFormField :name="`interviews.${i}.note`" :label="t('form.note')">
              <UTextarea v-model="interview.note" :rows="2" autoresize class="w-full" />
            </UFormField>
          </div>
          <UButton
            :icon="icons.add"
            variant="soft"
            color="neutral"
            block
            :label="t('form.addInterview')"
            @click="state.interviews.push(blank())"
          />
        </div>
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
