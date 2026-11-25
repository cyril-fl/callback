<script setup lang="ts">
defineProps<{ title: string; message: string; confirmLabel: string }>();
const open = defineModel<boolean>('open', { required: true });
const emit = defineEmits<{ confirm: [] }>();
const { t } = useI18n();
</script>

<template>
  <UModal v-model:open="open" :title="title" :description="message">
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="outline" :label="t('common.cancel')" @click="open = false" />
        <UButton
          color="error"
          :label="confirmLabel"
          @click="
            emit('confirm');
            open = false;
          "
        />
      </div>
    </template>
  </UModal>
</template>
