<script setup lang="ts">
import { z } from 'zod';
import type { Application, BatchResult, NewApplication, Source } from '~/types/items';
import { SOURCES, STATUSES } from '~/types/items';

const props = defineProps<{ application?: Application }>();
const open = defineModel<boolean>('open', { required: true });
const emit = defineEmits<{ saved: [] }>();

const { t } = useI18n();
const api = useApi();
const toast = useToast();
const loading = ref(false);

const schema = computed(() => {
  const required = z.string().trim().min(1, t('form.required'));
  return z
    .object({
      company: required,
      location: required,
      website: z.union([z.literal(''), z.url(t('form.invalidUrl'))]),
      source: z.enum(['spontaneous', 'job_post', 'recruiter']),
      source_detail: z.string(),
      status: z.enum(['applied', 'responded', 'interview', 'offer', 'rejected']),
      rating: z.number().int().min(0).max(5),
      comment: z.string(),
      applied_at: z.string().min(1, t('form.required')),
    })
    .superRefine((v, ctx) => {
      if (v.source === 'job_post' && !z.url().safeParse(v.source_detail).success)
        ctx.addIssue({ code: 'custom', path: ['source_detail'], message: t('form.invalidUrl') });
      if (v.source === 'recruiter' && !z.email().safeParse(v.source_detail).success)
        ctx.addIssue({ code: 'custom', path: ['source_detail'], message: t('form.invalidEmail') });
    });
});

const today = () => new Date().toISOString().slice(0, 10);
const blank = (): NewApplication & Required<Pick<NewApplication, 'source' | 'status' | 'rating'>> => ({
  company: '',
  location: '',
  website: '',
  source: 'spontaneous',
  source_detail: '',
  status: 'applied',
  rating: 0,
  comment: '',
  applied_at: today(),
});
const state = reactive(blank());

// Reload the form each time the slideover opens, from the edited row or empty.
watch(open, (isOpen) => {
  if (!isOpen) return;
  const a = props.application;
  Object.assign(
    state,
    a
      ? {
          company: a.company ?? '',
          location: a.location ?? '',
          website: a.website ?? '',
          source: a.source ?? 'spontaneous',
          source_detail: a.source_detail ?? '',
          status: a.status ?? 'applied',
          rating: a.rating ?? 0,
          comment: a.comment ?? '',
          applied_at: a.applied_at ?? today(),
        }
      : blank(),
  );
});

// A detail that does not fit the source is dropped.
watch(
  () => state.source,
  (source: Source) => {
    if (source === 'spontaneous') state.source_detail = '';
  },
);

const sourceItems = SOURCES.map((value) => ({ value, label: t(`source.${value}`) }));
const statusItems = STATUSES.map((value) => ({ value, label: t(`status.${value}`) }));

async function submit() {
  loading.value = true;
  try {
    const item = { ...state, source_detail: state.source === 'spontaneous' ? '' : state.source_detail };
    const result = props.application?.id
      ? await api<BatchResult>('/api/applications', {
          method: 'PATCH',
          body: { ops: [{ ids: [props.application.id], set: item }] },
        })
      : await api<BatchResult>('/api/applications', { method: 'POST', body: [item] });
    if (result.failed?.length) {
      for (const f of result.failed) toast.add({ title: t('form.failed', { error: f.error }), color: 'error' });
      return;
    }
    toast.add({ title: t('form.saved'), color: 'success' });
    open.value = false;
    emit('saved');
  } catch {
    // useApi already toasted.
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <USlideover v-model:open="open" :title="application ? t('form.edit') : t('form.add')" :description="t('app.name')">
    <template #body>
      <UForm id="application-form" :schema="schema" :state="state" class="space-y-4" @submit="submit">
        <UFormField name="company" :label="t('form.company')" required>
          <UInput v-model="state.company" class="w-full" />
        </UFormField>
        <UFormField name="location" :label="t('form.location')" required>
          <UInput v-model="state.location" class="w-full" />
        </UFormField>
        <UFormField name="website" :label="t('form.website')">
          <UInput v-model="state.website" type="url" placeholder="https://" class="w-full" />
        </UFormField>
        <UFormField name="source" :label="t('form.source')">
          <USelect v-model="state.source" :items="sourceItems" class="w-full" />
        </UFormField>
        <UFormField
          v-if="state.source !== 'spontaneous'"
          name="source_detail"
          :label="state.source === 'job_post' ? t('form.sourceUrl') : t('form.sourceEmail')"
          required
        >
          <UInput v-model="state.source_detail" :type="state.source === 'job_post' ? 'url' : 'email'" class="w-full" />
        </UFormField>
        <UFormField name="status" :label="t('form.status')">
          <USelect v-model="state.status" :items="statusItems" class="w-full" />
        </UFormField>
        <UFormField name="rating" :label="t('form.rating')">
          <UInputNumber v-model="state.rating" :min="0" :max="5" />
        </UFormField>
        <UFormField name="applied_at" :label="t('form.appliedAt')" required>
          <UInput v-model="state.applied_at" type="date" class="w-full" />
        </UFormField>
        <UFormField name="comment" :label="t('form.comment')">
          <UTextarea v-model="state.comment" :rows="4" class="w-full" />
        </UFormField>
      </UForm>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="outline" :label="t('common.cancel')" @click="open = false" />
        <UButton type="submit" form="application-form" :loading="loading" :label="t('common.save')" />
      </div>
    </template>
  </USlideover>
</template>
