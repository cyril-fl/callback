<script setup lang="ts">
const props = defineProps<{ items: { label: string; count: number }[] }>();
const most = computed(() => Math.max(1, ...props.items.map((i) => i.count)));
</script>

<template>
  <ul class="divide-y divide-default rounded-lg border border-default">
    <li v-for="item in items" :key="item.label" class="flex items-center gap-4 px-4 py-2.5">
      <p class="w-32 shrink-0 truncate text-sm">{{ item.label }}</p>
      <div class="h-2 flex-1 overflow-hidden rounded-full bg-elevated">
        <div class="h-full rounded-full bg-primary" :style="{ width: `${(item.count / most) * 100}%` }" />
      </div>
      <span class="w-8 shrink-0 text-right text-sm text-muted">{{ item.count }}</span>
    </li>
  </ul>
</template>
