<script setup lang="ts">
// expected, when given, must be typed before confirming: for what cannot be undone and loses a lot.
const props = defineProps<{ title: string; message: string; confirmLabel: string; expected?: string }>();
const open = defineModel<boolean>('open', { required: true });
const emit = defineEmits<{ confirm: [] }>();
const { t } = useI18n();
const typed = ref('');
watch(open, () => (typed.value = ''));
</script>

<template>
  <UModal v-model:open="open" :title="title" :description="message">
    <template v-if="expected" #body>
      <UFormField :label="t('common.typeToConfirm', { word: expected })">
        <UInput v-model="typed" autocomplete="off" class="w-full" />
      </UFormField>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="outline" :label="t('common.cancel')" @click="open = false" />
        <UButton
          color="error"
          :label="confirmLabel"
          :disabled="!!props.expected && typed.trim() !== props.expected"
          @click="
            emit('confirm');
            open = false;
          "
        />
      </div>
    </template>
  </UModal>
</template>
