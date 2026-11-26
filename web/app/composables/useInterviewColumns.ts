// The interviews table's columns, shared by the form and the interview modal.
export function useInterviewColumns() {
  const { t } = useI18n();
  return [
    { key: 'at', label: t('form.at'), type: 'datetime-local' },
    {
      key: 'kind',
      label: t('form.kind'),
      items: (['hr', 'manager', 'tech', 'other'] as const).map((value) => ({ value, label: t(`form.kinds.${value}`) })),
    },
    { key: 'note', label: t('form.note') },
  ] as const;
}
