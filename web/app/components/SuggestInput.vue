<script setup lang="ts">
// A text field that offers values already used, and takes any other typed; cleared, it is ''.
const props = defineProps<{ suggestions: string[] }>();
const model = defineModel<string>({ required: true });

const items = computed(() =>
  model.value && !props.suggestions.includes(model.value) ? [model.value, ...props.suggestions] : props.suggestions,
);
</script>

<template>
  <UInputMenu
    :model-value="model"
    :items="items"
    create-item
    @update:model-value="(v: string | null) => (model = v ?? '')"
    @create="(v: string) => (model = v.trim())"
  />
</template>
