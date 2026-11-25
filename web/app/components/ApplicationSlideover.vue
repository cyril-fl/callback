<script setup lang="ts">
import { parseDate, today as todayDate, getLocalTimeZone, type CalendarDate } from '@internationalized/date';
import { z } from 'zod';
import type { Application, BatchResult, NewApplication } from '~/types/items';
import { STATUSES } from '~/types/items';

const props = defineProps<{ application?: Application }>();
const open = defineModel<boolean>('open', { required: true });
const emit = defineEmits<{ saved: [] }>();

const { t } = useI18n();
const api = useApi();
const toast = useToast();
const sources = useSources();
const loading = ref(false);

const schema = computed(() => {
  const required = z.string().trim().min(1, t('form.required'));
  return z
    .object({
      company: required,
      location: required,
      street: z.string(),
      postal_code: z.string(),
      country: z.string(),
      website: z.union([z.literal(''), z.url(), z.email()], { error: t('form.invalidWebsite') }),
      source: z.string().trim().min(1, t('form.required')).max(60),
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

const today = () => todayDate(getLocalTimeZone()).toString();
type State = Required<Omit<NewApplication, 'status'>> & { status: NonNullable<NewApplication['status']> };
const blank = (): State => ({
  company: '',
  location: '',
  street: '',
  postal_code: '',
  country: '',
  website: '',
  source: 'spontaneous',
  source_detail: '',
  status: 'applied',
  rating: 0,
  comment: '',
  applied_at: today(),
});
const state = reactive(blank());
// The street and postal code stay folded until asked for, or filled.
const fullAddress = ref(false);

// Reload the form each time the slideover opens, from the edited row or empty.
watch(open, (isOpen) => {
  if (!isOpen) return;
  const a = props.application;
  const next = blank();
  if (a) for (const key of Object.keys(next) as (keyof State)[]) Object.assign(next, { [key]: a[key] ?? next[key] });
  Object.assign(state, next);
  fullAddress.value = !!(state.street || state.postal_code);
  sources.refresh();
});

// The date picker works on a CalendarDate, the API on "2026-01-31".
const appliedAt = computed<CalendarDate | undefined>({
  get: () => (state.applied_at ? parseDate(state.applied_at) : undefined),
  set: (value) => (state.applied_at = value ? value.toString() : ''),
});

// A source typed in the combobox that is not offered yet becomes one.
const sourceItems = ref(sources.items.value);
watch(sources.items, (items) => (sourceItems.value = items));
function createSource(name: string) {
  const value = name.trim();
  if (!value) return;
  if (!sourceItems.value.some((item) => item.value === value)) sourceItems.value.push({ value, label: value });
  state.source = value;
}

const detail = computed(() => {
  switch (state.source) {
    case 'spontaneous':
      return undefined;
    case 'job_post':
      return { label: t('form.sourceUrl'), type: 'url', required: true, placeholder: 'https://' };
    case 'recruiter':
      return { label: t('form.sourceEmail'), type: 'email', required: true, placeholder: '' };
    default:
      return { label: t('form.sourceOther'), type: 'text', required: false, placeholder: t('form.sourceOtherHint') };
  }
});

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
  <USlideover
    v-model:open="open"
    :title="application ? t('form.edit') : t('form.add')"
    :description="application ? t('form.edit') : t('form.add')"
    :ui="{ description: 'sr-only' }"
  >
    <template #body>
      <UForm id="application-form" :schema="schema" :state="state" class="space-y-4" @submit="submit">
        <UFormField name="company" :label="t('form.company')" required>
          <UInput v-model="state.company" class="w-full" />
        </UFormField>
        <UFormField name="website" :label="t('form.website')">
          <UInput v-model="state.website" placeholder="https://" class="w-full" />
        </UFormField>

        <div class="grid grid-cols-2 gap-3">
          <UFormField name="location" :label="t('form.city')" required>
            <UInput v-model="state.location" class="w-full" />
          </UFormField>
          <UFormField name="country" :label="t('form.country')">
            <UInput v-model="state.country" class="w-full" />
          </UFormField>
        </div>
        <div v-if="fullAddress" class="grid grid-cols-3 gap-3">
          <UFormField name="street" :label="t('form.street')" class="col-span-2">
            <UInput v-model="state.street" class="w-full" />
          </UFormField>
          <UFormField name="postal_code" :label="t('form.postalCode')">
            <UInput v-model="state.postal_code" class="w-full" />
          </UFormField>
        </div>
        <UButton
          v-else
          variant="link"
          color="neutral"
          size="sm"
          icon="i-lucide-map-pin-plus"
          :label="t('form.fullAddress')"
          class="-mt-2 px-0"
          @click="fullAddress = true"
        />

        <UFormField name="source" :label="t('form.source')">
          <UInputMenu
            v-model="state.source"
            :items="sourceItems"
            value-key="value"
            create-item
            class="w-full"
            @create="createSource"
          />
        </UFormField>
        <UFormField v-if="detail" name="source_detail" :label="detail.label" :required="detail.required">
          <UInput v-model="state.source_detail" :type="detail.type" :placeholder="detail.placeholder" class="w-full" />
        </UFormField>

        <div class="grid grid-cols-2 gap-3">
          <UFormField name="status" :label="t('form.status')">
            <USelect v-model="state.status" :items="statusItems" class="w-full" />
          </UFormField>
          <UFormField name="applied_at" :label="t('form.appliedAt')" required>
            <UInputDate v-model="appliedAt" class="w-full">
              <template #trailing>
                <UPopover>
                  <UButton
                    color="neutral"
                    variant="link"
                    size="sm"
                    icon="i-lucide-calendar"
                    :aria-label="t('form.pickDate')"
                    class="px-0"
                  />
                  <template #content>
                    <UCalendar v-model="appliedAt" class="p-2" />
                  </template>
                </UPopover>
              </template>
            </UInputDate>
          </UFormField>
        </div>

        <UFormField name="rating" :label="t('form.rating')">
          <UInputRating v-model="state.rating" clearable size="lg" />
        </UFormField>
        <UFormField name="comment" :label="t('form.comment')">
          <UTextarea v-model="state.comment" :rows="4" autoresize class="w-full" />
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
