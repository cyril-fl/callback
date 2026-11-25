<script setup lang="ts">
// A text field that offers values already used, and takes any other typed; cleared, it is ''.
const props = defineProps<{ suggestions: string[] }>();
const model = defineModel<string>({ required: true });
const search = ref('');

const items = computed(() =>
  model.value && !props.suggestions.includes(model.value) ? [model.value, ...props.suggestions] : props.suggestions,
);

// Leaving the field keeps what was typed, without having to pick "Create" (a pick empties the search first).
function commit() {
  const value = search.value.trim();
  if (value && value !== model.value) model.value = value;
}
</script>

<template>
  <UInputMenu
    v-model:search-term="search"
    :model-value="model"
    :items="items"
    create-item
    @update:model-value="(v: string | null) => (model = v ?? '')"
    @create="(v: string) => (model = v.trim())"
    @blur="commit"
  />
</template>
