<script setup lang="ts">
import { parseDate, today as todayDate, getLocalTimeZone, type CalendarDate } from '@internationalized/date';
import { z } from 'zod';
import type { Application, BatchResult, Contact, Interview, NewApplication } from '~/types/items';
import { STATUSES } from '~/types/items';
import type { FormPart } from '~/utils/workflow';

const props = defineProps<{ application?: Application }>();
const open = defineModel<boolean>('open', { required: true });
const emit = defineEmits<{ saved: [app: Application] }>();

const { t } = useI18n();
const { icons } = useIcons();
const api = useApi();
const toast = useToast();
const sources = useSources();
const places = usePlaces();
const countries = useCountries();
const { preferences, pro } = usePreferences();
const loading = ref(false);

const schema = computed(() => {
  const required = z.string().trim().min(1, t('form.required'));
  return z
    .object({
      company: required,
      title: z.string(),
      location: required,
      street: z.string(),
      postal_code: z.string(),
      country: z.string(),
      // In the pro plan an e-mail is a contact: the site is a URL.
      website: pro.value
        ? z.union([z.literal(''), z.url()], { error: t('form.invalidUrl') })
        : z.union([z.literal(''), z.url(), z.email()], { error: t('form.invalidWebsite') }),
      source: z.string().trim().min(1, t('form.required')).max(60),
      source_detail: z.string(),
      status: z.enum(['applied', 'responded', 'interview', 'offer', 'rejected', 'no_reply']),
      rating: z.number().int().min(0).max(5),
      comment: z.string(),
      applied_at: z.string().min(1, t('form.required')),
      interview_at: z.string(),
      contact_name: z.string(),
      contact_email: z.union([z.literal(''), z.email()], { error: t('form.invalidEmail') }),
      interviews: z
        .array(z.object({ at: z.string().min(1, t('form.required')), kind: z.string(), note: z.string().optional() }))
        // From the interview step on, the pro plan keeps at least one.
        .min(pro.value && requiresInterview(state.status) ? 1 : 0, t('form.interviewRequired')),
      contacts: z.array(
        z
          .object({
            name: z.string().optional(),
            email: z.union([z.literal(''), z.email()], { error: t('form.invalidEmail') }).optional(),
            role: z.string().optional(),
          })
          .refine((c) => c.name?.trim() || c.email?.trim(), { message: t('form.contactRequired'), path: ['name'] }),
      ),
    })
    .superRefine((v, ctx) => {
      if (v.source === 'job_post' && !z.url().safeParse(v.source_detail).success)
        ctx.addIssue({ code: 'custom', path: ['source_detail'], message: t('form.invalidUrl') });
      if (v.source === 'recruiter' && pro.value && !recruiterEmail(v.contacts))
        ctx.addIssue({ code: 'custom', path: ['contacts'], message: t('form.recruiterMissing') });
      if (v.source === 'recruiter' && !pro.value && !z.email().safeParse(v.source_detail).success)
        ctx.addIssue({ code: 'custom', path: ['source_detail'], message: t('form.invalidEmail') });
    });
});

const today = () => todayDate(getLocalTimeZone()).toString();
type State = Required<Omit<NewApplication, 'status'>> & { status: NonNullable<NewApplication['status']> };
const blank = (): State => ({
  company: '',
  title: '',
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
  interviews: [],
  contacts: [],
});
const state = reactive(blank());
let saved = new WeakSet<Interview>();
// The form as loaded: closing it changed asks before the changes are lost.
const initial = ref('');
const dirty = computed(() => JSON.stringify(state) !== initial.value);
const discardOpen = ref(false);
function requestClose() {
  if (dirty.value) discardOpen.value = true;
  else open.value = false;
}
// The street and postal code stay folded until asked for, or filled.
const fullAddress = ref(false);

// Reload the form each time the slideover opens, from the edited row or empty.
watch(open, async (isOpen) => {
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
  // Copies: the rows edited here are not the list's until saved.
  next.interviews = next.interviews.map((i) => ({ ...i }));
  // A saved interview happened: it cannot be deleted, only a row added here can.
  saved = new WeakSet(next.interviews);
  next.contacts = next.contacts.map((c) => ({ ...c }));
  // The pro plan keeps the e-mails as contacts: the recruiter's, and the one written as the site.
  if (pro.value) {
    if (next.source === 'recruiter') next.contacts = withContact(next.contacts, next.source_detail, RECRUITER_ROLE);
    if (next.website && !next.website.includes('://') && next.website.includes('@')) {
      next.contacts = withContact(next.contacts, next.website);
      next.website = '';
    }
  }
  Object.assign(state, next);
  fullAddress.value = !!(state.street || state.postal_code);
  sources.refresh();
  places.refresh();
  // After the watchers the load sets off (the recruiter row).
  await nextTick();
  initial.value = JSON.stringify(state);
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

// Picking the recruiter source in the pro plan asks for the recruiter among the contacts.
watch(
  () => state.source,
  (source) => {
    if (pro.value && source === 'recruiter') state.contacts = withRecruiterRow(state.contacts);
  },
);

const detail = computed(() => {
  switch (state.source) {
    case 'spontaneous':
      return undefined;
    case 'job_post':
      return { label: t('form.sourceUrl'), type: 'url', required: true, placeholder: 'https://' };
    case 'recruiter':
      if (pro.value) return undefined;
      return { label: t('form.sourceEmail'), type: 'email', required: true, placeholder: '' };
    default:
      return { label: t('form.sourceOther'), type: 'text', required: false, placeholder: t('form.sourceOtherHint') };
  }
});

const day = (v: string) => new Date(`${v}T00:00:00`).toLocaleDateString('fr-FR');

const interviewColumns = useInterviewColumns();
const contactColumns = [
  { key: 'name', label: t('form.name') },
  { key: 'email', label: t('form.email'), type: 'email' },
  { key: 'role', label: t('form.role') },
] as const;

const statusItems = computed(() =>
  STATUSES.map((value) => ({
    value,
    label: t(`status.${value}`),
  })),
);
const shows = (part: FormPart, filled = false) => showsPart(state.status, part, filled);

async function submit() {
  loading.value = true;
  try {
    // The pro plan writes the lists, the base plan the one interview it shows: interview_at edits the last one.
    const { interviews, contacts, interview_at, contact_name, contact_email, ...rest } = state;
    const item = {
      ...rest,
      ...(pro.value ? { interviews, contacts } : { interview_at, contact_name, contact_email }),
      source_detail:
        state.source === 'spontaneous'
          ? ''
          : pro.value && state.source === 'recruiter'
            ? (recruiterEmail(contacts) ?? '')
            : state.source_detail,
    };
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
    emit('saved', { ...item, id: props.application?.id ?? result.ids?.[0] ?? '' } as Application);
  } catch {
    // useApi already toasted.
  } finally {
    loading.value = false;
  }
}

const form = useTemplateRef('form');
const { key } = useShortcuts();
defineShortcuts({
  [key('save')]: { usingInput: true, handler: () => open.value && !loading.value && form.value?.submit() },
});

// On a phone the form is a sheet rising from the bottom.
const phone = useMediaQuery('(max-width: 639px)');
</script>

<template>
  <USlideover
    :open="open"
    :title="application ? t('form.edit') : t('form.add')"
    :description="application ? t('form.edit') : t('form.add')"
    :side="phone ? 'bottom' : 'right'"
    :ui="{ description: 'sr-only', content: phone ? 'max-h-[90dvh] rounded-t-2xl' : pro ? 'sm:max-w-xl' : undefined }"
    @update:open="(v) => (v ? (open = true) : requestClose())"
  >
    <template #body>
      <UForm id="application-form" ref="form" :schema="schema" :state="state" class="space-y-8" @submit="submit">
        <fieldset class="space-y-4">
          <legend
            class="mb-3 w-full border-b border-default pb-1 text-xs font-semibold tracking-wide text-muted uppercase"
          >
            {{ t('form.sections.company') }}
          </legend>
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
          <UFormField name="title" :label="t('form.title')">
            <UInput v-model="state.title" class="w-full" />
          </UFormField>
          <UFormField name="website" :label="pro ? t('form.site') : t('form.website')">
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
              <USelectMenu
                v-model="state.country"
                :items="countries.items.value"
                value-key="value"
                :placeholder="t('form.country')"
                clear
                class="w-full"
              />
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
            :icon="icons.fullAddress"
            :label="t('form.fullAddress')"
            class="-mt-2 px-0"
            @click="fullAddress = true"
          />
        </fieldset>
        <fieldset class="space-y-4">
          <legend
            class="mb-3 w-full border-b border-default pb-1 text-xs font-semibold tracking-wide text-muted uppercase"
          >
            {{ t('form.sections.application') }}
          </legend>
          <div class="grid grid-cols-2 gap-3">
            <UFormField name="applied_at" :label="t('form.appliedAt')" required>
              <UPopover>
                <UButton
                  color="neutral"
                  variant="outline"
                  :icon="icons.calendar"
                  :label="state.applied_at ? day(state.applied_at) : t('form.pickDate')"
                  class="w-full"
                />
                <template #content>
                  <UCalendar v-model="appliedAt" class="p-2" />
                </template>
              </UPopover>
            </UFormField>
            <UFormField name="status" :label="t('form.status')">
              <USelect v-model="state.status" :items="statusItems" class="w-full" />
            </UFormField>
          </div>
          <UFormField name="source" :label="t('form.source')">
            <UInputMenu
              v-model="state.source"
              v-model:search-term="sourceSearch"
              :items="sourceItems"
              value-key="value"
              class="w-full"
              :ui="{ content: 'has-data-[slot=empty]:hidden!' }"
              @blur="createSource(sourceSearch)"
              @keydown.enter="noneHighlighted($event) && createSource(sourceSearch)"
            />
          </UFormField>
          <UFormField v-if="detail" name="source_detail" :label="detail.label" :required="detail.required">
            <UInput
              v-model="state.source_detail"
              :type="detail.type"
              :placeholder="detail.placeholder"
              class="w-full"
            />
          </UFormField>
        </fieldset>
        <fieldset v-if="pro || shows('interviews', !!state.interview_at)" class="space-y-4">
          <legend
            class="mb-3 w-full border-b border-default pb-1 text-xs font-semibold tracking-wide text-muted uppercase"
          >
            {{ t('form.sections.followUp') }}
          </legend>
          <UFormField
            v-if="!pro && shows('interviews', !!state.interview_at)"
            name="interview_at"
            :label="t('form.interviewAt')"
          >
            <DateTimeInput v-model="state.interview_at" class="w-full" />
          </UFormField>

          <template v-if="pro">
            <!-- The parts each status shows: app/utils/workflow.ts. -->
            <ListTable
              v-if="shows('interviews', state.interviews.length > 0)"
              v-model="state.interviews"
              name="interviews"
              :title="t('form.interviews')"
              :hint="state.interviews.length ? undefined : t('form.interviewsHint')"
              :columns="interviewColumns"
              :blank="(): Interview => ({ at: '', kind: 'other', note: '' })"
              :sort="[{ id: 'at', desc: true }]"
              :deletable="(row: Interview) => !saved.has(toRaw(row))"
            />
            <ListTable
              v-if="shows('contacts')"
              v-model="state.contacts"
              name="contacts"
              :title="t('form.contacts')"
              :columns="contactColumns"
              :blank="(): Contact => ({ name: '', email: '', role: '' })"
            />
          </template>
        </fieldset>
        <fieldset class="space-y-4">
          <legend
            class="mb-3 w-full border-b border-default pb-1 text-xs font-semibold tracking-wide text-muted uppercase"
          >
            {{ t('form.sections.notes') }}
          </legend>
          <UFormField name="rating" :label="t('form.rating')">
            <UInputRating
              v-model="state.rating"
              :icon="icons.rating"
              :empty-icon="icons.ratingEmpty"
              clearable
              size="lg"
            />
          </UFormField>
          <UFormField name="comment" :label="t('form.comment')">
            <UTextarea v-model="state.comment" :rows="4" autoresize class="w-full" />
          </UFormField>
        </fieldset>
      </UForm>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="outline" :label="t('common.cancel')" @click="requestClose" />
        <UButton type="submit" form="application-form" :loading="loading" :label="t('common.save')" />
      </div>
      <ConfirmModal
        v-model:open="discardOpen"
        :title="t('form.discardTitle')"
        :message="t('form.discardMessage')"
        :confirm-label="t('form.discard')"
        @confirm="open = false"
      />
    </template>
  </USlideover>
</template>
