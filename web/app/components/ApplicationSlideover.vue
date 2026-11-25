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
const places = usePlaces();
const { preferences, pro } = usePreferences();
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
      interview_at: z.string(),
      contact_name: z.string(),
      contact_email: z.union([z.literal(''), z.email()], { error: t('form.invalidEmail') }),
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
  interview_at: '',
  contact_name: '',
  contact_email: '',
});
const state = reactive(blank());
// The street and postal code, and the contact, stay folded until asked for, or filled.
const fullAddress = ref(false);
const contact = ref(false);

// Reload the form each time the slideover opens, from the edited row or empty.
watch(open, (isOpen) => {
  if (!isOpen) return;
  const a = props.application;
  // A new application starts at the default city and country of the settings.
  const next = a
    ? blank()
    : {
        ...blank(),
        location: preferences.value?.default_location ?? '',
        country: preferences.value?.default_country ?? '',
      };
  if (a) for (const key of Object.keys(next) as (keyof State)[]) Object.assign(next, { [key]: a[key] ?? next[key] });
  Object.assign(state, next);
  fullAddress.value = !!(state.street || state.postal_code);
  contact.value = !!(state.contact_name || state.contact_email);
  sources.refresh();
  places.refresh();
});

// The date picker works on a CalendarDate, the API on "2026-01-31".
const appliedAt = computed<CalendarDate | undefined>({
  get: () => (state.applied_at ? parseDate(state.applied_at) : undefined),
  set: (value) => (state.applied_at = value ? value.toString() : ''),
});

// A source typed in the combobox that is not offered yet becomes one.
const sourceItems = ref(sources.items.value);
// Typed and left without picking "Create", it is taken all the same.
const sourceSearch = ref('');
watch(sources.items, (items) => (sourceItems.value = items));
function createSource(name: string) {
  const value = name.trim();
  if (!value) return;
  if (!sourceItems.value.some((item) => item.value === value)) sourceItems.value.push({ value, label: value });
  state.source = value;
}

// Picking a known company fills the website and the address left empty from its last application.
const previous = computed(() => places.lastTo(state.company, props.application?.id));
function pickCompany(company: string) {
  state.company = company;
  const last = places.lastTo(company, props.application?.id);
  if (!last) return;
  for (const key of ['website', 'location', 'street', 'postal_code', 'country'] as const)
    state[key] ||= last[key] ?? '';
  fullAddress.value ||= !!(state.street || state.postal_code);
}

// Picking a known city fills its country if empty.
function pickCity(city: string) {
  state.location = city.trim();
  state.country ||= places.countryOf(state.location) ?? '';
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

const day = (v: string) => new Date(`${v}T00:00:00`).toLocaleDateString('fr-FR');

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
        <UFormField
          name="company"
          :label="t('form.company')"
          :help="previous?.applied_at ? t('form.appliedBefore', { date: day(previous.applied_at) }) : undefined"
          required
        >
          <SuggestInput
            :model-value="state.company"
            :suggestions="places.companies.value"
            class="w-full"
            @update:model-value="pickCompany"
          />
        </UFormField>
        <UFormField name="website" :label="t('form.website')">
          <UInput v-model="state.website" placeholder="https://" class="w-full" />
        </UFormField>

        <div class="grid grid-cols-2 gap-3">
          <UFormField name="location" :label="t('form.city')" required>
            <SuggestInput
              :model-value="state.location"
              :suggestions="places.cities.value"
              class="w-full"
              @update:model-value="pickCity"
            />
          </UFormField>
          <UFormField name="country" :label="t('form.country')">
            <SuggestInput v-model="state.country" :suggestions="places.countries.value" class="w-full" />
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
            v-model:search-term="sourceSearch"
            :items="sourceItems"
            value-key="value"
            class="w-full"
            :ui="{ content: 'has-data-[slot=empty]:hidden' }"
            @blur="createSource(sourceSearch)"
            @keydown.enter="createSource(sourceSearch)"
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

        <UFormField v-if="state.status === 'interview'" name="interview_at" :label="t('form.interviewAt')">
          <UInput v-model="state.interview_at" type="datetime-local" class="w-full" />
        </UFormField>

        <template v-if="pro">
          <div v-if="contact" class="grid grid-cols-2 gap-3">
            <UFormField name="contact_name" :label="t('form.contactName')">
              <UInput v-model="state.contact_name" class="w-full" />
            </UFormField>
            <UFormField name="contact_email" :label="t('form.contactEmail')">
              <UInput v-model="state.contact_email" type="email" class="w-full" />
            </UFormField>
          </div>
          <UButton
            v-else
            variant="link"
            color="neutral"
            size="sm"
            icon="i-lucide-user-plus"
            :label="t('form.contact')"
            class="-mt-2 px-0"
            @click="contact = true"
          />
        </template>

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
